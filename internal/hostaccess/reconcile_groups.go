package hostaccess

import (
	"context"
	"fmt"
	"strings"
)

type DesiredGroup struct {
	ID          *HostGroupID
	Name        string
	Color       string
	Description *string
	Icon        *string
	HostIDs     []KnownHostID
}

// prepare Normalizes and validates. Trimming strings, deduplication...
func (g *DesiredGroup) prepare() error {
	g.Name = strings.TrimSpace(g.Name)
	g.HostIDs = deduplicateHostIDs(g.HostIDs)

	if g.Name == "" {
		return ErrGroupNameRequired
	}

	return nil
}

type ReconcileGroupsInput struct {
	Groups []DesiredGroup
}

// prepare Normalizes and validates. Normalizes each group as well as checks that group IDs are not duplicated
func (in *ReconcileGroupsInput) prepare() error {
	// Prepare each group within
	for i := range in.Groups {
		if err := in.Groups[i].prepare(); err != nil {
			return err
		}
	}

	// Ensure groups are not repeated
	seenIDs := make(map[HostGroupID]struct{}, len(in.Groups))
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

type groupReconcilePlan struct {
	toCreate []HostGroupDraft
	toUpdate []HostGroup
	toDelete []HostGroupID
}

type HostGroupDraft struct {
	Name        string
	Color       string
	Description *string
	Icon        *string
	HostIDs     []KnownHostID
}

func (s *Service) ReconcileGroups(ctx context.Context, in ReconcileGroupsInput) error {
	err := in.prepare()
	if err != nil {
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

	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		for _, draft := range plan.toCreate {
			if _, err := s.repo.CreateHostGroup(ctx, draft); err != nil {
				return err
			}
		}

		for _, group := range plan.toUpdate {
			if err := s.repo.UpdateHostGroup(ctx, group); err != nil {
				return err
			}
		}

		for _, id := range plan.toDelete {
			if err := s.repo.DeleteHostGroup(ctx, id); err != nil {
				return err
			}
		}

		return nil
	})
}

func buildGroupReconcilePlan(current []HostGroup, desired []DesiredGroup) (groupReconcilePlan, error) {
	currentByID := make(map[HostGroupID]HostGroup, len(current))
	for _, g := range current {
		currentByID[g.ID] = g
	}

	desiredIDs := make(map[HostGroupID]struct{})

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
			return groupReconcilePlan{}, ErrHostGroupNotFound
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

	// Remove all existing groups whose ID is not in the desired list
	for _, g := range current {
		if _, ok := desiredIDs[g.ID]; !ok {
			plan.toDelete = append(plan.toDelete, g.ID)
		}
	}

	//TODO: Add logging to this plan

	return plan, nil
}
func (s *Service) validateReferencedHosts(ctx context.Context, groups []DesiredGroup) error {
	hostSet := make(map[KnownHostID]struct{})
	for _, g := range groups {
		for _, id := range g.HostIDs {
			hostSet[id] = struct{}{}
		}
	}

	if len(hostSet) == 0 {
		return nil
	}

	ids := make([]KnownHostID, 0, len(hostSet))
	for id := range hostSet {
		ids = append(ids, id)
	}

	hosts, err := s.repo.ListKnownHostsByIDs(ctx, ids)
	if err != nil {
		return err
	}

	found := make(map[KnownHostID]struct{}, len(hosts))
	for _, h := range hosts {
		found[h.ID] = struct{}{}
	}

	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return fmt.Errorf("%w: host_id=%d", ErrKnownHostNotFound, id)
		}
	}

	return nil
}
