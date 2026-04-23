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
}

var _ repository = (*fakeRepo)(nil)

func (f *fakeRepo) BulkCreateKnownHosts(_ context.Context, fqdns []string) ([]KnownHost, error) {
	if f.err != nil {
		return nil, f.err
	}
	hosts := make([]KnownHost, len(fqdns))
	for i, fqdn := range fqdns {
		hosts[i] = KnownHost{ID: KnownHostID(i + 1), FQDN: fqdn}
	}
	return hosts, nil
}
func (f *fakeRepo) UpdateKnownHost(_ context.Context, id KnownHostID, icon *string) (KnownHost, error) {
	if f.err != nil {
		return KnownHost{}, f.err
	}
	return KnownHost{ID: id, Icon: icon}, nil
}
func (f *fakeRepo) DeleteKnownHost(_ context.Context, _ KnownHostID) error { return f.err }

func (f *fakeRepo) CreateHostGroupWithMembers(_ context.Context, _ string, _ *string, _ *string, _ []KnownHostID) (HostGroupID, error) {
	if f.err != nil {
		return 0, f.err
	}
	return HostGroupID(1), nil
}
func (f *fakeRepo) UpdateHostGroupWithMembers(_ context.Context, _ HostGroupID, _ string, _ *string, _ *string, _ []KnownHostID) error {
	return f.err
}
func (f *fakeRepo) UpdateHostGroupMetadata(_ context.Context, _ HostGroupID, _ string, _ *string, _ *string) error {
	return f.err
}
func (f *fakeRepo) DeleteHostGroup(_ context.Context, _ HostGroupID) error { return f.err }

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

// ── Observer notification tests ───────────────────────────────────────────────

func TestService_BulkCreateKnownHosts_DoesNotNotifyObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	_, err := svc.BulkCreateKnownHosts(context.Background(), []string{"example.com"})

	is.NoErr(err)
	is.Equal(obs.calls, 0) // no user access change from creating unassigned hosts
}

func TestService_BulkCreateKnownHosts_RepoError_NoNotification(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{err: errors.New("db")})
	svc.AddUserHostAccessObserver(obs)

	_, err := svc.BulkCreateKnownHosts(context.Background(), []string{"example.com"})

	is.True(err != nil)
	is.Equal(obs.calls, 0)
}

func TestService_DeleteKnownHost_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	err := svc.DeleteKnownHost(context.Background(), KnownHostID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_CreateHostGroup_DoesNotNotifyObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	_, err := svc.CreateHostGroup(context.Background(), "g", nil, nil, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 0) // no user access change from creating a group with no users
}

func TestService_UpdateHostGroup_WithMembers_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	hostIDs := []KnownHostID{1, 2}
	err := svc.UpdateHostGroup(context.Background(), HostGroupID(1), "g", nil, nil, &hostIDs)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_UpdateHostGroup_MetadataOnly_NoNotification(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	err := svc.UpdateHostGroup(context.Background(), HostGroupID(1), "g", nil, nil, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 0)
}

func TestService_DeleteHostGroup_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs)

	err := svc.DeleteHostGroup(context.Background(), HostGroupID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
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

func TestService_MultipleObservers_AllNotified(t *testing.T) {
	is := is.New(t)
	obs1 := &mockObserver{}
	obs2 := &mockObserver{}
	svc, _ := newTestService(&fakeRepo{})
	svc.AddUserHostAccessObserver(obs1)
	svc.AddUserHostAccessObserver(obs2)

	err := svc.DeleteKnownHost(context.Background(), KnownHostID(1))

	is.NoErr(err)
	is.Equal(obs1.calls, 1)
	is.Equal(obs2.calls, 1)
}

// ── GetAllUserHostAccess merge tests ──────────────────────────────────────────

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
