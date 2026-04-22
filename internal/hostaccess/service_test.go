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
	err error
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

func TestService_BulkCreateKnownHosts_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	_, err := svc.BulkCreateKnownHosts(context.Background(), []string{"example.com"})

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_BulkCreateKnownHosts_RepoError_NoNotification(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{err: errors.New("db")})
	svc.AddObserver(obs)

	_, err := svc.BulkCreateKnownHosts(context.Background(), []string{"example.com"})

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

func TestService_CreateHostGroup_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	_, err := svc.CreateHostGroup(context.Background(), "g", nil, nil, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_UpdateHostGroup_WithMembers_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	hostIDs := []KnownHostID{1, 2}
	err := svc.UpdateHostGroup(context.Background(), HostGroupID(1), "g", nil, nil, &hostIDs)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_UpdateHostGroup_MetadataOnly_NoNotification(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	err := svc.UpdateHostGroup(context.Background(), HostGroupID(1), "g", nil, nil, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 0)
}

func TestService_DeleteHostGroup_NotifiesObservers(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	err := svc.DeleteHostGroup(context.Background(), HostGroupID(1))

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_SetFullUserGrants_NotifiesObserversOnce(t *testing.T) {
	is := is.New(t)
	obs := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs)

	bypass := true
	err := svc.SetFullUserGrants(context.Background(), auth.UserID(1), &bypass, nil, nil)

	is.NoErr(err)
	is.Equal(obs.calls, 1)
}

func TestService_AddObserver_NilObserver_Ignored(t *testing.T) {
	is := is.New(t)
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(nil)

	is.Equal(len(svc.observers), 0)
}

func TestService_MultipleObservers_AllNotified(t *testing.T) {
	is := is.New(t)
	obs1 := &mockObserver{}
	obs2 := &mockObserver{}
	svc := newTestService(&fakeRepo{})
	svc.AddObserver(obs1)
	svc.AddObserver(obs2)

	_, err := svc.BulkCreateKnownHosts(context.Background(), []string{"example.com"})

	is.NoErr(err)
	is.Equal(obs1.calls, 1)
	is.Equal(obs2.calls, 1)
}
