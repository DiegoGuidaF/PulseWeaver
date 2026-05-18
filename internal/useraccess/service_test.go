//go:build test

package useraccess

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	err      error
	settings []UserHostSetting
	grants   []UserHostGrant
}

var _ repository = (*fakeRepo)(nil)

func (f *fakeRepo) SetUserAccess(_ context.Context, _ ids.UserID, _ bool, _ []ids.HostGroupID) error {
	return f.err
}
func (f *fakeRepo) EnsureUserSettings(_ context.Context, _ ids.UserID) error { return f.err }
func (f *fakeRepo) DeleteUserData(_ context.Context, _ ids.UserID) error     { return f.err }
func (f *fakeRepo) GetAllUserHostSettings(_ context.Context) ([]UserHostSetting, error) {
	return f.settings, f.err
}
func (f *fakeRepo) GetAllUserHostGrants(_ context.Context) ([]UserHostGrant, error) {
	return f.grants, f.err
}

type fakeTransactor struct {
	calls int
}

func (f *fakeTransactor) WithinTx(_ context.Context, fn func(context.Context) error) error {
	f.calls++
	return fn(context.Background())
}

type mockObserver struct {
	calls int
}

var _ Observer = (*mockObserver)(nil)

func (m *mockObserver) OnHostAccessChanged(_ context.Context) { m.calls++ }

func newTestService(repo repository) (*Service, *fakeTransactor) {
	tx := &fakeTransactor{}
	return NewService(repo, tx, slog.New(slog.DiscardHandler)), tx
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_SetUserAccess_NotifiesObserver(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	err := svc.SetUserAccess(context.Background(), ids.UserID(1), true, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_AddObserver_NilObserver_Ignored(t *testing.T) {
	is := is.New(t)
	svc, _ := newTestService(&fakeRepo{})
	svc.AddObserver(nil)

	is.Equal(len(svc.observers), 0)
}

func TestService_GetAllUserHostAccess_Empty(t *testing.T) {
	is := is.New(t)
	svc, tx := newTestService(&fakeRepo{})

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
	is.Equal(tx.calls, 1)
}

func TestService_GetAllUserHostAccess_GroupGrant(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassHostCheck: false}},
		grants:   []UserHostGrant{{UserID: 1, FQDN: "group-host.com"}},
	}
	svc, tx := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(tx.calls, 1)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, ids.UserID(1))
	is.True(!result[0].BypassAllowlist)
	is.Equal(result[0].AllowedHosts, []string{"group-host.com"})
}

func TestService_GetAllUserHostAccess_BypassOnly(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassHostCheck: true}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, ids.UserID(1))
	is.True(result[0].BypassAllowlist)
	is.Equal(len(result[0].AllowedHosts), 0)
}

func TestService_GetAllUserHostAccess_BypassWithHosts(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassHostCheck: true}},
		grants:   []UserHostGrant{{UserID: 1, FQDN: "b.com"}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.True(result[0].BypassAllowlist)
	is.Equal(result[0].AllowedHosts, []string{"b.com"})
}

func TestService_GetAllUserHostAccess_ExcludesDenyAllUsers(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassHostCheck: false}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestService_GetAllUserHostAccess_DeduplicatesAcrossGroups(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassHostCheck: false}},
		grants: []UserHostGrant{
			{UserID: 1, FQDN: "shared.com"},
			{UserID: 1, FQDN: "shared.com"},
			{UserID: 1, FQDN: "unique.com"},
		},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)

	sort.Strings(result[0].AllowedHosts)
	is.Equal(result[0].AllowedHosts, []string{"shared.com", "unique.com"})
}

func TestService_GetAllUserHostAccess_MultipleUsers(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{
			{UserID: 1, BypassHostCheck: false},
			{UserID: 2, BypassHostCheck: true},
			{UserID: 3, BypassHostCheck: false},
		},
		grants: []UserHostGrant{
			{UserID: 1, FQDN: "a.com"},
			{UserID: 1, FQDN: "c.com"},
			{UserID: 3, FQDN: "b.com"},
		},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 3)

	byUser := make(map[ids.UserID]policy.UserHostAccess)
	for _, r := range result {
		sort.Strings(r.AllowedHosts)
		byUser[r.UserID] = r
	}

	is.Equal(byUser[ids.UserID(1)].AllowedHosts, []string{"a.com", "c.com"})
	is.True(byUser[ids.UserID(2)].BypassAllowlist)
	is.Equal(len(byUser[ids.UserID(2)].AllowedHosts), 0)
	is.Equal(byUser[ids.UserID(3)].AllowedHosts, []string{"b.com"})
}

func TestService_GetAllUserHostAccess_RepoError(t *testing.T) {
	is := is.New(t)
	svc, _ := newTestService(&fakeRepo{err: errors.New("db")})

	_, err := svc.GetAllUserHostAccess(context.Background())
	is.True(err != nil)
}
