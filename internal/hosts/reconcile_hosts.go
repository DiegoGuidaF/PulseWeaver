package hosts

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/slicex"
)

// DesiredHost is the caller-provided shape of a single host inside a reconcile request.
// A nil ID marks a brand-new host; a non-nil ID must match an existing row.
// FQDN is immutable on updates — a mismatch returns ErrHostFQDNImmutable.
// GroupIDs replaces the host's full group membership.
type DesiredHost struct {
	ID       *ids.HostID
	FQDN     string
	GroupIDs []ids.HostGroupID
}

func (h *DesiredHost) prepare() error {
	h.FQDN = NormaliseFQDN(h.FQDN)
	if err := ValidateFQDN(h.FQDN); err != nil {
		return err
	}
	h.GroupIDs = slicex.Dedup(h.GroupIDs)
	return nil
}

// ReconcileHostsInput is the full desired image of hosts that the caller wants
// the database to converge to.
type ReconcileHostsInput struct {
	Hosts []DesiredHost
}

func (in *ReconcileHostsInput) prepare() error {
	for i := range in.Hosts {
		if err := in.Hosts[i].prepare(); err != nil {
			return err
		}
	}

	seenIDs := make(map[ids.HostID]struct{}, len(in.Hosts))
	for _, h := range in.Hosts {
		if h.ID != nil {
			if _, ok := seenIDs[*h.ID]; ok {
				return ErrDuplicateHostID
			}
			seenIDs[*h.ID] = struct{}{}
		}
	}

	seenFQDNs := make(map[string]struct{}, len(in.Hosts))
	for _, h := range in.Hosts {
		if _, ok := seenFQDNs[h.FQDN]; ok {
			return ErrDuplicateHostFQDN
		}
		seenFQDNs[h.FQDN] = struct{}{}
	}

	return nil
}

type hostReconcilePlan struct {
	toCreate    []HostDraft
	toDelete    []ids.HostID
	toSetGroups []hostGroupSet
}

type hostGroupSet struct {
	HostID   ids.HostID
	GroupIDs []ids.HostGroupID
}

// HostDraft is the minimum shape needed to insert a new hosts row.
type HostDraft struct {
	FQDN     string
	GroupIDs []ids.HostGroupID
}

// ReconcileHosts makes the database converge to the desired image of hosts in a
// single transaction. Hosts present in `in` with a non-nil ID retain their group
// membership as specified; hosts with a nil ID are created; hosts currently in the
// database whose ID is absent from `in` are deleted.
//
// Note: deleting a host cascades to host_group_members (ON DELETE CASCADE), so
// a reconcile that drops hosts implicitly changes effective user host access.
// Observers are always notified on success.
func (s *Service) ReconcileHosts(ctx context.Context, in ReconcileHostsInput) error {
	if err := in.prepare(); err != nil {
		return err
	}

	currentHosts, err := s.repo.ListHosts(ctx)
	if err != nil {
		return err
	}

	plan, err := buildHostReconcilePlan(currentHosts, in.Hosts)
	if err != nil {
		return err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Order matters: deletes free up FQDNs (unique index on hosts.fqdn)
		// that subsequent creates may want to claim.
		// CASCADE on host_group_members handles group membership for deleted hosts.
		for _, id := range plan.toDelete {
			if err := s.repo.DeleteHost(ctx, id); err != nil {
				return err
			}
		}
		for _, draft := range plan.toCreate {
			newID, err := s.repo.CreateHost(ctx, draft)
			if err != nil {
				return err
			}
			if err := s.repo.SetHostGroupMembership(ctx, newID, draft.GroupIDs); err != nil {
				return err
			}
		}
		for _, gs := range plan.toSetGroups {
			if err := s.repo.SetHostGroupMembership(ctx, gs.HostID, gs.GroupIDs); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	s.notifyObservers(ctx)
	return nil
}

// ListHosts returns all hosts ordered by ID.
func (s *Service) ListHosts(ctx context.Context) ([]Host, error) {
	return s.repo.ListHosts(ctx)
}

// buildHostReconcilePlan diffs the current hosts against the desired image and
// produces the create/delete buckets. A desired host with a non-nil ID that is
// unknown returns ErrHostNotFound. A desired host whose FQDN differs from the
// current row's FQDN returns ErrHostFQDNImmutable.
func buildHostReconcilePlan(current []Host, desired []DesiredHost) (hostReconcilePlan, error) {
	currentByID := make(map[ids.HostID]Host, len(current))
	currentByFQDN := make(map[string]struct{}, len(current))
	for _, h := range current {
		currentByID[h.ID] = h
		currentByFQDN[h.FQDN] = struct{}{}
	}

	desiredIDs := make(map[ids.HostID]struct{})
	var plan hostReconcilePlan

	for _, h := range desired {
		if h.ID == nil {
			if _, exists := currentByFQDN[h.FQDN]; exists {
				return hostReconcilePlan{}, fmt.Errorf("%w: fqdn=%s", ErrHostConflict, h.FQDN)
			}
			plan.toCreate = append(plan.toCreate, HostDraft{FQDN: h.FQDN, GroupIDs: h.GroupIDs})
			continue
		}

		id := *h.ID
		existing, ok := currentByID[id]
		if !ok {
			return hostReconcilePlan{}, fmt.Errorf("%w: id=%d", ErrHostNotFound, id)
		}
		desiredIDs[id] = struct{}{}

		if existing.FQDN != h.FQDN {
			return hostReconcilePlan{}, fmt.Errorf("%w: id=%d current=%s desired=%s",
				ErrHostFQDNImmutable, id, existing.FQDN, h.FQDN)
		}

		// Always replace group membership for existing hosts (replace-all semantics).
		plan.toSetGroups = append(plan.toSetGroups, hostGroupSet{HostID: id, GroupIDs: h.GroupIDs})
	}

	for _, h := range current {
		if _, ok := desiredIDs[h.ID]; !ok {
			plan.toDelete = append(plan.toDelete, h.ID)
		}
	}

	return plan, nil
}
