//go:build test

package registration

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/matryer/is"
)

// --- mock repository ---

type mockRepo struct {
	invites     map[PendingRegistrationID]*PendingRegistration
	claimResult *ClaimResult
	claimErr    error
}

func newMockRepo() *mockRepo {
	return &mockRepo{invites: make(map[PendingRegistrationID]*PendingRegistration)}
}

var _ repository = (*mockRepo)(nil)

func (m *mockRepo) CreateInvite(_ context.Context, p CreateInviteRequest) (*PendingRegistration, error) {
	pendReg := &PendingRegistration{
		ID:               PendingRegistrationID(len(m.invites) + 1),
		DeviceName:       p.DeviceName,
		RegistrationCode: new(p.RegistrationCode),
		ExpiresAt:        p.ExpiresAt,
	}

	m.invites[pendReg.ID] = pendReg
	return pendReg, nil
}

func (m *mockRepo) GetInvite(_ context.Context, id PendingRegistrationID) (*PendingRegistration, error) {
	p, ok := m.invites[id]
	if !ok {
		return nil, ErrInviteNotFound
	}
	return p, nil
}

func (m *mockRepo) ListInvites(_ context.Context, filter InviteFilter) ([]PendingRegistration, error) {
	var result []PendingRegistration
	for _, p := range m.invites {
		if !filter.IncludeAll && p.Status() != StatusPending {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockRepo) InvalidateInvite(_ context.Context, id PendingRegistrationID) error {
	p, ok := m.invites[id]
	if !ok {
		return ErrInviteNotFound
	}
	if p.UsedAt != nil || p.InvalidatedAt != nil {
		return ErrInviteNotPending
	}
	now := time.Now()
	p.InvalidatedAt = &now
	return nil
}

func (m *mockRepo) ClaimInvite(_ context.Context, _ string) (*ClaimResult, error) {
	return m.claimResult, m.claimErr
}

// --- tests ---

func newTestService(repo repository) *Service {
	return NewService(repo, slog.New(slog.DiscardHandler))
}

func TestService_CreateInvite_StoresInvite(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	invite, err := svc.CreateInvite(context.Background(), CreateInviteRequest{
		DeviceName:         "Dad's Phone",
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	})
	is.NoErr(err)
	is.True(invite != nil)
	is.Equal(invite.DeviceName, "Dad's Phone")
	is.True(invite.RegistrationCode != nil && *invite.RegistrationCode != "")
	is.Equal(invite.Status(), StatusPending)

	// Verify persisted in repo
	stored, err := repo.GetInvite(context.Background(), invite.ID)
	is.NoErr(err)
	is.Equal(stored.DeviceName, "Dad's Phone")
}

func TestService_CreateInvite_ExpirySetCorrectly(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	before := time.Now().Add(23 * time.Hour)
	invite, err := svc.CreateInvite(context.Background(), CreateInviteRequest{
		DeviceName:         "Test Device",
		HeartbeatServerURL: "https://example.com",
		IntervalSeconds:    300,
		ExpiresInHours:     24,
	})
	after := time.Now().Add(25 * time.Hour)

	is.NoErr(err)
	is.True(invite.ExpiresAt.After(before))
	is.True(invite.ExpiresAt.Before(after))
}

func TestService_GetInvite_ReturnsNotFound(t *testing.T) {
	is := is.New(t)
	svc := newTestService(newMockRepo())

	_, err := svc.GetInvite(context.Background(), PendingRegistrationID(0))
	is.True(errors.Is(err, ErrInviteNotFound))
}

func TestService_ListInvites_PendingOnly(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	invalidatedAt := now.Add(-30 * time.Minute)

	repo.invites[PendingRegistrationID(0)] = &PendingRegistration{
		ID: 0, DeviceName: "Active",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
	}
	repo.invites[PendingRegistrationID(1)] = &PendingRegistration{
		ID: 1, DeviceName: "Used",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UsedAt: new(now.Add(-1 * time.Hour)),
	}
	repo.invites[PendingRegistrationID(2)] = &PendingRegistration{
		ID: 2, DeviceName: "Expired",
		ExpiresAt: now.Add(-1 * time.Hour), CreatedAt: now.Add(-2 * time.Hour),
	}
	repo.invites[PendingRegistrationID(3)] = &PendingRegistration{
		ID: 3, DeviceName: "Invalidated",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, InvalidatedAt: &invalidatedAt,
	}

	pending, err := svc.ListInvites(context.Background(), InviteFilter{})
	is.NoErr(err)
	is.Equal(len(pending), 1)
	is.Equal(pending[0].ID, PendingRegistrationID(0))
	is.Equal(pending[0].DeviceName, "Active")

	all, err := svc.ListInvites(context.Background(), InviteFilter{IncludeAll: true})
	is.NoErr(err)
	is.Equal(len(all), 4)
}

func TestService_ClaimInvite_ReturnsConfig(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	repo.claimResult = &ClaimResult{
		ServerURL:           "https://pulse.home.lan",
		IntervalSeconds:     900,
		AppBiometricEnabled: false,
		AppSettingsLocked:   false,
		RawAPIKey:           "wdk_testkey",
	}
	svc := newTestService(repo)

	result, err := svc.ClaimInvite(context.Background(), "someCode")
	is.NoErr(err)
	is.Equal(result.ServerURL, "https://pulse.home.lan")
	is.Equal(result.RawAPIKey, "wdk_testkey")
}

func TestService_ClaimInvite_ReturnsNotFoundOnError(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	repo.claimErr = ErrInviteNotFound
	svc := newTestService(repo)

	_, err := svc.ClaimInvite(context.Background(), "badCode")
	is.True(errors.Is(err, ErrInviteNotFound))
}

func TestService_InvalidateInvite_SoftDeletesPendingInvite(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	repo.invites[0] = &PendingRegistration{
		ID: 0, DeviceName: "Test",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
	}

	err := svc.InvalidateInvite(context.Background(), 0)
	is.NoErr(err)

	// Invite still exists (soft delete), but its status is now Invalidated.
	invite, err := repo.GetInvite(context.Background(), 0)
	is.NoErr(err)
	is.Equal(invite.Status(), StatusInvalidated)
}

func TestService_InvalidateInvite_NotFound(t *testing.T) {
	is := is.New(t)
	svc := newTestService(newMockRepo())

	err := svc.InvalidateInvite(context.Background(), 0)
	is.True(errors.Is(err, ErrInviteNotFound))
}

func TestService_InvalidateInvite_AlreadyUsed(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	repo.invites[0] = &PendingRegistration{
		ID: 0, DeviceName: "Used Device",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now.Add(-2 * time.Hour), UsedAt: new(now.Add(-1 * time.Hour)),
	}

	err := svc.InvalidateInvite(context.Background(), 0)
	is.True(errors.Is(err, ErrInviteNotPending))
}

func TestPendingRegistration_Status(t *testing.T) {
	is := is.New(t)
	now := time.Now().UTC()
	past := now.Add(-time.Minute)

	pending := &PendingRegistration{ExpiresAt: now.Add(time.Hour)}
	is.Equal(pending.Status(), StatusPending)

	expired := &PendingRegistration{ExpiresAt: now.Add(-time.Hour)}
	is.Equal(expired.Status(), StatusExpired)

	used := &PendingRegistration{ExpiresAt: now.Add(time.Hour), UsedAt: &past}
	is.Equal(used.Status(), StatusUsed)

	invalidated := &PendingRegistration{ExpiresAt: now.Add(time.Hour), InvalidatedAt: &past}
	is.Equal(invalidated.Status(), StatusInvalidated)
}
