//go:build test

package hostaccess

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	createKnownHostFn func(ctx context.Context, fqdn string, icon *string) (KnownHost, error)
	deleteKnownHostFn func(ctx context.Context, id KnownHostID) error
	grantUserHostFn   func(ctx context.Context, userID auth.UserID, hostID KnownHostID) error
	revokeUserHostFn  func(ctx context.Context, userID auth.UserID, hostID KnownHostID) error
	err               error
}

var _ repository = (*fakeRepo)(nil)

func (f *fakeRepo) CreateKnownHost(ctx context.Context, fqdn string, icon *string) (KnownHost, error) {
	if f.createKnownHostFn != nil {
		return f.createKnownHostFn(ctx, fqdn, icon)
	}
	if f.err != nil {
		return KnownHost{}, f.err
	}
	return KnownHost{ID: 1, FQDN: fqdn}, nil
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
func (f *fakeRepo) GetKnownHost(_ context.Context, _ KnownHostID) (KnownHost, error) {
	return KnownHost{}, f.err
}
func (f *fakeRepo) ListKnownHosts(_ context.Context) ([]KnownHost, error) {
	return []KnownHost{}, f.err
}
func (f *fakeRepo) UpdateKnownHost(_ context.Context, id KnownHostID, icon *string) (KnownHost, error) {
	if f.err != nil {
		return KnownHost{}, f.err
	}
	return KnownHost{ID: id, Icon: icon}, nil
}
func (f *fakeRepo) DeleteKnownHost(ctx context.Context, id KnownHostID) error {
	if f.deleteKnownHostFn != nil {
		return f.deleteKnownHostFn(ctx, id)
	}
	return f.err
}

func (f *fakeRepo) CreateHostGroup(_ context.Context, _ string, _ *string, _ *string) (HostGroup, error) {
	if f.err != nil {
		return HostGroup{}, f.err
	}
	return HostGroup{ID: 1, Name: "g"}, nil
}
func (f *fakeRepo) GetHostGroup(_ context.Context, _ HostGroupID) (HostGroup, error) {
	return HostGroup{}, f.err
}
func (f *fakeRepo) ListHostGroups(_ context.Context) ([]HostGroup, error) {
	return []HostGroup{}, f.err
}
func (f *fakeRepo) ListHostGroupsWithMembers(_ context.Context) ([]HostGroupWithMembers, error) {
	return []HostGroupWithMembers{}, f.err
}
func (f *fakeRepo) UpdateHostGroup(_ context.Context, id HostGroupID, name string, _ *string, _ *string) (HostGroup, error) {
	if f.err != nil {
		return HostGroup{}, f.err
	}
	return HostGroup{ID: id, Name: name}, nil
}
func (f *fakeRepo) DeleteHostGroup(_ context.Context, _ HostGroupID) error { return f.err }

func (f *fakeRepo) AddHostToGroup(_ context.Context, _ HostGroupID, _ KnownHostID) error {
	return f.err
}
func (f *fakeRepo) RemoveHostFromGroup(_ context.Context, _ HostGroupID, _ KnownHostID) error {
	return f.err
}
func (f *fakeRepo) ListHostGroupMembers(_ context.Context, _ HostGroupID) ([]KnownHost, error) {
	return []KnownHost{}, f.err
}
func (f *fakeRepo) SetHostGroupMembers(_ context.Context, _ HostGroupID, _ []KnownHostID) error {
	return f.err
}

func (f *fakeRepo) GrantUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error {
	if f.grantUserHostFn != nil {
		return f.grantUserHostFn(ctx, userID, hostID)
	}
	return f.err
}
func (f *fakeRepo) RevokeUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error {
	if f.revokeUserHostFn != nil {
		return f.revokeUserHostFn(ctx, userID, hostID)
	}
	return f.err
}
func (f *fakeRepo) GrantUserHostGroup(_ context.Context, _ auth.UserID, _ HostGroupID) error {
	return f.err
}
func (f *fakeRepo) RevokeUserHostGroup(_ context.Context, _ auth.UserID, _ HostGroupID) error {
	return f.err
}
func (f *fakeRepo) ListUserGrants(_ context.Context, _ auth.UserID) ([]KnownHost, []HostGroup, error) {
	return []KnownHost{}, []HostGroup{}, f.err
}
func (f *fakeRepo) SetUserGrants(_ context.Context, _ auth.UserID, _ []KnownHostID, _ []HostGroupID) error {
	return f.err
}
func (f *fakeRepo) SetUserBypassAllowlist(_ context.Context, _ auth.UserID, _ bool) error {
	return f.err
}

func (f *fakeRepo) AddIgnoredSuggestion(_ context.Context, _ string) (IgnoredHostSuggestion, error) {
	if f.err != nil {
		return IgnoredHostSuggestion{}, f.err
	}
	return IgnoredHostSuggestion{ID: 1}, nil
}
func (f *fakeRepo) FindIgnoredSuggestionByFQDN(_ context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	if f.err != nil {
		return IgnoredHostSuggestion{}, f.err
	}
	return IgnoredHostSuggestion{ID: 1, FQDN: fqdn}, nil
}
func (f *fakeRepo) RemoveIgnoredSuggestion(_ context.Context, _ int64) error { return f.err }
func (f *fakeRepo) ListIgnoredSuggestions(_ context.Context) ([]IgnoredHostSuggestion, error) {
	return []IgnoredHostSuggestion{}, f.err
}
func (f *fakeRepo) GetUserBypassAllowlist(_ context.Context, _ auth.UserID) (bool, error) {
	return false, f.err
}

func (f *fakeRepo) GetAllUserHostAccess(_ context.Context) ([]policy.UserHostAccess, error) {
	return nil, f.err
}

// mockObserver records calls to OnHostAccessChanged.
type mockObserver struct {
	calls int
}

var _ Observer = (*mockObserver)(nil)

func (m *mockObserver) OnHostAccessChanged(_ context.Context) { m.calls++ }

func newTestService(repo repository) *Service {
	return NewService(repo, slog.New(slog.DiscardHandler))
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_CreateKnownHost_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	_, err := svc.CreateKnownHost(context.Background(), "example.com", nil)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_CreateKnownHost_RepoError_NoNotification(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{err: errors.New("db")})
	svc.AddObserver(obs)

	_, err := svc.CreateKnownHost(context.Background(), "example.com", nil)

	is.True(err != nil)
	is.Equal(obs.calls, 0)
}

func TestService_DeleteKnownHost_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	err := svc.DeleteKnownHost(context.Background(), KnownHostID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_GrantUserHost_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	err := svc.GrantUserHost(context.Background(), auth.UserID(1), KnownHostID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_RevokeUserHost_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	err := svc.RevokeUserHost(context.Background(), auth.UserID(1), KnownHostID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_AddObserver_NilObserver_Ignored(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(nil)

	is.Equal(len(svc.observers), 0)
}

func TestService_ListKnownHosts_DoesNotNotify(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	_, err := svc.ListKnownHosts(context.Background())

	is.NoErr(err)
	is.Equal(obs.calls, 0)
}

func TestService_MultipleObservers_AllNotified(t *testing.T) {
	is := is.New(t)
	obs1 := &mockObserver{}
	obs2 := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs1)
	svc.AddObserver(obs2)

	_, err := svc.CreateKnownHost(context.Background(), "example.com", nil)

	is.NoErr(err)
	is.Equal(obs1.calls, 1)
	is.Equal(obs2.calls, 1)
}
