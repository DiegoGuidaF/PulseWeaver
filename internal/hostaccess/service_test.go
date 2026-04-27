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
func (f *fakeRepo) CreateKnownHost(_ context.Context, draft KnownHostDraft) error {
	if f.err != nil {
		return f.err
	}
	f.createHostCalls = append(f.createHostCalls, draft)
	f.callOrder = append(f.callOrder, "createHost")
	return nil
}
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

func TestService_ReconcileHostGroups_NotifiesObserversOnce(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	repo := &fakeRepo{
		hostGroups:     []HostGroup{{ID: 1, Name: "old"}},
		knownHostsByID: nil,
	}
	svc, _ := newTestService(repo)
	svc.AddUserHostAccessObserver(obs)

	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{{Name: "new"}},
	})

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_ReconcileHostGroups_DeleteThenUpdateThenCreate(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{
		hostGroups: []HostGroup{
			{ID: 1, Name: "to-update"},
			{ID: 2, Name: "to-delete"},
		},
	}
	svc, _ := newTestService(repo)

	id1 := HostGroupID(1)
	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{
			{ID: &id1, Name: "renamed"},
			{Name: "new-one"},
		},
	})
	is.NoErr(err)
	is.Equal(repo.callOrder, []string{"delete", "update", "create"})
}

func TestService_ReconcileHostGroups_NoOp_NoCalls(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	current := HostGroup{ID: 1, Name: "stable"}
	repo := &fakeRepo{hostGroups: []HostGroup{current}}
	svc, _ := newTestService(repo)
	svc.AddUserHostAccessObserver(obs)

	id := current.ID
	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{{ID: &id, Name: "stable"}},
	})

	is.NoErr(err)
	is.Equal(len(repo.callOrder), 0)
	is.Equal(obs.calls, 1) // observers still notified after a successful tx, even if it was a no-op
}

func TestService_ReconcileHostGroups_EmptyName_Rejected(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	svc, _ := newTestService(repo)

	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{{Name: "  "}},
	})
	is.True(errors.Is(err, ErrGroupNameRequired))
	is.Equal(len(repo.callOrder), 0)
}

func TestService_ReconcileHostGroups_DuplicateID_Rejected(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	svc, _ := newTestService(repo)

	id := HostGroupID(7)
	err := svc.ReconcileHostGroups(context.Background(), ReconcileHostGroupsInput{
		Groups: []DesiredHostGroup{
			{ID: &id, Name: "a"},
			{ID: &id, Name: "b"},
		},
	})
	is.True(errors.Is(err, ErrDuplicateGroupID))
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

// ── buildGroupReconcilePlan (pure function) ───────────────────────────────────

func TestBuildGroupReconcilePlan_CreateOnly(t *testing.T) {
	is := is.New(t)
	plan, err := buildGroupReconcilePlan(nil, []DesiredHostGroup{{Name: "new"}})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].Name, "new")
	is.Equal(len(plan.toUpdate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildGroupReconcilePlan_DeleteOnly(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{{ID: 1, Name: "doomed"}}
	plan, err := buildGroupReconcilePlan(current, nil)
	is.NoErr(err)
	is.Equal(plan.toDelete, []HostGroupID{1})
}

func TestBuildGroupReconcilePlan_UpdateChanged(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{{ID: 1, Name: "before"}}
	id := HostGroupID(1)
	plan, err := buildGroupReconcilePlan(current, []DesiredHostGroup{{ID: &id, Name: "after"}})
	is.NoErr(err)
	is.Equal(len(plan.toUpdate), 1)
	is.Equal(plan.toUpdate[0].Name, "after")
}

func TestBuildGroupReconcilePlan_UpdateUnchanged_SkipsUpdate(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{{ID: 1, Name: "stable"}}
	id := HostGroupID(1)
	plan, err := buildGroupReconcilePlan(current, []DesiredHostGroup{{ID: &id, Name: "stable"}})
	is.NoErr(err)
	is.Equal(len(plan.toUpdate), 0)
	is.Equal(len(plan.toCreate), 0)
	is.Equal(len(plan.toDelete), 0)
}

func TestBuildGroupReconcilePlan_UnknownDesiredID_Errors(t *testing.T) {
	is := is.New(t)
	id := HostGroupID(42)
	_, err := buildGroupReconcilePlan(nil, []DesiredHostGroup{{ID: &id, Name: "ghost"}})
	is.True(errors.Is(err, ErrHostGroupNotFound))
}

func TestBuildGroupReconcilePlan_Mixed(t *testing.T) {
	is := is.New(t)
	current := []HostGroup{
		{ID: 1, Name: "keep"},
		{ID: 2, Name: "rename-me"},
		{ID: 3, Name: "remove-me"},
	}
	id1 := HostGroupID(1)
	id2 := HostGroupID(2)
	plan, err := buildGroupReconcilePlan(current, []DesiredHostGroup{
		{ID: &id1, Name: "keep"},
		{ID: &id2, Name: "renamed"},
		{Name: "fresh"},
	})
	is.NoErr(err)
	is.Equal(len(plan.toCreate), 1)
	is.Equal(plan.toCreate[0].Name, "fresh")
	is.Equal(len(plan.toUpdate), 1)
	is.Equal(plan.toUpdate[0].ID, HostGroupID(2))
	is.Equal(plan.toDelete, []HostGroupID{3})
}
