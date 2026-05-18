//go:build test

package registration_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/registration"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// --- mock repository ---

type mockRepo struct {
	invites  map[registration.PendingRegistrationID]*registration.PendingRegistration
	claimErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{invites: make(map[registration.PendingRegistrationID]*registration.PendingRegistration)}
}

func (m *mockRepo) CreateInvite(_ context.Context, p registration.CreateInviteRequest) (*registration.PendingRegistration, error) {
	pendReg := &registration.PendingRegistration{
		ID:               registration.PendingRegistrationID(len(m.invites) + 1),
		DeviceName:       p.DeviceName,
		RegistrationCode: new(p.RegistrationCode),
		ExpiresAt:        p.ExpiresAt,
	}

	m.invites[pendReg.ID] = pendReg
	return pendReg, nil
}

func (m *mockRepo) GetInvite(_ context.Context, id registration.PendingRegistrationID) (*registration.PendingRegistration, error) {
	p, ok := m.invites[id]
	if !ok {
		return nil, registration.ErrInviteNotFound
	}
	return p, nil
}

func (m *mockRepo) GetInviteByCode(_ context.Context, code string) (*registration.PendingRegistration, error) {
	p, ok := m.invites[1]
	if !ok {
		return nil, registration.ErrInviteNotFound
	}
	return p, nil
}

func (m *mockRepo) ListInvites(_ context.Context, filter registration.InviteFilter) ([]registration.PendingRegistration, error) {
	var result []registration.PendingRegistration
	for _, p := range m.invites {
		if !filter.IncludeAll && p.Status() != registration.StatusPending {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockRepo) InvalidateInvite(_ context.Context, id registration.PendingRegistrationID) error {
	p, ok := m.invites[id]
	if !ok {
		return registration.ErrInviteNotFound
	}
	if p.UsedAt != nil || p.InvalidatedAt != nil {
		return registration.ErrInviteNotPending
	}
	now := time.Now()
	p.InvalidatedAt = &now
	return nil
}

func (m *mockRepo) ClaimInvite(_ context.Context, id registration.PendingRegistrationID, _ ids.DeviceID) (*registration.PendingRegistration, error) {
	return m.invites[id], m.claimErr
}

// Implement a mock device provisioner
type mockDeviceProvisioner struct {
}

func newMockDeviceProvisioner() *mockDeviceProvisioner {
	return &mockDeviceProvisioner{}
}

func (mp *mockDeviceProvisioner) CreateDeviceWithAPIKey(ctx context.Context, name string, ownerID ids.UserID) (deviceID ids.DeviceID, rawAPIKey string, err error) {
	return ids.DeviceID(1), "an api key", nil
}

// --- tests ---
func newTestService(repo *mockRepo) *registration.Service {
	mockProv := newMockDeviceProvisioner()
	return registration.NewService(repo, testutils.NoopTransactor{}, mockProv, slog.New(slog.DiscardHandler))
}

func TestService_CreateInvite_StoresInvite(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	invite, err := svc.CreateInvite(context.Background(), registration.CreateInviteRequest{
		DeviceName:         "Dad's Phone",
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	})
	is.NoErr(err)
	is.True(invite != nil)
	is.Equal(invite.DeviceName, "Dad's Phone")
	is.True(invite.RegistrationCode != nil && *invite.RegistrationCode != "")
	is.Equal(invite.Status(), registration.StatusPending)

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
	invite, err := svc.CreateInvite(context.Background(), registration.CreateInviteRequest{
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

	_, err := svc.GetInvite(context.Background(), registration.PendingRegistrationID(0))
	is.True(errors.Is(err, registration.ErrInviteNotFound))
}

func TestService_ListInvites_PendingOnly(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	invalidatedAt := now.Add(-30 * time.Minute)

	repo.invites[registration.PendingRegistrationID(0)] = &registration.PendingRegistration{
		ID: 0, DeviceName: "Active",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
	}
	repo.invites[registration.PendingRegistrationID(1)] = &registration.PendingRegistration{
		ID: 1, DeviceName: "Used",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UsedAt: new(now.Add(-1 * time.Hour)),
	}
	repo.invites[registration.PendingRegistrationID(2)] = &registration.PendingRegistration{
		ID: 2, DeviceName: "Expired",
		ExpiresAt: now.Add(-1 * time.Hour), CreatedAt: now.Add(-2 * time.Hour),
	}
	repo.invites[registration.PendingRegistrationID(3)] = &registration.PendingRegistration{
		ID: 3, DeviceName: "Invalidated",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, InvalidatedAt: &invalidatedAt,
	}

	pending, err := svc.ListInvites(context.Background(), registration.InviteFilter{})
	is.NoErr(err)
	is.Equal(len(pending), 1)
	is.Equal(pending[0].ID, registration.PendingRegistrationID(0))
	is.Equal(pending[0].DeviceName, "Active")

	all, err := svc.ListInvites(context.Background(), registration.InviteFilter{IncludeAll: true})
	is.NoErr(err)
	is.Equal(len(all), 4)
}

func TestService_ClaimInvite_ReturnsConfig(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	repo.invites[1] = &registration.PendingRegistration{
		ID:                       1,
		DeviceName:               "Dad's Phone",
		HeartbeatServerURL:       "https://pulse.home.lan",
		HeartbeatIntervalSeconds: 900,
		ExpiresAt:                time.Now().Add(24 * time.Hour),
	}
	svc := newTestService(repo)

	result, err := svc.ClaimInvite(context.Background(), "someCode")
	is.NoErr(err)
	is.Equal(result.ServerURL, "https://pulse.home.lan")
	is.Equal(result.RawAPIKey, "an api key")
}

func TestService_ClaimInvite_ReturnsNotFoundOnError(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	repo.claimErr = registration.ErrInviteNotFound
	svc := newTestService(repo)

	_, err := svc.ClaimInvite(context.Background(), "badCode")
	is.True(errors.Is(err, registration.ErrInviteNotFound))
}

func TestService_InvalidateInvite_SoftDeletesPendingInvite(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	repo.invites[0] = &registration.PendingRegistration{
		ID: 0, DeviceName: "Test",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
	}

	err := svc.InvalidateInvite(context.Background(), 0)
	is.NoErr(err)

	// Invite still exists (soft delete), but its status is now Invalidated.
	invite, err := repo.GetInvite(context.Background(), 0)
	is.NoErr(err)
	is.Equal(invite.Status(), registration.StatusInvalidated)
}

func TestService_InvalidateInvite_NotFound(t *testing.T) {
	is := is.New(t)
	svc := newTestService(newMockRepo())

	err := svc.InvalidateInvite(context.Background(), 0)
	is.True(errors.Is(err, registration.ErrInviteNotFound))
}

func TestService_InvalidateInvite_AlreadyUsed(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	repo.invites[0] = &registration.PendingRegistration{
		ID: 0, DeviceName: "Used Device",
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now.Add(-2 * time.Hour), UsedAt: new(now.Add(-1 * time.Hour)),
	}

	err := svc.InvalidateInvite(context.Background(), 0)
	is.True(errors.Is(err, registration.ErrInviteNotPending))
}

func TestPendingRegistration_Status(t *testing.T) {
	is := is.New(t)
	now := time.Now().UTC()
	past := now.Add(-time.Minute)

	pending := &registration.PendingRegistration{ExpiresAt: now.Add(time.Hour)}
	is.Equal(pending.Status(), registration.StatusPending)

	expired := &registration.PendingRegistration{ExpiresAt: now.Add(-time.Hour)}
	is.Equal(expired.Status(), registration.StatusExpired)

	used := &registration.PendingRegistration{ExpiresAt: now.Add(time.Hour), UsedAt: &past}
	is.Equal(used.Status(), registration.StatusUsed)

	invalidated := &registration.PendingRegistration{ExpiresAt: now.Add(time.Hour), InvalidatedAt: &past}
	is.Equal(invalidated.Status(), registration.StatusInvalidated)
}
