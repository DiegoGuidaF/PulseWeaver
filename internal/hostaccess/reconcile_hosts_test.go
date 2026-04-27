//go:build test

package hostaccess

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"
)

// ── buildKnownHostReconcilePlan (pure) ───────────────────────────────────────

func TestBuildKnownHostReconcilePlan_CreateOnly(t *testing.T) {
	is := is.New(t)
	plan, err := buildKnownHostReconcilePlan(nil, []DesiredKnownHost{{FQDN: "new.example.com"}})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].FQDN, "new.example.com")
	is.Equal(len(plan.toUpdate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildKnownHostReconcilePlan_DeleteOnly(t *testing.T) {
	is := is.New(t)
	current := []KnownHost{{ID: 1, FQDN: "doomed.com"}}
	plan, err := buildKnownHostReconcilePlan(current, nil)
	is.NoErr(err)
	is.Equal(plan.toDelete, []KnownHostID{1})
	is.Equal(len(plan.toCreate), 0)
	is.Equal(len(plan.toUpdate), 0)
}

func TestBuildKnownHostReconcilePlan_IconUpdateOnly(t *testing.T) {
	is := is.New(t)
	current := []KnownHost{{ID: 1, FQDN: "host.example.com", Icon: nil}}
	icon := "server"
	id := KnownHostID(1)
	plan, err := buildKnownHostReconcilePlan(current, []DesiredKnownHost{{ID: &id, FQDN: "host.example.com", Icon: &icon}})
	is.NoErr(err)
	is.Equal(len(plan.toUpdate), 1)
	is.Equal(plan.toUpdate[0].ID, KnownHostID(1))
	is.Equal(*plan.toUpdate[0].Icon, "server")
	is.Equal(len(plan.toCreate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildKnownHostReconcilePlan_IconNoOp_Skipped(t *testing.T) {
	is := is.New(t)
	icon := "server"
	current := []KnownHost{{ID: 1, FQDN: "host.example.com", Icon: &icon}}
	id := KnownHostID(1)
	plan, err := buildKnownHostReconcilePlan(current, []DesiredKnownHost{{ID: &id, FQDN: "host.example.com", Icon: &icon}})
	is.NoErr(err)
	is.Equal(len(plan.toUpdate), 0)
	is.Equal(len(plan.toCreate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildKnownHostReconcilePlan_UnknownDesiredID_Errors(t *testing.T) {
	is := is.New(t)
	id := KnownHostID(42)
	_, err := buildKnownHostReconcilePlan(nil, []DesiredKnownHost{{ID: &id, FQDN: "ghost.example.com"}})
	is.True(errors.Is(err, ErrKnownHostNotFound))
}

func TestBuildKnownHostReconcilePlan_FQDNChangeOnExistingID_Errors(t *testing.T) {
	is := is.New(t)
	current := []KnownHost{{ID: 1, FQDN: "original.com"}}
	id := KnownHostID(1)
	_, err := buildKnownHostReconcilePlan(current, []DesiredKnownHost{{ID: &id, FQDN: "changed.com"}})
	is.True(errors.Is(err, ErrKnownHostFQDNImmutable))
}

func TestBuildKnownHostReconcilePlan_CreateConflictsWithCurrent_Errors(t *testing.T) {
	is := is.New(t)
	current := []KnownHost{{ID: 1, FQDN: "taken.com"}}
	_, err := buildKnownHostReconcilePlan(current, []DesiredKnownHost{{FQDN: "taken.com"}})
	is.True(errors.Is(err, ErrKnownHostConflict))
}

func TestBuildKnownHostReconcilePlan_Mixed(t *testing.T) {
	is := is.New(t)
	icon := "server"
	current := []KnownHost{
		{ID: 1, FQDN: "keep.com"},
		{ID: 2, FQDN: "update-me.com"},
		{ID: 3, FQDN: "remove-me.com"},
	}
	id1 := KnownHostID(1)
	id2 := KnownHostID(2)
	plan, err := buildKnownHostReconcilePlan(current, []DesiredKnownHost{
		{ID: &id1, FQDN: "keep.com"},                   // no-op
		{ID: &id2, FQDN: "update-me.com", Icon: &icon}, // update icon
		{FQDN: "fresh.com"},                            // create
		// ID 3 absent → delete
	})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].FQDN, "fresh.com")
	is.Equal(len(plan.toUpdate), 1)
	is.Equal(plan.toUpdate[0].ID, KnownHostID(2))
	is.Equal(plan.toDelete, []KnownHostID{3})
}

// ── ReconcileKnownHostsInput.prepare() ───────────────────────────────────────

func TestReconcileKnownHostsInput_DuplicateID_Rejected(t *testing.T) {
	is := is.New(t)
	id := KnownHostID(7)
	in := ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{
			{ID: &id, FQDN: "a.example.com"},
			{ID: &id, FQDN: "b.example.com"},
		},
	}
	is.True(errors.Is(in.prepare(), ErrDuplicateKnownHostID))
}

func TestReconcileKnownHostsInput_DuplicateFQDN_Rejected(t *testing.T) {
	is := is.New(t)
	in := ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{
			{FQDN: "dup.example.com"},
			{FQDN: "dup.example.com"},
		},
	}
	is.True(errors.Is(in.prepare(), ErrDuplicateKnownHostFQDN))
}

func TestReconcileKnownHostsInput_InvalidFQDN_Rejected(t *testing.T) {
	is := is.New(t)
	in := ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{{FQDN: "not-valid"}},
	}
	is.True(errors.Is(in.prepare(), ErrBadRequest))
}

func TestReconcileKnownHostsInput_NormalisesAndDeduplicatesFQDN(t *testing.T) {
	is := is.New(t)
	in := ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{
			{FQDN: "  UPPER.Example.COM  "},
			{FQDN: "other.example.com"},
		},
	}
	is.NoErr(in.prepare())
	is.Equal(in.Hosts[0].FQDN, "upper.example.com")
}

func TestReconcileKnownHostsInput_IconTrimmedAndNilledIfEmpty(t *testing.T) {
	is := is.New(t)
	empty := "   "
	in := ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{{FQDN: "host.example.com", Icon: &empty}},
	}
	is.NoErr(in.prepare())
	is.Equal(in.Hosts[0].Icon, (*string)(nil))
}

// ── Service.ReconcileKnownHosts ───────────────────────────────────────────────

func TestService_ReconcileKnownHosts_NotifiesObserversOnce(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	repo := &fakeRepo{knownHosts: []KnownHost{{ID: 1, FQDN: "old.com"}}}
	svc, _ := newTestService(repo)
	svc.AddUserHostAccessObserver(obs)

	err := svc.ReconcileKnownHosts(context.Background(), ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{{FQDN: "new.example.com"}},
	})
	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_ReconcileKnownHosts_NoOp_StillNotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	id := KnownHostID(1)
	repo := &fakeRepo{knownHosts: []KnownHost{{ID: 1, FQDN: "stable.example.com"}}}
	svc, _ := newTestService(repo)
	svc.AddUserHostAccessObserver(obs)

	err := svc.ReconcileKnownHosts(context.Background(), ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{{ID: &id, FQDN: "stable.example.com"}},
	})
	is.NoErr(err)
	is.Equal(len(repo.callOrder), 0) // no writes
	is.Equal(obs.calls, 1)           // observer still fired
}

func TestService_ReconcileKnownHosts_DeleteUpdateCreate_Order(t *testing.T) {
	is := is.New(t)
	icon := "server"
	repo := &fakeRepo{
		knownHosts: []KnownHost{
			{ID: 1, FQDN: "update-me.example.com"},
			{ID: 2, FQDN: "delete-me.example.com"},
		},
	}
	svc, _ := newTestService(repo)

	id1 := KnownHostID(1)
	err := svc.ReconcileKnownHosts(context.Background(), ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{
			{ID: &id1, FQDN: "update-me.example.com", Icon: &icon},
			{FQDN: "new.example.com"},
		},
	})
	is.NoErr(err)
	is.Equal(repo.callOrder, []string{"deleteHost", "updateHost", "createHost"})
}

func TestService_ReconcileKnownHosts_EmptyInput_DeletesAll(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		knownHosts: []KnownHost{
			{ID: 1, FQDN: "a.example.com"},
			{ID: 2, FQDN: "b.example.com"},
		},
	}
	svc, _ := newTestService(repo)

	err := svc.ReconcileKnownHosts(context.Background(), ReconcileKnownHostsInput{})
	is.NoErr(err)
	is.Equal(len(repo.deleteHostCalls), 2)
	is.Equal(len(repo.createHostCalls), 0)
}

func TestService_ReconcileKnownHosts_ConflictFromCreate_Surfaces(t *testing.T) {
	is := is.New(t)
	svc, _ := newTestService(&createConflictRepo{})

	err := svc.ReconcileKnownHosts(context.Background(), ReconcileKnownHostsInput{
		Hosts: []DesiredKnownHost{{FQDN: "conflict.example.com"}},
	})
	is.True(errors.Is(err, ErrKnownHostConflict))
}

// createConflictRepo is a minimal fakeRepo where CreateKnownHost always returns ErrKnownHostConflict.
type createConflictRepo struct{ fakeRepo }

func (c *createConflictRepo) CreateKnownHost(_ context.Context, _ KnownHostDraft) error {
	return ErrKnownHostConflict
}
func (c *createConflictRepo) ListKnownHosts(_ context.Context) ([]KnownHost, error) {
	return nil, nil // empty current list so plan always has creates
}
