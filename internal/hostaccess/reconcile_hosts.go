package hostaccess

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/slicex"
)

// DesiredKnownHost is the caller-provided shape of a single known host inside
// a reconcile request. A nil ID marks a brand-new host; a non-nil ID must
// match an existing row. FQDN is immutable on updates — a mismatch returns
// ErrKnownHostFQDNImmutable. GroupIDs replaces the host's full group membership.
type DesiredKnownHost struct {
	ID       *ids.KnownHostID
	FQDN     string
	Icon     *string
	GroupIDs []ids.HostGroupID
}

// prepare normalises and validates a single desired host: lowercases and
// trims FQDN, validates it, trims Icon (setting it to nil if empty), and
// deduplicates GroupIDs (host_group_members has a composite PK).
func (h *DesiredKnownHost) prepare() error {
	h.FQDN = NormaliseFQDN(h.FQDN)
	if err := ValidateFQDN(h.FQDN); err != nil {
		return err
	}
	if h.Icon != nil {
		trimmed := strings.TrimSpace(*h.Icon)
		if trimmed == "" {
			h.Icon = nil
		} else {
			h.Icon = &trimmed
		}
	}
	h.GroupIDs = slicex.Dedup(h.GroupIDs)
	return nil
}

// ReconcileKnownHostsInput is the full desired image of known_hosts that the
// caller wants the database to converge to.
type ReconcileKnownHostsInput struct {
	Hosts []DesiredKnownHost
}

// prepare normalises every host entry and rejects duplicate IDs or FQDNs
// across the desired list (which would make the create/update/delete plan
// ambiguous).
func (in *ReconcileKnownHostsInput) prepare() error {
	for i := range in.Hosts {
		if err := in.Hosts[i].prepare(); err != nil {
			return err
		}
	}

	seenIDs := make(map[ids.KnownHostID]struct{}, len(in.Hosts))
	for _, h := range in.Hosts {
		if h.ID != nil {
			if _, ok := seenIDs[*h.ID]; ok {
				return ErrDuplicateKnownHostID
			}
			seenIDs[*h.ID] = struct{}{}
		}
	}

	seenFQDNs := make(map[string]struct{}, len(in.Hosts))
	for _, h := range in.Hosts {
		if _, ok := seenFQDNs[h.FQDN]; ok {
			return ErrDuplicateKnownHostFQDN
		}
		seenFQDNs[h.FQDN] = struct{}{}
	}

	return nil
}

// knownHostReconcilePlan is the ordered set of write operations needed to
// converge the current state to the desired state.
type knownHostReconcilePlan struct {
	toCreate    []KnownHostDraft
	toUpdate    []KnownHost
	toDelete    []ids.KnownHostID
	toSetGroups []knownHostGroupSet // group membership for all non-deleted hosts
}

// knownHostGroupSet pairs a host with the full set of groups it should belong to.
type knownHostGroupSet struct {
	HostID   ids.KnownHostID
	GroupIDs []ids.HostGroupID
}

// KnownHostDraft is the minimum shape needed to insert a new known_hosts row.
type KnownHostDraft struct {
	FQDN     string
	Icon     *string
	GroupIDs []ids.HostGroupID
}

// ReconcileKnownHosts makes the database converge to the desired image of
// known_hosts in a single transaction. Hosts present in `in` with a non-nil
// ID are updated (icon only); hosts with a nil ID are created; hosts currently
// in the database whose ID is absent from `in` are deleted.
//
// Note: deleting a known host cascades to host_group_members and
// user_allowed_hosts (ON DELETE CASCADE — migration 000018 lines 48, 62),
// so a reconcile that drops hosts implicitly changes effective user host
// access. Observers are always notified on success.
func (s *Service) ReconcileKnownHosts(ctx context.Context, in ReconcileKnownHostsInput) error {
	if err := in.prepare(); err != nil {
		return err
	}

	currentHosts, err := s.repo.ListKnownHosts(ctx)
	if err != nil {
		return err
	}

	plan, err := buildKnownHostReconcilePlan(currentHosts, in.Hosts)
	if err != nil {
		return err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Order matters: deletes free up FQDNs (unique index on known_hosts.fqdn)
		// that subsequent creates may want to claim. Updates can't change FQDN
		// (server-side invariant), so they never cause index collisions.
		// CASCADE on host_group_members handles group membership for deleted hosts.
		for _, id := range plan.toDelete {
			if err := s.repo.DeleteKnownHost(ctx, id); err != nil {
				return err
			}
		}

		for _, host := range plan.toUpdate {
			if _, err := s.repo.UpdateKnownHost(ctx, host.ID, host.Icon); err != nil {
				return err
			}
		}

		for _, draft := range plan.toCreate {
			newID, err := s.repo.CreateKnownHost(ctx, draft)
			if err != nil {
				return err
			}
			if err := s.repo.SetKnownHostGroupMembership(ctx, newID, draft.GroupIDs); err != nil {
				return err
			}
		}

		for _, gs := range plan.toSetGroups {
			if err := s.repo.SetKnownHostGroupMembership(ctx, gs.HostID, gs.GroupIDs); err != nil {
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

// ListKnownHosts returns all known hosts ordered by ID.
func (s *Service) ListKnownHosts(ctx context.Context) ([]KnownHost, error) {
	return s.repo.ListKnownHosts(ctx)
}

// buildKnownHostReconcilePlan diffs the current known hosts against the
// desired image and produces the create/update/delete buckets. A desired host
// with a non-nil ID that is unknown returns ErrKnownHostNotFound. A desired
// host whose FQDN differs from the current row's FQDN returns
// ErrKnownHostFQDNImmutable. Icon-only no-ops (icon unchanged) are skipped.
func buildKnownHostReconcilePlan(current []KnownHost, desired []DesiredKnownHost) (knownHostReconcilePlan, error) {
	currentByID := make(map[ids.KnownHostID]KnownHost, len(current))
	currentByFQDN := make(map[string]struct{}, len(current))
	for _, h := range current {
		currentByID[h.ID] = h
		currentByFQDN[h.FQDN] = struct{}{}
	}

	desiredIDs := make(map[ids.KnownHostID]struct{})
	var plan knownHostReconcilePlan

	for _, h := range desired {
		if h.ID == nil {
			// Create: reject if a current row already has that FQDN.
			if _, exists := currentByFQDN[h.FQDN]; exists {
				return knownHostReconcilePlan{}, fmt.Errorf("%w: fqdn=%s", ErrKnownHostConflict, h.FQDN)
			}
			plan.toCreate = append(plan.toCreate, KnownHostDraft{FQDN: h.FQDN, Icon: h.Icon, GroupIDs: h.GroupIDs})
			continue
		}

		id := *h.ID
		existing, ok := currentByID[id]
		if !ok {
			return knownHostReconcilePlan{}, fmt.Errorf("%w: id=%d", ErrKnownHostNotFound, id)
		}
		desiredIDs[id] = struct{}{}

		if existing.FQDN != h.FQDN {
			return knownHostReconcilePlan{}, fmt.Errorf("%w: id=%d current=%s desired=%s",
				ErrKnownHostFQDNImmutable, id, existing.FQDN, h.FQDN)
		}

		if !iconEqual(existing.Icon, h.Icon) {
			plan.toUpdate = append(plan.toUpdate, KnownHost{ID: id, FQDN: existing.FQDN, Icon: h.Icon})
		}

		// Always replace group membership for existing hosts (replace-all semantics).
		plan.toSetGroups = append(plan.toSetGroups, knownHostGroupSet{HostID: id, GroupIDs: h.GroupIDs})
	}

	for _, h := range current {
		if _, ok := desiredIDs[h.ID]; !ok {
			plan.toDelete = append(plan.toDelete, h.ID)
		}
	}

	return plan, nil
}

// iconEqual compares two nullable icon strings for equality.
func iconEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
