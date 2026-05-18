//go:build test

package hostaccess

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

	hosts      []Host
	hostsByID  []Host
	hostGroups []HostGroup

	createHostCalls []HostDraft
	deleteHostCalls []ids.HostID
	setGroupCalls   []hostGroupSet

	createCalls []HostGroupDraft
	updateCalls []HostGroup
	deleteCalls []ids.HostGroupID
	callOrder   []string
}

var _ repository = (*fakeRepo)(nil)

func (f *fakeRepo) ListHosts(_ context.Context) ([]Host, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hosts, nil
}
func (f *fakeRepo) CreateHost(_ context.Context, draft HostDraft) (ids.HostID, error) {
	if f.err != nil {
		return ids.HostID(1), f.err
	}
	f.createHostCalls = append(f.createHostCalls, draft)
	f.callOrder = append(f.callOrder, "createHost")
	return ids.HostID(len(f.createHostCalls) + 100), nil
}
func (f *fakeRepo) DeleteHost(_ context.Context, id ids.HostID) error {
	if f.err != nil {
		return f.err
	}
	f.deleteHostCalls = append(f.deleteHostCalls, id)
	f.callOrder = append(f.callOrder, "deleteHost")
	return nil
}

func (f *fakeRepo) SetHostGroupMembership(_ context.Context, hostID ids.HostID, groupIDs []ids.HostGroupID) error {
	if f.err != nil {
		return f.err
	}
	f.setGroupCalls = append(f.setGroupCalls, hostGroupSet{HostID: hostID, GroupIDs: groupIDs})
	f.callOrder = append(f.callOrder, "setHostGroups")
	return nil
}

func (f *fakeRepo) ListHostsByIDs(_ context.Context, _ []ids.HostID) ([]Host, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hostsByID, nil
}

func (f *fakeRepo) ListHostGroups(_ context.Context) ([]HostGroup, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hostGroups, nil
}
func (f *fakeRepo) CreateHostGroup(_ context.Context, draft HostGroupDraft) (ids.HostGroupID, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.createCalls = append(f.createCalls, draft)
	f.callOrder = append(f.callOrder, "create")
	return ids.HostGroupID(len(f.createCalls) + 100), nil
}
func (f *fakeRepo) UpdateHostGroup(_ context.Context, group HostGroup) error {
	if f.err != nil {
		return f.err
	}
	f.updateCalls = append(f.updateCalls, group)
	f.callOrder = append(f.callOrder, "update")
	return nil
}
func (f *fakeRepo) DeleteHostGroup(_ context.Context, id ids.HostGroupID) error {
	if f.err != nil {
		return f.err
	}
	f.deleteCalls = append(f.deleteCalls, id)
	f.callOrder = append(f.callOrder, "delete")
	return nil
}

func (f *fakeRepo) SetUserAccess(_ context.Context, _ ids.UserID, _ bool, _ []ids.HostGroupID) error {
	return f.err
}

func (f *fakeRepo) AddIgnoredSuggestion(_ context.Context, _ string) (IgnoredHostSuggestion, error) {
	if f.err != nil {
		return IgnoredHostSuggestion{}, f.err
	}
	return IgnoredHostSuggestion{ID: 1}, nil
}
func (f *fakeRepo) RemoveIgnoredSuggestionByFQDN(_ context.Context, _ string) error { return f.err }

func (f *fakeRepo) EnsureUserSettings(_ context.Context, _ ids.UserID) error { return f.err }
func (f *fakeRepo) DeleteUserData(_ context.Context, _ ids.UserID) error     { return f.err }

func (f *fakeRepo) GetAllUserHostSettings(_ context.Context) ([]UserHostSetting, error) {
	return f.settings, f.err
}
func (f *fakeRepo) GetAllUserHostGrants(_ context.Context) ([]UserHostGrant, error) {
	return f.grants, f.err
}

// fakeTransactor executes fn directly and records whether it was called.
type fakeTransactor struct {
	calls int
}

func (f *fakeTransactor) WithinTx(_ context.Context, fn func(context.Context) error) error {
	f.calls++
	return fn(context.Background())
}

// mockObserver records calls to OnHostAccessChanged.
type mockObserver struct {
	calls int
}

var _ Observer = (*mockObserver)(nil)

func (m *mockObserver) OnHostAccessChanged(_ context.Context) { m.calls++ }

func newTestService(repo repository) (*Service, *fakeTransactor) {
	tx := &fakeTransactor{}
	return NewService(repo, tx, slog.New(slog.DiscardHandler)), tx
}

func TestService_SetUserAccess_NotifiesObserver(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	err := svc.SetUserAccess(context.Background(), ids.UserID(1), true, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_AddObserver_NilObserver_Ignored(t *testing.T) {
	is := is.New(t)
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(nil)

	is.Equal(len(svc.userHostAccessObservers), 0)
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
