//go:build test

package networkpolicies_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	policies     []networkpolicies.NetworkPolicy
	createErr    error
	getErr       error
	updateErr    error
	deleteErr    error
	setAccessErr error
	nextID       ids.NetworkPolicyID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{nextID: 1}
}

func (f *fakeRepo) CreatePolicy(_ context.Context, p networkpolicies.NetworkPolicy) (networkpolicies.NetworkPolicy, error) {
	if f.createErr != nil {
		return networkpolicies.NetworkPolicy{}, f.createErr
	}
	p.ID = f.nextID
	f.nextID++
	f.policies = append(f.policies, p)
	return p, nil
}

func (f *fakeRepo) GetPolicy(_ context.Context, id ids.NetworkPolicyID) (networkpolicies.NetworkPolicy, error) {
	if f.getErr != nil {
		return networkpolicies.NetworkPolicy{}, f.getErr
	}
	for _, p := range f.policies {
		if p.ID == id {
			return p, nil
		}
	}
	return networkpolicies.NetworkPolicy{}, networkpolicies.ErrNotFound
}

func (f *fakeRepo) UpdatePolicy(_ context.Context, p networkpolicies.NetworkPolicy) (networkpolicies.NetworkPolicy, error) {
	if f.updateErr != nil {
		return networkpolicies.NetworkPolicy{}, f.updateErr
	}
	for i, existing := range f.policies {
		if existing.ID == p.ID {
			f.policies[i] = p
			return p, nil
		}
	}
	return networkpolicies.NetworkPolicy{}, networkpolicies.ErrNotFound
}

func (f *fakeRepo) DeletePolicy(_ context.Context, id ids.NetworkPolicyID) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	for i, p := range f.policies {
		if p.ID == id {
			f.policies = append(f.policies[:i], f.policies[i+1:]...)
			return nil
		}
	}
	return networkpolicies.ErrNotFound
}

func (f *fakeRepo) SetHostAccess(_ context.Context, _ ids.NetworkPolicyID, _ bool, _ []ids.HostGroupID) error {
	return f.setAccessErr
}

type fakeObserver struct{ calls int }

func (o *fakeObserver) OnNetworkPolicyChanged(_ context.Context) { o.calls++ }

type errTransactor struct{ err error }

func (e errTransactor) WithinTx(_ context.Context, _ func(context.Context) error) error {
	return e.err
}

func newService(repo *fakeRepo) *networkpolicies.Service {
	return networkpolicies.NewService(repo, testutils.NoopTransactor{}, slog.New(slog.DiscardHandler))
}

// ── CreatePolicy ─────────────────────────────────────────────────────────────

func TestService_CreatePolicy_NormalizesCIDR(t *testing.T) {
	is := is.New(t)
	svc := newService(newFakeRepo())

	p, err := svc.CreatePolicy(context.Background(), "home", "192.168.1.5/24", nil)

	is.NoErr(err)
	is.Equal(p.CIDR, "192.168.1.0/24")
}

func TestService_CreatePolicy_InvalidCIDR_ReturnsErrInvalidCIDR(t *testing.T) {
	is := is.New(t)
	svc := newService(newFakeRepo())

	_, err := svc.CreatePolicy(context.Background(), "bad", "not-a-cidr", nil)

	is.True(errors.Is(err, networkpolicies.ErrInvalidCIDR))
}

func TestService_CreatePolicy_DoesNotNotifyObservers(t *testing.T) {
	is := is.New(t)
	obs := &fakeObserver{}
	svc := newService(newFakeRepo())
	svc.AddObserver(obs)

	_, err := svc.CreatePolicy(context.Background(), "home", "10.0.0.0/8", nil)

	is.NoErr(err)
	is.Equal(obs.calls, 0)
}

func TestService_CreatePolicy_RepoError_Propagated(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	repo.createErr = networkpolicies.ErrCIDRConflict
	svc := newService(repo)

	_, err := svc.CreatePolicy(context.Background(), "dup", "10.1.0.0/8", nil)

	is.True(errors.Is(err, networkpolicies.ErrCIDRConflict))
}

// ── UpdatePolicy ─────────────────────────────────────────────────────────────

func TestService_UpdatePolicy_AppliesFieldsAndNotifiesObserver(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	obs := &fakeObserver{}
	svc := newService(repo)
	svc.AddObserver(obs)

	created, err := repo.CreatePolicy(context.Background(), networkpolicies.NetworkPolicy{
		ID: 1, Name: "original", CIDR: "10.0.0.0/8", Enabled: true,
	})
	is.NoErr(err)

	desc := "updated desc"
	updated, err := svc.UpdatePolicy(context.Background(), created.ID, networkpolicies.UpdateFields{
		Name:        "renamed",
		CIDR:        "10.0.0.0/8",
		Description: &desc,
		Enabled:     false,
	})

	is.NoErr(err)
	is.Equal(updated.Name, "renamed")
	is.Equal(updated.Enabled, false)
	is.True(updated.Description != nil)
	is.Equal(*updated.Description, "updated desc")
	is.Equal(obs.calls, 1)
}

func TestService_UpdatePolicy_InvalidCIDR_ReturnsErrInvalidCIDR(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	svc := newService(repo)

	_, err := repo.CreatePolicy(context.Background(), networkpolicies.NetworkPolicy{
		ID: 1, Name: "p", CIDR: "10.0.0.0/8", Enabled: true,
	})
	is.NoErr(err)

	_, err = svc.UpdatePolicy(context.Background(), ids.NetworkPolicyID(1), networkpolicies.UpdateFields{
		Name: "p", CIDR: "bad-cidr",
	})

	is.True(errors.Is(err, networkpolicies.ErrInvalidCIDR))
}

func TestService_UpdatePolicy_GetError_Propagated(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	repo.getErr = networkpolicies.ErrNotFound
	svc := newService(repo)

	_, err := svc.UpdatePolicy(context.Background(), ids.NetworkPolicyID(1), networkpolicies.UpdateFields{
		Name: "p", CIDR: "10.0.0.0/8",
	})

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
}

func TestService_UpdatePolicy_TxError_NoNotify(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	obs := &fakeObserver{}
	sentinel := errors.New("tx failed")
	svc := networkpolicies.NewService(repo, errTransactor{err: sentinel}, slog.New(slog.DiscardHandler))
	svc.AddObserver(obs)

	_, err := svc.UpdatePolicy(context.Background(), ids.NetworkPolicyID(1), networkpolicies.UpdateFields{
		Name: "p", CIDR: "10.0.0.0/8",
	})

	is.True(errors.Is(err, sentinel))
	is.Equal(obs.calls, 0)
}

// ── DeletePolicy ─────────────────────────────────────────────────────────────

func TestService_DeletePolicy_NotifiesObserver(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	obs := &fakeObserver{}
	svc := newService(repo)
	svc.AddObserver(obs)

	_, err := repo.CreatePolicy(context.Background(), networkpolicies.NetworkPolicy{
		ID: 1, Name: "p", CIDR: "10.0.0.0/8", Enabled: true,
	})
	is.NoErr(err)

	err = svc.DeletePolicy(context.Background(), ids.NetworkPolicyID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_DeletePolicy_RepoError_NoNotify(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	repo.deleteErr = networkpolicies.ErrNotFound
	obs := &fakeObserver{}
	svc := newService(repo)
	svc.AddObserver(obs)

	err := svc.DeletePolicy(context.Background(), ids.NetworkPolicyID(99))

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
	is.Equal(obs.calls, 0)
}

// ── SetHostAccess ─────────────────────────────────────────────────────────────

func TestService_SetHostAccess_NotifiesObserver(t *testing.T) {
	is := is.New(t)
	obs := &fakeObserver{}
	svc := newService(newFakeRepo())
	svc.AddObserver(obs)

	err := svc.SetHostAccess(context.Background(), ids.NetworkPolicyID(1), true, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_SetHostAccess_RepoError_NoNotify(t *testing.T) {
	is := is.New(t)
	repo := newFakeRepo()
	repo.setAccessErr = networkpolicies.ErrNotFound
	obs := &fakeObserver{}
	svc := newService(repo)
	svc.AddObserver(obs)

	err := svc.SetHostAccess(context.Background(), ids.NetworkPolicyID(99), false, nil)

	is.True(errors.Is(err, networkpolicies.ErrNotFound))
	is.Equal(obs.calls, 0)
}
