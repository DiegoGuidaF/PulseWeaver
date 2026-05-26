//go:build test

package devicepairing_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/devicepairing"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// --- mock repository ---

type mockRepo struct {
	pairings map[ids.DevicePairingID]*devicepairing.DevicePairing
	claimErr error
}

func newMockRepo() *mockRepo {
	return &mockRepo{pairings: make(map[ids.DevicePairingID]*devicepairing.DevicePairing)}
}

func (m *mockRepo) CreatePairing(_ context.Context, p devicepairing.CreatePairingRequest) (*devicepairing.DevicePairing, error) {
	dp := &devicepairing.DevicePairing{
		ID:          ids.DevicePairingID(len(m.pairings) + 1),
		DeviceID:    p.DeviceID,
		PairingCode: p.PairingCode,
		ExpiresAt:   p.ExpiresAt,
		Status:      devicepairing.StatusPending,
	}
	m.pairings[dp.ID] = dp
	return dp, nil
}

func (m *mockRepo) GetPairing(_ context.Context, id ids.DevicePairingID) (*devicepairing.DevicePairing, error) {
	p, ok := m.pairings[id]
	if !ok {
		return nil, devicepairing.ErrPairingNotFound
	}
	return p, nil
}

func (m *mockRepo) GetPairingByCode(_ context.Context, _ string) (*devicepairing.DevicePairing, error) {
	p, ok := m.pairings[1]
	if !ok {
		return nil, devicepairing.ErrPairingNotFound
	}
	return p, nil
}

func (m *mockRepo) ListPairings(_ context.Context, filter devicepairing.PairingFilter) ([]devicepairing.DevicePairing, error) {
	var result []devicepairing.DevicePairing
	for _, p := range m.pairings {
		if !filter.IncludeAll && p.Status != devicepairing.StatusPending {
			continue
		}
		result = append(result, *p)
	}
	return result, nil
}

func (m *mockRepo) ReplacePendingPairings(_ context.Context, deviceID ids.DeviceID) error {
	for _, p := range m.pairings {
		if p.DeviceID == deviceID && p.Status == devicepairing.StatusPending {
			p.Status = devicepairing.StatusReplaced
		}
	}
	return nil
}

func (m *mockRepo) InvalidatePairing(_ context.Context, _ ids.DeviceID, id ids.DevicePairingID) error {
	p, ok := m.pairings[id]
	if !ok {
		return devicepairing.ErrPairingNotFound
	}
	if p.Status != devicepairing.StatusPending {
		return devicepairing.ErrPairingNotPending
	}
	p.Status = devicepairing.StatusInvalidated
	return nil
}

func (m *mockRepo) ClaimPairing(_ context.Context, id ids.DevicePairingID) (*devicepairing.DevicePairing, error) {
	return m.pairings[id], m.claimErr
}

// --- mock api key manager ---

type mockAPIKeyManager struct{}

func (m *mockAPIKeyManager) RegenerateAPIKey(_ context.Context, _ ids.DeviceID) (*device.Device, string, error) {
	return nil, "an api key", nil
}

// --- tests ---

func newTestService(repo *mockRepo) *devicepairing.Service {
	return devicepairing.NewService(repo, testutils.NoopTransactor{}, &mockAPIKeyManager{}, slog.New(slog.DiscardHandler))
}

func TestService_CreatePairing_StoresPairing(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	pairing, err := svc.CreatePairing(context.Background(), devicepairing.CreatePairingRequest{
		DeviceID:           ids.DeviceID(1),
		HeartbeatServerURL: "https://pulse.home.lan",
		IntervalSeconds:    900,
		ExpiresInHours:     24,
	})
	is.NoErr(err)
	is.True(pairing != nil)
	is.Equal(pairing.DeviceID, ids.DeviceID(1))
	is.True(pairing.PairingCode != "")
	is.Equal(pairing.Status, devicepairing.StatusPending)

	stored, err := repo.GetPairing(context.Background(), pairing.ID)
	is.NoErr(err)
	is.Equal(stored.DeviceID, ids.DeviceID(1))
}

func TestService_CreatePairing_ReplacesPreviousPending(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	first, err := svc.CreatePairing(context.Background(), devicepairing.CreatePairingRequest{
		DeviceID: ids.DeviceID(1), HeartbeatServerURL: "https://pulse.home.lan", ExpiresInHours: 24,
	})
	is.NoErr(err)

	_, err = svc.CreatePairing(context.Background(), devicepairing.CreatePairingRequest{
		DeviceID: ids.DeviceID(1), HeartbeatServerURL: "https://pulse.home.lan", ExpiresInHours: 24,
	})
	is.NoErr(err)

	replaced, err := repo.GetPairing(context.Background(), first.ID)
	is.NoErr(err)
	is.Equal(replaced.Status, devicepairing.StatusReplaced)
}

func TestService_CreatePairing_ExpirySetCorrectly(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	before := time.Now().Add(23 * time.Hour)
	pairing, err := svc.CreatePairing(context.Background(), devicepairing.CreatePairingRequest{
		DeviceID:           ids.DeviceID(1),
		HeartbeatServerURL: "https://example.com",
		IntervalSeconds:    300,
		ExpiresInHours:     24,
	})
	after := time.Now().Add(25 * time.Hour)

	is.NoErr(err)
	is.True(pairing.ExpiresAt.After(before))
	is.True(pairing.ExpiresAt.Before(after))
}

func TestService_GetPairing_ReturnsNotFound(t *testing.T) {
	is := is.New(t)
	svc := newTestService(newMockRepo())

	_, err := svc.GetPairing(context.Background(), ids.DevicePairingID(0))
	is.True(errors.Is(err, devicepairing.ErrPairingNotFound))
}

func TestService_ListPairings_PendingOnly(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()

	repo.pairings[ids.DevicePairingID(0)] = &devicepairing.DevicePairing{
		ID: 0, DeviceID: ids.DeviceID(1),
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
		Status: devicepairing.StatusPending,
	}
	repo.pairings[ids.DevicePairingID(1)] = &devicepairing.DevicePairing{
		ID: 1, DeviceID: ids.DeviceID(1),
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
		Status: devicepairing.StatusUsed,
	}
	repo.pairings[ids.DevicePairingID(2)] = &devicepairing.DevicePairing{
		ID: 2, DeviceID: ids.DeviceID(1),
		ExpiresAt: now.Add(-1 * time.Hour), CreatedAt: now.Add(-2 * time.Hour),
		Status: devicepairing.StatusExpired,
	}
	repo.pairings[ids.DevicePairingID(3)] = &devicepairing.DevicePairing{
		ID: 3, DeviceID: ids.DeviceID(1),
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
		Status: devicepairing.StatusInvalidated,
	}

	pending, err := svc.ListPairings(context.Background(), devicepairing.PairingFilter{DeviceID: ids.DeviceID(1)})
	is.NoErr(err)
	is.Equal(len(pending), 1)
	is.Equal(pending[0].ID, ids.DevicePairingID(0))
	is.Equal(pending[0].DeviceID, ids.DeviceID(1))

	all, err := svc.ListPairings(context.Background(), devicepairing.PairingFilter{DeviceID: ids.DeviceID(1), IncludeAll: true})
	is.NoErr(err)
	is.Equal(len(all), 4)
}

func TestService_ClaimPairing_ReturnsConfig(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	repo.pairings[1] = &devicepairing.DevicePairing{
		ID:                       1,
		DeviceID:                 ids.DeviceID(1),
		HeartbeatServerURL:       "https://pulse.home.lan",
		HeartbeatIntervalSeconds: 900,
		ExpiresAt:                time.Now().Add(24 * time.Hour),
		Status:                   devicepairing.StatusPending,
	}
	svc := newTestService(repo)

	result, err := svc.ClaimPairing(context.Background(), "someCode")
	is.NoErr(err)
	is.Equal(result.ServerURL, "https://pulse.home.lan")
	is.Equal(result.RawAPIKey, "an api key")
}

func TestService_ClaimPairing_ReturnsNotFoundOnError(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	repo.claimErr = devicepairing.ErrPairingNotFound
	svc := newTestService(repo)

	_, err := svc.ClaimPairing(context.Background(), "badCode")
	is.True(errors.Is(err, devicepairing.ErrPairingNotFound))
}

func TestService_InvalidatePairing_SoftDeletesPendingPairing(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	repo.pairings[0] = &devicepairing.DevicePairing{
		ID: 0, DeviceID: ids.DeviceID(1),
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now,
		Status: devicepairing.StatusPending,
	}

	err := svc.InvalidatePairing(context.Background(), ids.DeviceID(1), 0)
	is.NoErr(err)

	pairing, err := repo.GetPairing(context.Background(), 0)
	is.NoErr(err)
	is.Equal(pairing.Status, devicepairing.StatusInvalidated)
}

func TestService_InvalidatePairing_NotFound(t *testing.T) {
	is := is.New(t)
	svc := newTestService(newMockRepo())

	err := svc.InvalidatePairing(context.Background(), ids.DeviceID(1), 0)
	is.True(errors.Is(err, devicepairing.ErrPairingNotFound))
}

func TestService_InvalidatePairing_AlreadyUsed(t *testing.T) {
	is := is.New(t)
	repo := newMockRepo()
	svc := newTestService(repo)

	now := time.Now().UTC()
	repo.pairings[0] = &devicepairing.DevicePairing{
		ID: 0, DeviceID: ids.DeviceID(1),
		ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now.Add(-2 * time.Hour),
		Status: devicepairing.StatusUsed,
	}

	err := svc.InvalidatePairing(context.Background(), ids.DeviceID(1), 0)
	is.True(errors.Is(err, devicepairing.ErrPairingNotPending))
}

func TestEvalStatus(t *testing.T) {
	is := is.New(t)
	now := time.Now().UTC()

	is.Equal(devicepairing.EvalStatus("pending", now.Add(time.Hour)), devicepairing.StatusPending)
	is.Equal(devicepairing.EvalStatus("pending", now.Add(-time.Hour)), devicepairing.StatusExpired)
	is.Equal(devicepairing.EvalStatus("used", now.Add(time.Hour)), devicepairing.StatusUsed)
	is.Equal(devicepairing.EvalStatus("invalidated", now.Add(time.Hour)), devicepairing.StatusInvalidated)
	is.Equal(devicepairing.EvalStatus("replaced", now.Add(time.Hour)), devicepairing.StatusReplaced)
}
