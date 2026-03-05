package lease

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/matryer/is"
)

func TestService_AddAddressLease_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	ttlSeconds := 60
	mockTTL := &mockTTLConfigRetriever{ttl: &ttlSeconds}

	service := NewService(mockRepo, mockTTL, slog.Default())

	deviceID := device.DeviceID(1)
	addressID := device.AddressID(10)

	lease, err := service.AddAddressLease(ctx, deviceID, addressID)
	is.NoErr(err)
	is.True(lease != nil)
	is.Equal(lease.AddressID, addressID)
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
	is.True(lease == nil)
	is.Equal(mockRepo.upsertCalls, 0)
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

func TestService_DeleteAddressLease_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	addressID := device.AddressID(10)
	mockRepo.leases[addressID] = &AddressLease{AddressID: addressID}

	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	err := service.DeleteAddressLease(ctx, addressID)
	is.NoErr(err)
	is.Equal(mockRepo.deleteCalls, 1)
	_, exists := mockRepo.leases[addressID]
	is.True(!exists)
}

func TestService_DeleteAddressLease_NotFoundIsNotError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	err := service.DeleteAddressLease(ctx, device.AddressID(999))
	is.NoErr(err)
	is.Equal(mockRepo.deleteCalls, 1)
}

func TestService_DeleteAddressLease_RepositoryError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	repoErr := errors.New("delete failed")
	mockRepo.deleteErr = repoErr

	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	err := service.DeleteAddressLease(ctx, device.AddressID(10))
	is.True(err != nil)
	is.Equal(err, repoErr)
	is.Equal(mockRepo.deleteCalls, 1)
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

func TestService_OnAddressEvent_AssignedEventProcessedByRunListener(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRepo := newMockRepository()
	mockRepo.upsertCalledCh = make(chan struct{}, 1)
	ttlSeconds := 30
	mockTTL := &mockTTLConfigRetriever{ttl: &ttlSeconds}

	service := NewService(mockRepo, mockTTL, slog.Default())

	done := make(chan error, 1)
	go func() {
		done <- service.RunListener(ctx)
	}()

	event := device.AddressEvent{
		Type:      device.EventTypeAddressAssigned,
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
	mockRepo.deleteCalledCh = make(chan struct{}, 1)

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
	case <-mockRepo.deleteCalledCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for DeleteAddressLeaseByAddressID to be called")
	}

	cancel()
	err := <-done
	is.NoErr(err)
	is.Equal(mockRepo.lastDeletedID, event.AddressID)
}

func TestService_OnAddressEvent_ContextCancellationUnblocksWhenChannelFull(t *testing.T) {
	is := is.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	mockRepo := newMockRepository()
	service := NewService(mockRepo, &mockTTLConfigRetriever{}, slog.Default())

	// Reduce buffer size for this test to ensure the channel can fill.
	service.events = make(chan device.AddressEvent, 1)

	// Fill the channel so the next send would block.
	service.events <- device.AddressEvent{Type: device.EventTypeAddressAssigned}

	done := make(chan struct{})
	started := make(chan struct{})
	go func() {
		close(started)
		service.OnAddressEvent(ctx, device.AddressEvent{Type: device.EventTypeAddressAssigned})
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
	deleteErr         error
	getExpiredErr     error
	expiredAddressIDs []device.AddressID

	upsertCalls int
	deleteCalls int

	lastUpsertLease *AddressLease
	lastDeletedID   device.AddressID

	upsertCalledCh chan struct{}
	deleteCalledCh chan struct{}
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

func (m *mockRepository) DeleteAddressLeaseByAddressID(_ context.Context, addressID device.AddressID) error {
	m.deleteCalls++
	m.lastDeletedID = addressID

	if m.deleteCalledCh != nil {
		select {
		case m.deleteCalledCh <- struct{}{}:
		default:
		}
	}

	if m.deleteErr != nil {
		return m.deleteErr
	}

	if m.leases == nil {
		return ErrAddressLeaseNotFound
	}

	if _, ok := m.leases[addressID]; !ok {
		return ErrAddressLeaseNotFound
	}

	delete(m.leases, addressID)

	return nil
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
