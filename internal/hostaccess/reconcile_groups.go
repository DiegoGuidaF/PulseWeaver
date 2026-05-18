package hostaccess

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/slicex"
)

// DesiredHostGroup is the caller-provided shape of a single host group inside
// a reconcile request. A nil ID marks a brand-new group; a non-nil ID must
// match an existing group or the reconcile fails with ErrHostGroupNotFound.
type DesiredHostGroup struct {
	ID          *ids.HostGroupID
	Name        string
	Color       string
	Icon        string
	Description *string
	HostIDs     []ids.KnownHostID
}

// prepare normalises and validates a single desired group: trims the name,
// rejects empty names, and dedups host IDs.
func (g *DesiredHostGroup) prepare() error {
	g.Name = strings.TrimSpace(g.Name)
	g.HostIDs = slicex.Dedup(g.HostIDs)

	if g.Name == "" {
		return ErrGroupNameRequired
	}

	return nil
}

// ReconcileHostGroupsInput is the full desired image of host_groups that the
// caller wants the database to converge to.
type ReconcileHostGroupsInput struct {
	Groups []DesiredHostGroup
}

// prepare normalises every group and rejects requests that reuse the same
// existing group ID twice (which would make the create/update/delete plan
// ambiguous).
func (in *ReconcileHostGroupsInput) prepare() error {
	for i := range in.Groups {
		if err := in.Groups[i].prepare(); err != nil {
			return err
		}
	}

	seenIDs := make(map[ids.HostGroupID]struct{}, len(in.Groups))
	for i := range in.Groups {
		g := in.Groups[i]
		if g.ID != nil {
			if _, ok := seenIDs[*g.ID]; ok {
				return ErrDuplicateGroupID
			}
			seenIDs[*g.ID] = struct{}{}
		}
	}

	return nil
}

// groupReconcilePlan is the ordered set of write operations needed to converge
// the current state to the desired state.
type groupReconcilePlan struct {
	toCreate []HostGroupDraft
	toUpdate []HostGroup
	toDelete []ids.HostGroupID
}

// HostGroupDraft is the minimum shape needed to insert a new host_groups row
// plus its members in a single repository call.
type HostGroupDraft struct {
	Name        string
	Color       string
	Icon        string
	Description *string
	HostIDs     []ids.KnownHostID
}

// ReconcileHostGroups makes the database converge to the desired image of
// host_groups + members in a single transaction. Groups present in `in` with a
// non-nil ID are updated; groups with a nil ID are created; groups currently
// in the database whose ID is absent from `in` are deleted.
//
// All referenced known-host IDs are validated up-front so the transaction can
// fail fast without partial work. Observers are notified once on success
// because group membership changes can shift effective per-user host access.
func (s *Service) ReconcileHostGroups(ctx context.Context, in ReconcileHostGroupsInput) error {
	if err := in.prepare(); err != nil {
		return err
	}

	if err := s.validateReferencedHosts(ctx, in.Groups); err != nil {
		return err
	}

	currentGroups, err := s.repo.ListHostGroups(ctx)
	if err != nil {
		return err
	}

	plan, err := buildGroupReconcilePlan(currentGroups, in.Groups)
	if err != nil {
		return err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Order matters: deletes first frees unique names (idx_host_groups_name)
		// that subsequent updates/creates may want to claim. Updates run before
		// creates so a renamed group can hand its old name off to a new one.
		// NOTE: this still does NOT handle the rename-swap case where two
		// existing groups exchange names — that hits the unique index on the
		// first UPDATE. A two-phase rename-via-temp would be needed; deferred
		// for now since the UI does not currently allow it.
		for _, id := range plan.toDelete {
			if err := s.repo.DeleteHostGroup(ctx, id); err != nil {
				return err
			}
		}

		for _, group := range plan.toUpdate {
			if err := s.repo.UpdateHostGroup(ctx, group); err != nil {
				return err
			}
		}

		for _, draft := range plan.toCreate {
			if _, err := s.repo.CreateHostGroup(ctx, draft); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.notifyUserHostAccessObservers(ctx)
	return nil
}

// buildGroupReconcilePlan diffs the current persisted groups against the
// desired image and produces the create/update/delete buckets. A desired
// group whose ID is unknown returns ErrHostGroupNotFound. A desired group
// whose definition matches the current one is silently skipped.
func buildGroupReconcilePlan(current []HostGroup, desired []DesiredHostGroup) (groupReconcilePlan, error) {
	currentByID := make(map[ids.HostGroupID]HostGroup, len(current))
	for _, g := range current {
		currentByID[g.ID] = g
	}

	desiredIDs := make(map[ids.HostGroupID]struct{})
	var plan groupReconcilePlan

	for _, g := range desired {
		if g.ID == nil {
			plan.toCreate = append(plan.toCreate, HostGroupDraft{
				Name:        g.Name,
				Color:       g.Color,
				Description: g.Description,
				Icon:        g.Icon,
				HostIDs:     g.HostIDs,
			})
			continue
		}

		id := *g.ID
		currentGroup, ok := currentByID[id]
		if !ok {
			return groupReconcilePlan{}, fmt.Errorf("%w: id=%d", ErrHostGroupNotFound, id)
		}
		desiredIDs[id] = struct{}{}

		candidate := HostGroup{
			ID:          id,
			Name:        g.Name,
			Color:       g.Color,
			Description: g.Description,
			Icon:        g.Icon,
			HostIDs:     g.HostIDs,
		}

		if !currentGroup.SameDefinitionAs(candidate) {
			plan.toUpdate = append(plan.toUpdate, candidate)
		}
	}

	for _, g := range current {
		if _, ok := desiredIDs[g.ID]; !ok {
			plan.toDelete = append(plan.toDelete, g.ID)
		}
	}

	return plan, nil
}

// validateReferencedHosts ensures every host_id mentioned by the desired image
// exists. Wrapping with ErrReferenceNotFound lets handlers map this to a 404
// without having to discriminate between "host not found" (data) and
// "reference not found" (constraint).
func (s *Service) validateReferencedHosts(ctx context.Context, groups []DesiredHostGroup) error {
	hostSet := make(map[ids.KnownHostID]struct{})
	for _, g := range groups {
		for _, id := range g.HostIDs {
			hostSet[id] = struct{}{}
		}
	}

	if len(hostSet) == 0 {
		return nil
	}

	hostIDs := make([]ids.KnownHostID, 0, len(hostSet))
	for id := range hostSet {
		hostIDs = append(hostIDs, id)
	}

	hosts, err := s.repo.ListKnownHostsByIDs(ctx, hostIDs)
	if err != nil {
		return err
	}

	found := make(map[ids.KnownHostID]struct{}, len(hosts))
	for _, h := range hosts {
		found[h.ID] = struct{}{}
	}

	for _, id := range hostIDs {
		if _, ok := found[id]; !ok {
			return fmt.Errorf("%w: known_host_id=%d", ErrReferenceNotFound, id)
		}
	}

	return nil
}
