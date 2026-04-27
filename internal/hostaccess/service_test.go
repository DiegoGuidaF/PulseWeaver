//go:build test

package hostaccess

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	err          error
	settings     []UserHostSetting
	directGrants []UserHostGrant
	groupGrants  []UserHostGrant

	knownHosts     []KnownHost
	knownHostsByID []KnownHost
	hostGroups     []HostGroup

	createHostCalls []KnownHostDraft
	updateHostCalls []KnownHost
	deleteHostCalls []KnownHostID
	setGroupCalls   []knownHostGroupSet

	createCalls []HostGroupDraft
	updateCalls []HostGroup
	deleteCalls []HostGroupID
	callOrder   []string
}

var _ repository = (*fakeRepo)(nil)

func (f *fakeRepo) ListKnownHosts(_ context.Context) ([]KnownHost, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.knownHosts, nil
}
func (f *fakeRepo) CreateKnownHost(_ context.Context, draft KnownHostDraft) (KnownHostID, error) {
	if f.err != nil {
		return KnownHostID(1), f.err
	}
	f.createHostCalls = append(f.createHostCalls, draft)
	f.callOrder = append(f.callOrder, "createHost")
	return KnownHostID(len(f.createHostCalls) + 100), nil
}
func (f *fakeRepo) UpdateKnownHost(_ context.Context, id KnownHostID, icon *string) (KnownHost, error) {
	if f.err != nil {
		return KnownHost{}, f.err
	}
	h := KnownHost{ID: id, Icon: icon}
	f.updateHostCalls = append(f.updateHostCalls, h)
	f.callOrder = append(f.callOrder, "updateHost")
	return h, nil
}
func (f *fakeRepo) DeleteKnownHost(_ context.Context, id KnownHostID) error {
	if f.err != nil {
		return f.err
	}
	f.deleteHostCalls = append(f.deleteHostCalls, id)
	f.callOrder = append(f.callOrder, "deleteHost")
	return nil
}

func (f *fakeRepo) SetKnownHostGroupMembership(_ context.Context, hostID KnownHostID, groupIDs []HostGroupID) error {
	if f.err != nil {
		return f.err
	}
	f.setGroupCalls = append(f.setGroupCalls, knownHostGroupSet{HostID: hostID, GroupIDs: groupIDs})
	f.callOrder = append(f.callOrder, "setHostGroups")
	return nil
}

func (f *fakeRepo) ListKnownHostsByIDs(_ context.Context, _ []KnownHostID) ([]KnownHost, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.knownHostsByID, nil
}

func (f *fakeRepo) ListHostGroups(_ context.Context) ([]HostGroup, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.hostGroups, nil
}
func (f *fakeRepo) CreateHostGroup(_ context.Context, draft HostGroupDraft) (HostGroupID, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.createCalls = append(f.createCalls, draft)
	f.callOrder = append(f.callOrder, "create")
	return HostGroupID(len(f.createCalls) + 100), nil
}
func (f *fakeRepo) UpdateHostGroup(_ context.Context, group HostGroup) error {
	if f.err != nil {
		return f.err
	}
	f.updateCalls = append(f.updateCalls, group)
	f.callOrder = append(f.callOrder, "update")
	return nil
}
func (f *fakeRepo) DeleteHostGroup(_ context.Context, id HostGroupID) error {
	if f.err != nil {
		return f.err
	}
	f.deleteCalls = append(f.deleteCalls, id)
	f.callOrder = append(f.callOrder, "delete")
	return nil
}

func (f *fakeRepo) SetFullUserGrants(_ context.Context, _ auth.UserID, _ *bool, _ []KnownHostID, _ []HostGroupID) error {
	return f.err
}

func (f *fakeRepo) AddIgnoredSuggestion(_ context.Context, _ string) (IgnoredHostSuggestion, error) {
	if f.err != nil {
		return IgnoredHostSuggestion{}, f.err
	}
	return IgnoredHostSuggestion{ID: 1}, nil
}
func (f *fakeRepo) RemoveIgnoredSuggestionByFQDN(_ context.Context, _ string) error { return f.err }

func (f *fakeRepo) EnsureUserSettings(_ context.Context, _ auth.UserID) error { return f.err }
func (f *fakeRepo) DeleteUserData(_ context.Context, _ auth.UserID) error     { return f.err }

func (f *fakeRepo) GetAllUserHostSettings(_ context.Context) ([]UserHostSetting, error) {
	return f.settings, f.err
}
func (f *fakeRepo) GetAllUserDirectHostGrants(_ context.Context) ([]UserHostGrant, error) {
	return f.directGrants, f.err
}
func (f *fakeRepo) GetAllUserGroupHostGrants(_ context.Context) ([]UserHostGrant, error) {
	return f.groupGrants, f.err
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

func TestService_SetFullUserGrants_NotifiesObserversOnce(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	bypass := true
	err := svc.SetFullUserGrants(context.Background(), auth.UserID(1), &bypass, nil, nil)

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

func TestService_GetAllUserHostAccess_DirectGrant(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings:     []UserHostSetting{{UserID: 1, BypassAllowlist: false}},
		directGrants: []UserHostGrant{{UserID: 1, FQDN: "example.com"}},
	}
	svc, tx := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(tx.calls, 1)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, auth.UserID(1))
	is.True(!result[0].BypassAllowlist)
	is.Equal(result[0].AllowedHosts, []string{"example.com"})
}

func TestService_GetAllUserHostAccess_GroupGrant(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings:    []UserHostSetting{{UserID: 1, BypassAllowlist: false}},
		groupGrants: []UserHostGrant{{UserID: 1, FQDN: "group-host.com"}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].AllowedHosts, []string{"group-host.com"})
}

func TestService_GetAllUserHostAccess_BypassOnly(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassAllowlist: true}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].UserID, auth.UserID(1))
	is.True(result[0].BypassAllowlist)
	is.Equal(len(result[0].AllowedHosts), 0)
}

func TestService_GetAllUserHostAccess_BypassWithHosts(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings:     []UserHostSetting{{UserID: 1, BypassAllowlist: true}},
		directGrants: []UserHostGrant{{UserID: 1, FQDN: "a.com"}},
		groupGrants:  []UserHostGrant{{UserID: 1, FQDN: "b.com"}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.True(result[0].BypassAllowlist)
	sort.Strings(result[0].AllowedHosts)
	is.Equal(result[0].AllowedHosts, []string{"a.com", "b.com"})
}

func TestService_GetAllUserHostAccess_ExcludesDenyAllUsers(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassAllowlist: false}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 0)
}

func TestService_GetAllUserHostAccess_DeduplicatesDirectAndGroup(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings:     []UserHostSetting{{UserID: 1, BypassAllowlist: false}},
		directGrants: []UserHostGrant{{UserID: 1, FQDN: "shared.com"}},
		groupGrants:  []UserHostGrant{{UserID: 1, FQDN: "shared.com"}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 1)
	is.Equal(result[0].AllowedHosts, []string{"shared.com"})
}

func TestService_GetAllUserHostAccess_DeduplicatesMultipleGroups(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{{UserID: 1, BypassAllowlist: false}},
		groupGrants: []UserHostGrant{
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

func TestService_GetAllUserHostAccess_IgnoresOrphanedGrants(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings:     []UserHostSetting{{UserID: 1, BypassAllowlist: false}},
		directGrants: []UserHostGrant{{UserID: 99, FQDN: "orphan.com"}},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	// User 1 has no hosts → excluded. User 99 is orphaned → ignored.
	is.Equal(len(result), 0)
}

func TestService_GetAllUserHostAccess_MultipleUsers(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		settings: []UserHostSetting{
			{UserID: 1, BypassAllowlist: false},
			{UserID: 2, BypassAllowlist: true},
			{UserID: 3, BypassAllowlist: false},
		},
		directGrants: []UserHostGrant{
			{UserID: 1, FQDN: "a.com"},
			{UserID: 3, FQDN: "b.com"},
		},
		groupGrants: []UserHostGrant{
			{UserID: 1, FQDN: "c.com"},
		},
	}
	svc, _ := newTestService(repo)

	result, err := svc.GetAllUserHostAccess(context.Background())
	is.NoErr(err)
	is.Equal(len(result), 3)

	byUser := make(map[auth.UserID]policy.UserHostAccess)
	for _, r := range result {
		sort.Strings(r.AllowedHosts)
		byUser[r.UserID] = r
	}

	is.Equal(byUser[auth.UserID(1)].AllowedHosts, []string{"a.com", "c.com"})
	is.True(byUser[auth.UserID(2)].BypassAllowlist)
	is.Equal(len(byUser[auth.UserID(2)].AllowedHosts), 0)
	is.Equal(byUser[auth.UserID(3)].AllowedHosts, []string{"b.com"})
}

func TestService_GetAllUserHostAccess_RepoError(t *testing.T) {
	is := is.New(t)
	svc, _ := newTestService(&fakeRepo{err: errors.New("db")})

	_, err := svc.GetAllUserHostAccess(context.Background())
	is.True(err != nil)
}
