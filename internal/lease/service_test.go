//go:build test

package lease

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/matryer/is"
)

func TestService_AddAddressLease_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockTTL := &mockTTLConfigRetriever{ttl: new(60)}

	service := NewService(mockRepo, mockTTL, slog.Default())

	deviceID := device.DeviceID(1)
	addressID := device.AddressID(10)

	lease, err := service.AddAddressLease(ctx, deviceID, addressID)
	is.NoErr(err)
	is.True(lease != nil)
	is.Equal(lease.AddressID, addressID)
	is.Equal(lease.DeviceID, deviceID)
	is.True(lease.ExpiresAt != nil)
	is.Equal(mockTTL.lastDeviceID, deviceID)
	is.Equal(mockRepo.upsertCalls, 1)

	stored, ok := mockRepo.leases[addressID]
	is.True(ok)
	is.Equal(stored, lease)
}

func TestService_AddAddressLease_NoTTLConfigured(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockTTL := &mockTTLConfigRetriever{ttl: nil}

	service := NewService(mockRepo, mockTTL, slog.Default())

	lease, err := service.AddAddressLease(ctx, device.DeviceID(1), device.AddressID(10))
	is.NoErr(err)
	is.True(lease != nil)
	is.True(lease.ExpiresAt == nil)
	is.Equal(mockRepo.upsertCalls, 1)
}

func TestService_AddAddressLease_TTLConfigError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	cfgErr := errors.New("ttl config error")
	mockTTL := &mockTTLConfigRetriever{err: cfgErr}

	service := NewService(mockRepo, mockTTL, slog.Default())

	lease, err := service.AddAddressLease(ctx, device.DeviceID(1), device.AddressID(10))
	is.True(err != nil)
	is.Equal(err, cfgErr)
	is.True(lease == nil)
	is.Equal(mockRepo.upsertCalls, 0)
}

func TestService_ClearAddressLease_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	deviceID := device.DeviceID(1)
	addressID := device.AddressID(10)
	mockRepo.leases[addressID] = &AddressLease{AddressID: addressID, DeviceID: deviceID, ExpiresAt: new(time.Now().UTC().Add(1 * time.Minute))}

	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	lease, err := service.ClearAddressLease(ctx, deviceID, addressID)
	is.NoErr(err)
	is.True(lease != nil)
	is.True(lease.ExpiresAt == nil)
	is.Equal(mockRepo.upsertCalls, 1)
	is.Equal(mockRepo.lastUpsertLease.DeviceID, deviceID)
}

func TestService_GetExpiredAddressIDs(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.expiredAddressIDs = []device.AddressID{device.AddressID(1), device.AddressID(2)}

	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	ids, err := service.GetExpiredAddressIDs(ctx)
	is.NoErr(err)
	is.Equal(len(ids), 2)
	is.Equal(ids[0], device.AddressID(1))
	is.Equal(ids[1], device.AddressID(2))
}

func TestService_OnAddressEvent_CreatedEventProcessedByRunListener(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRepo := newMockRepository()
	mockRepo.upsertCalledCh = make(chan struct{}, 1)
	mockTTL := &mockTTLConfigRetriever{ttl: new(30)}

	service := NewService(mockRepo, mockTTL, slog.Default())

	done := make(chan error, 1)
	go func() {
		done <- service.RunListener(ctx)
	}()

	event := device.AddressEvent{
		Type:      device.EventTypeAddressCreated,
		DeviceID:  device.DeviceID(1),
		AddressID: device.AddressID(42),
	}

	service.OnAddressEvent(ctx, event)

	select {
	case <-mockRepo.upsertCalledCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for UpsertAddressLease to be called")
	}

	cancel()
	err := <-done
	is.NoErr(err)
	is.Equal(mockRepo.lastUpsertLease.AddressID, event.AddressID)
}

func TestService_OnAddressEvent_DisabledEventProcessedByRunListener(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRepo := newMockRepository()
	mockRepo.upsertCalledCh = make(chan struct{}, 1)

	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	done := make(chan error, 1)
	go func() {
		done <- service.RunListener(ctx)
	}()

	event := device.AddressEvent{
		Type:      device.EventTypeAddressDisabled,
		DeviceID:  device.DeviceID(1),
		AddressID: device.AddressID(100),
	}

	mockRepo.leases[event.AddressID] = &AddressLease{AddressID: event.AddressID}

	service.OnAddressEvent(ctx, event)

	select {
	case <-mockRepo.upsertCalledCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for UpsertAddressLease to be called")
	}

	cancel()
	err := <-done
	is.NoErr(err)
	is.Equal(mockRepo.lastUpsertLease.AddressID, event.AddressID)
	is.Equal(mockRepo.lastUpsertLease.DeviceID, event.DeviceID)
	is.True(mockRepo.lastUpsertLease.ExpiresAt == nil)
}

func TestService_OnAddressEvent_ContextCancellationUnblocksWhenChannelFull(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	// Reduce buffer size for this test to ensure the channel can fill.
	service.events = make(chan device.AddressEvent, 1)

	// Fill the channel so the next send would block.
	service.events <- device.AddressEvent{Type: device.EventTypeAddressCreated}

	done := make(chan struct{})
	started := make(chan struct{})
	go func() {
		close(started)
		service.OnAddressEvent(ctx, device.AddressEvent{Type: device.EventTypeAddressCreated})
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("OnAddressEvent did not return after context cancellation with full buffer")
	}

	// No further assertions; test ensures we don't deadlock when buffer is full.
	is.True(true)
}

func TestService_handleLeaseRuleEvent_EnabledRuleUpdatesDeviceLeases(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	event := rule.RuleEvent{
		Type:       rule.RuleEventTypeEnabled,
		DeviceID:   device.DeviceID(99),
		RuleType:   rule.RuleTypeDeviceAddressLease,
		TTLSeconds: new(300),
		OccurredAt: time.Now().UTC(),
	}

	service.handleLeaseRuleEvent(ctx, event)

	is.Equal(mockRepo.setDeviceLeasesExpiryCalls, 1)
	is.Equal(mockRepo.lastSetDeviceID, event.DeviceID)
	is.True(mockRepo.lastSetExpiresAt != nil)
}

func TestService_handleLeaseRuleEvent_DisabledRuleClearsDeviceLeases(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	event := rule.RuleEvent{
		Type:       rule.RuleEventTypeDisabled,
		DeviceID:   device.DeviceID(99),
		RuleType:   rule.RuleTypeDeviceAddressLease,
		TTLSeconds: nil,
		OccurredAt: time.Now().UTC(),
	}

	service.handleLeaseRuleEvent(ctx, event)

	is.Equal(mockRepo.setDeviceLeasesExpiryCalls, 1)
	is.Equal(mockRepo.lastSetDeviceID, event.DeviceID)
	is.True(mockRepo.lastSetExpiresAt == nil)
}

func TestService_handleLeaseRuleEvent_WrongRuleTypeIgnored(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	event := rule.RuleEvent{
		Type:       rule.RuleEventTypeEnabled,
		DeviceID:   device.DeviceID(99),
		RuleType:   rule.RuleType("other_rule"),
		TTLSeconds: new(60),
		OccurredAt: time.Now().UTC(),
	}

	service.handleLeaseRuleEvent(ctx, event)

	is.Equal(mockRepo.setDeviceLeasesExpiryCalls, 0)
}

func TestService_OnRuleEvent_EnabledEventProcessedByRunListener(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRepo := newMockRepository()
	mockRepo.setExpiryCalledCh = make(chan struct{}, 1)

	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	done := make(chan error, 1)
	go func() {
		done <- service.RunListener(ctx)
	}()

	event := rule.RuleEvent{
		Type:       rule.RuleEventTypeEnabled,
		DeviceID:   device.DeviceID(99),
		RuleType:   rule.RuleTypeDeviceAddressLease,
		TTLSeconds: new(300),
		OccurredAt: time.Now().UTC(),
	}

	service.OnRuleEvent(ctx, event)

	select {
	case <-mockRepo.setExpiryCalledCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for SetDeviceAddressLeasesExpiry to be called")
	}

	cancel()
	err := <-done
	is.NoErr(err)
	is.Equal(mockRepo.setDeviceLeasesExpiryCalls, 1)
	is.Equal(mockRepo.lastSetDeviceID, event.DeviceID)
	is.True(mockRepo.lastSetExpiresAt != nil)
}

func TestService_RunListener_ContextCancellationExitsCleanly(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	done := make(chan error, 1)
	go func() {
		done <- service.RunListener(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(1 * time.Second):
		t.Fatal("RunListener did not exit after context cancellation")
	}
}

type mockTTLConfigRetriever struct {
	ttl          *int
	err          error
	lastDeviceID device.DeviceID
	calls        int
}

func (m *mockTTLConfigRetriever) GetDeviceAddressLeaseTTLSeconds(_ context.Context, deviceID device.DeviceID) (*int, error) {
	m.calls++
	m.lastDeviceID = deviceID
	if m.err != nil {
		return nil, m.err
	}
	return m.ttl, nil
}

type mockRepository struct {
	leases            map[device.AddressID]*AddressLease
	upsertErr         error
	getExpiredErr     error
	expiredAddressIDs []device.AddressID
	setExpiryErr      error

	upsertCalls                int
	setDeviceLeasesExpiryCalls int

	lastUpsertLease  *AddressLease
	lastSetDeviceID  device.DeviceID
	lastSetExpiresAt *time.Time

	upsertCalledCh    chan struct{}
	setExpiryCalledCh chan struct{}
}

var _ repository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		leases: make(map[device.AddressID]*AddressLease),
	}
}

func (m *mockRepository) UpsertAddressLease(_ context.Context, lease *AddressLease) (*AddressLease, error) {
	m.upsertCalls++
	m.lastUpsertLease = lease

	if m.upsertCalledCh != nil {
		select {
		case m.upsertCalledCh <- struct{}{}:
		default:
		}
	}

	if m.upsertErr != nil {
		return nil, m.upsertErr
	}

	if m.leases == nil {
		m.leases = make(map[device.AddressID]*AddressLease)
	}
	m.leases[lease.AddressID] = lease

	return lease, nil
}

func (m *mockRepository) GetExpiredAddressIDs(_ context.Context) ([]device.AddressID, error) {
	if m.getExpiredErr != nil {
		return nil, m.getExpiredErr
	}

	if m.expiredAddressIDs == nil {
		return []device.AddressID{}, nil
	}

	return m.expiredAddressIDs, nil
}

func (m *mockRepository) SetDeviceAddressLeasesExpiry(_ context.Context, deviceID device.DeviceID, expiresAt *time.Time, updatedAt time.Time) error {
	m.setDeviceLeasesExpiryCalls++
	m.lastSetDeviceID = deviceID
	m.lastSetExpiresAt = expiresAt

	if m.setExpiryCalledCh != nil {
		select {
		case m.setExpiryCalledCh <- struct{}{}:
		default:
		}
	}

	if m.setExpiryErr != nil {
		return m.setExpiryErr
	}
	return nil
}
