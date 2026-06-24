//go:build test

package hosts

import (
	"context"
	"errors"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

// ── buildHostReconcilePlan (pure) ────────────────────────────────────────────

func TestBuildHostReconcilePlan_CreateOnly(t *testing.T) {
	is := is.New(t)
	plan, err := buildHostReconcilePlan(nil, []DesiredHost{{FQDN: "new.example.com"}})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].FQDN, "new.example.com")
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildHostReconcilePlan_DeleteOnly(t *testing.T) {
	is := is.New(t)
	current := []Host{{ID: 1, FQDN: "doomed.com"}}
	plan, err := buildHostReconcilePlan(current, nil)
	is.NoErr(err)
	is.Equal(plan.toDelete, []ids.HostID{1})
	is.Equal(len(plan.toCreate), 0)
}

func TestBuildHostReconcilePlan_UnknownDesiredID_Errors(t *testing.T) {
	is := is.New(t)
	id := ids.HostID(42)
	_, err := buildHostReconcilePlan(nil, []DesiredHost{{ID: &id, FQDN: "ghost.example.com"}})
	is.True(errors.Is(err, ErrHostNotFound))
}

func TestBuildHostReconcilePlan_FQDNChangeOnExistingID_Errors(t *testing.T) {
	is := is.New(t)
	current := []Host{{ID: 1, FQDN: "original.com"}}
	id := ids.HostID(1)
	_, err := buildHostReconcilePlan(current, []DesiredHost{{ID: &id, FQDN: "changed.com"}})
	is.True(errors.Is(err, ErrHostFQDNImmutable))
}

func TestBuildHostReconcilePlan_CreateConflictsWithCurrent_Errors(t *testing.T) {
	is := is.New(t)
	current := []Host{{ID: 1, FQDN: "taken.com"}}
	_, err := buildHostReconcilePlan(current, []DesiredHost{{FQDN: "taken.com"}})
	is.True(errors.Is(err, ErrHostConflict))
}

func TestBuildHostReconcilePlan_Mixed(t *testing.T) {
	is := is.New(t)
	current := []Host{
		{ID: 1, FQDN: "keep.com"},
		{ID: 3, FQDN: "remove-me.com"},
	}
	id1 := ids.HostID(1)
	plan, err := buildHostReconcilePlan(current, []DesiredHost{
		{ID: &id1, FQDN: "keep.com"},
		{FQDN: "fresh.com"},
	})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].FQDN, "fresh.com")
	is.Equal(plan.toDelete, []ids.HostID{3})
}

// ── ReconcileHostsInput.prepare() ────────────────────────────────────────────

func TestReconcileHostsInput_DuplicateID_Rejected(t *testing.T) {
	is := is.New(t)
	id := ids.HostID(7)
	in := ReconcileHostsInput{
		Hosts: []DesiredHost{
			{ID: &id, FQDN: "a.example.com"},
			{ID: &id, FQDN: "b.example.com"},
		},
	}
	is.True(errors.Is(in.prepare(), ErrDuplicateHostID))
}

func TestReconcileHostsInput_DuplicateFQDN_Rejected(t *testing.T) {
	is := is.New(t)
	in := ReconcileHostsInput{
		Hosts: []DesiredHost{
			{FQDN: "dup.example.com"},
			{FQDN: "dup.example.com"},
		},
	}
	is.True(errors.Is(in.prepare(), ErrDuplicateHostFQDN))
}

func TestReconcileHostsInput_InvalidFQDN_Rejected(t *testing.T) {
	is := is.New(t)
	in := ReconcileHostsInput{
		Hosts: []DesiredHost{{FQDN: "not-valid"}},
	}
	is.True(errors.Is(in.prepare(), ErrBadRequest))
}

func TestReconcileHostsInput_NormalisesAndDeduplicatesFQDN(t *testing.T) {
	is := is.New(t)
	in := ReconcileHostsInput{
		Hosts: []DesiredHost{
			{FQDN: "  UPPER.Example.COM  "},
			{FQDN: "other.example.com"},
			{FQDN: "ported.example.com:8443"}, // a pasted host:port resolves to the bare FQDN
		},
	}
	is.NoErr(in.prepare())
	is.Equal(in.Hosts[0].FQDN, "upper.example.com")
	is.Equal(in.Hosts[2].FQDN, "ported.example.com")
}

// ── Service.ReconcileHosts ────────────────────────────────────────────────────

func TestService_ReconcileHosts_NotifiesObserversOnce(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	repo := &fakeRepo{hosts: []Host{{ID: 1, FQDN: "old.com"}}}
	svc := newTestService(repo)
	svc.AddObserver(obs)

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{{FQDN: "new.example.com"}},
	})
	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_ReconcileHosts_NoOp_StillNotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	repo := &fakeRepo{hosts: []Host{{ID: 1, FQDN: "stable.example.com"}}}
	svc := newTestService(repo)
	svc.AddObserver(obs)

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{{ID: new(ids.HostID(1)), FQDN: "stable.example.com"}},
	})
	is.NoErr(err)
	is.Equal(repo.callOrder, []string{"setHostGroups"})
	is.Equal(obs.calls, 1)
}

func TestService_ReconcileHosts_DeleteAndCreate_Order(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		hosts: []Host{
			{ID: 2, FQDN: "delete-me.example.com"},
		},
	}
	svc := newTestService(repo)

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{
			{FQDN: "new.example.com"},
		},
	})
	is.NoErr(err)
	is.Equal(repo.callOrder, []string{"deleteHost", "createHost", "setHostGroups"})
}

func TestService_ReconcileHosts_EmptyInput_DeletesAll(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		hosts: []Host{
			{ID: 1, FQDN: "a.example.com"},
			{ID: 2, FQDN: "b.example.com"},
		},
	}
	svc := newTestService(repo)

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{})
	is.NoErr(err)
	is.Equal(len(repo.deleteHostCalls), 2)
	is.Equal(len(repo.createHostCalls), 0)
}

func TestService_ReconcileHosts_ConflictFromCreate_Surfaces(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&createConflictRepo{})

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{{FQDN: "conflict.example.com"}},
	})
	is.True(errors.Is(err, ErrHostConflict))
}

type createConflictRepo struct{ fakeRepo }

func (c *createConflictRepo) CreateHost(_ context.Context, _ HostDraft) (ids.HostID, error) {
	return ids.HostID(0), ErrHostConflict
}
func (c *createConflictRepo) ListHosts(_ context.Context) ([]Host, error) { return nil, nil }

// ── Group ID reconciliation ───────────────────────────────────────────────────

func TestService_ReconcileHosts_GroupIDsSetOnCreate(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	svc := newTestService(repo)

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{
			{FQDN: "new.example.com", GroupIDs: []ids.HostGroupID{10, 20}},
		},
	})
	is.NoErr(err)
	is.Equal(len(repo.setGroupCalls), 1)
	is.Equal(repo.setGroupCalls[0].GroupIDs, []ids.HostGroupID{10, 20})
}

func TestService_ReconcileHosts_GroupIDsSetOnExisting(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{hosts: []Host{{ID: 1, FQDN: "host.example.com"}}}
	svc := newTestService(repo)

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{
			{ID: new(ids.HostID(1)), FQDN: "host.example.com", GroupIDs: []ids.HostGroupID{5}},
		},
	})
	is.NoErr(err)
	is.Equal(len(repo.setGroupCalls), 1)
	is.Equal(repo.setGroupCalls[0].HostID, ids.HostID(1))
	is.Equal(repo.setGroupCalls[0].GroupIDs, []ids.HostGroupID{5})
}

func TestService_ReconcileHosts_GroupIDsDeduplicated(t *testing.T) {
	is := is.New(t)
	in := ReconcileHostsInput{
		Hosts: []DesiredHost{
			{FQDN: "host.example.com", GroupIDs: []ids.HostGroupID{3, 3, 7}},
		},
	}
	is.NoErr(in.prepare())
	is.Equal(in.Hosts[0].GroupIDs, []ids.HostGroupID{3, 7})
}

func TestService_ReconcileHosts_BadGroupID_SurfacesReferenceNotFound(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&badGroupRepo{fakeRepo: fakeRepo{hosts: nil}})

	err := svc.ReconcileHosts(context.Background(), ReconcileHostsInput{
		Hosts: []DesiredHost{{FQDN: "new.example.com", GroupIDs: []ids.HostGroupID{999}}},
	})
	is.True(errors.Is(err, ErrReferenceNotFound))
}

type badGroupRepo struct{ fakeRepo }

func (b *badGroupRepo) SetHostGroupMembership(_ context.Context, _ ids.HostID, _ []ids.HostGroupID) error {
	return ErrReferenceNotFound
}
func (b *badGroupRepo) ListHosts(_ context.Context) ([]Host, error) { return nil, nil }
func (b *badGroupRepo) CreateHost(_ context.Context, _ HostDraft) (ids.HostID, error) {
	return ids.HostID(101), nil
}
