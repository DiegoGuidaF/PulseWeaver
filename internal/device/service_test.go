//go:build test

package device

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/matryer/is"
)

func TestService_RegisterAddressActivity_NewAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.ID, "192.168.1.100", EventSourceManual)
	is.NoErr(err)
	is.Equal(eventType, EventTypeAddressCreated)
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.IsEnabled)
}

func TestService_RegisterAddressActivity_ExistingAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	existingAddr := &Address{
		ID:        AddressID(1),
		DeviceID:  device.ID,
		IP:        "192.168.1.100",
		IsEnabled: false,
	}
	key := device.ID.String() + ":192.168.1.100"
	mockRepo.addresses[existingAddr.ID] = existingAddr
	mockRepo.deviceAddressByIP[key] = existingAddr

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.ID, "192.168.1.100", EventSourceManual)
	is.NoErr(err)
	is.Equal(eventType, EventTypeAddressEnabled) // Address already existed, we just enabled it
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.IsEnabled) // Should be enabled
}

func TestService_RegisterAddressActivity_ExistingEnabledAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	existingAddr := &Address{
		ID:        AddressID(1),
		DeviceID:  device.ID,
		IP:        "192.168.1.100",
		IsEnabled: true, // already enabled
	}
	key := device.ID.String() + ":192.168.1.100"
	mockRepo.addresses[existingAddr.ID] = existingAddr
	mockRepo.deviceAddressByIP[key] = existingAddr

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.ID, "192.168.1.100", EventSourceHeartbeat)
	is.NoErr(err)
	is.Equal(eventType, EventTypeAddressRefreshed)
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.IsEnabled)

	is.Equal(len(observer.events), 1)
	event := observer.events[0]
	is.Equal(event.Type, EventTypeAddressRefreshed)
	is.Equal(event.AddressID, addr.ID)
	is.Equal(event.DeviceID, device.ID)
	is.True(!event.OccurredAt.IsZero())
}

func TestService_RegisterAddressActivity_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceErr = ErrDeviceNotFound

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	addr, eventType, err := service.RegisterAddressActivity(ctx, DeviceID(999), "192.168.1.100", EventSourceManual)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
	is.True(addr == nil)
	is.Equal(eventType, EventType(""))
}

func TestService_RegisterAddressActivity_RejectsTrustedProxyIP(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.MustParseAddr("10.1.2.3"))

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.ID, "10.1.2.3", EventSourceHeartbeat)
	is.True(errors.Is(err, ErrTrustedProxyIPRejected))
	is.True(addr == nil)
	is.Equal(eventType, EventType(""))
}

func TestService_RegisterAddressActivity_TransactionRollback(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	// Simulate transaction failure
	testErr := errors.New("transaction error")
	mockRepo.runInTxFn = func(repo repository) error {
		// Try to create address
		params, _ := NewCreateAddressParams(device.ID, "192.168.1.100", netip.Addr{})
		_, err := repo.CreateAddress(ctx, params, EventSourceManual)
		if err != nil {
			return err
		}
		// Return error to trigger rollback
		return testErr
	}

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.ID, "192.168.1.100", EventSourceManual)
	is.True(err != nil)
	is.Equal(err, testErr)
	is.True(addr == nil)
	is.Equal(eventType, EventType(""))
}

func TestService_DisableAddress_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	address := &Address{
		ID:        AddressID(1),
		DeviceID:  device.ID,
		IP:        "192.168.1.100",
		IsEnabled: true,
	}
	mockRepo.addresses[address.ID] = address

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	disabledAddr, err := service.DisableAddress(ctx, device.ID, address.ID)
	is.NoErr(err)
	is.True(disabledAddr != nil)
	is.True(!disabledAddr.IsEnabled)
}

func TestService_DisableAddress_OwnershipValidation(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device1 := &Device{ID: DeviceID(1), Name: "device-1"}
	device2 := &Device{ID: DeviceID(2), Name: "device-2"}
	mockRepo.devices[device1.ID] = device1
	mockRepo.devices[device2.ID] = device2

	address := &Address{
		ID:        AddressID(1),
		DeviceID:  device1.ID,
		IP:        "192.168.1.100",
		IsEnabled: true,
	}
	mockRepo.addresses[address.ID] = address

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	// Try to disable address using wrong device ID
	disabledAddr, err := service.DisableAddress(ctx, device2.ID, address.ID)
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
	is.True(disabledAddr == nil)
}

func TestService_DisableAddress_AddressNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device
	mockRepo.checkOwnershipErr = ErrAddressNotOwnedByDevice

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	disabledAddr, err := service.DisableAddress(ctx, device.ID, AddressID(999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
	is.True(disabledAddr == nil)
}

func testAdminPrincipal() *auth.Principal {
	return auth.NewPrincipal(auth.UserID(1), auth.SessionID(0), auth.AdminRole)
}

func TestService_CreateDevice_ReturnsDevice(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	createdDevice, err := service.CreateDevice(ctx, testAdminPrincipal(), "my-device", nil)
	is.NoErr(err)
	is.True(createdDevice != nil)
	is.Equal(createdDevice.Name, "my-device")
	is.True(createdDevice.ID != 0)
	// No API key is generated on device creation — key must be generated separately.
	is.True(createdDevice.KeyPrefix == nil)
}

func TestService_Authenticate_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	// Create device then generate an API key so the hash is stored in the mock.
	deviceWithPrefix, err := service.CreateDevice(ctx, testAdminPrincipal(), "auth-device", nil)
	is.NoErr(err)

	_, rawKey, err := service.RegenerateAPIKey(ctx, deviceWithPrefix.ID)
	is.NoErr(err)
	is.True(rawKey != "")

	principal, err := service.Authenticate(ctx, rawKey)
	is.NoErr(err)
	is.True(principal != nil)
	is.Equal(principal.DeviceID, deviceWithPrefix.ID)
}

func TestService_Authenticate_InvalidKeyFormat(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	_, err := service.Authenticate(ctx, "invalid-no-prefix")
	is.True(err != nil)
	is.Equal(err, ErrInvalidAPIKey)

	_, err = service.Authenticate(ctx, "wdk") // too short
	is.True(err != nil)
	is.Equal(err, ErrInvalidAPIKey)
}

func TestService_Authenticate_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	// No devices/keys in mock; valid format but unknown key
	rawKey := APIKeyPrefix + "unknownkey123456789012345678901234"
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	_, err := service.Authenticate(ctx, rawKey)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestService_GetDevice_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "single-device"}
	mockRepo.devices[device.ID] = device
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	got, err := service.GetDevice(ctx, device.ID)
	is.NoErr(err)
	is.True(got != nil)
	is.Equal(got.ID, device.ID)
	is.Equal(got.Name, device.Name)
}

func TestService_GetDevice_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceErr = ErrDeviceNotFound
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	got, err := service.GetDevice(ctx, DeviceID(999))
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
	is.True(got == nil)
}

func TestService_GetDevice_RepoError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	repoErr := errors.New("db error")
	mockRepo.getDeviceErr = repoErr
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	got, err := service.GetDevice(ctx, DeviceID(1))
	is.True(err != nil)
	is.Equal(err, repoErr)
	is.True(got == nil)
}

func TestService_DeleteDevice_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.devices[DeviceID(1)] = &Device{ID: DeviceID(1), Name: "to-delete"}
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	err := service.DeleteDevice(ctx, DeviceID(1))
	is.NoErr(err)
	// Mock removes from map
	_, ok := mockRepo.devices[DeviceID(1)]
	is.True(!ok)
}

func TestService_DeleteDevice_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	err := service.DeleteDevice(ctx, DeviceID(999))
	is.True(err != nil)
	is.True(errors.Is(err, ErrDeviceNotFound))
}

type testAddressObserver struct {
	events []AddressEvent
}

func (o *testAddressObserver) OnAddressEvent(_ context.Context, event AddressEvent) {
	o.events = append(o.events, event)
}

func TestService_RegisterAddressActivity_NotifiesObserverOnNewAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.ID, "192.168.1.100", EventSourceManual)
	is.NoErr(err)
	is.Equal(eventType, EventTypeAddressCreated)
	is.True(addr != nil)

	is.Equal(len(observer.events), 1)
	event := observer.events[0]
	is.Equal(event.Type, EventTypeAddressCreated)
	is.Equal(event.AddressID, addr.ID)
	is.Equal(event.DeviceID, device.ID)
	is.True(!event.OccurredAt.IsZero())
}

func TestService_DisableAddress_NotifiesObserver(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	address := &Address{
		ID:        AddressID(1),
		DeviceID:  device.ID,
		IP:        "192.168.1.100",
		IsEnabled: true,
	}
	mockRepo.addresses[address.ID] = address

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	disabledAddr, err := service.DisableAddress(ctx, device.ID, address.ID)
	is.NoErr(err)
	is.True(disabledAddr != nil)

	is.Equal(len(observer.events), 1)
	event := observer.events[0]
	is.Equal(event.Type, EventTypeAddressDisabled)
	is.Equal(event.AddressID, disabledAddr.ID)
	is.Equal(event.DeviceID, device.ID)
	is.True(!event.OccurredAt.IsZero())
}

func TestService_DisableAddresses_NotifiesObserverPerAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()

	address1 := &Address{
		ID:        AddressID(1),
		DeviceID:  DeviceID(1),
		IP:        "192.168.1.1",
		IsEnabled: true,
	}
	address2 := &Address{
		ID:        AddressID(2),
		DeviceID:  DeviceID(2),
		IP:        "192.168.1.2",
		IsEnabled: true,
	}
	mockRepo.addresses[address1.ID] = address1
	mockRepo.addresses[address2.ID] = address2

	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	err := service.DisableAddresses(ctx, []AddressID{address1.ID, address2.ID}, EventSourceManual)
	is.NoErr(err)

	is.Equal(len(observer.events), 2)
	seen := map[AddressID]bool{}
	for _, event := range observer.events {
		is.Equal(event.Type, EventTypeAddressDisabled)
		seen[event.AddressID] = true
		is.True(!event.OccurredAt.IsZero())
	}
	is.True(seen[address1.ID])
	is.True(seen[address2.ID])
}

func TestService_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.createDeviceErr = ErrDuplicateDeviceName
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	device, err := service.CreateDevice(ctx, testAdminPrincipal(), "dup-name", nil)
	is.True(err != nil)
	is.True(errors.Is(err, ErrDuplicateDeviceName))
	is.True(device == nil)
}

func TestService_DisableAddress_DeviceDeleted(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	// Device not in map simulates deleted device; GetDevice returns ErrDeviceNotFound
	mockRepo.getDeviceErr = ErrDeviceNotFound
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	addr, err := service.DisableAddress(ctx, DeviceID(1), AddressID(1))
	is.True(err != nil)
	is.True(errors.Is(err, ErrDeviceNotFound))
	is.True(addr == nil)
}

func TestService_RegenerateAPIKey_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	oldPrefix := "wdk_oldpre"
	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "regen-device", KeyPrefix: &oldPrefix}
	mockRepo.devices[device.ID] = device
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	updatedDevice, rawKey, err := service.RegenerateAPIKey(ctx, device.ID)
	is.NoErr(err)
	is.True(updatedDevice != nil)
	is.Equal(updatedDevice.ID, device.ID)
	is.True(len(rawKey) > len(APIKeyPrefix))
	is.Equal(rawKey[:len(APIKeyPrefix)], APIKeyPrefix)
	// New prefix should be stored and differ from the old one
	is.True(updatedDevice.KeyPrefix != nil && *updatedDevice.KeyPrefix != oldPrefix)
}

func TestService_RegenerateAPIKey_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceErr = ErrDeviceNotFound
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	updatedDevice, rawKey, err := service.RegenerateAPIKey(ctx, DeviceID(999))
	is.True(err != nil)
	is.True(errors.Is(err, ErrDeviceNotFound))
	is.True(updatedDevice == nil)
	is.Equal(rawKey, "")
}

func TestService_RegenerateAPIKey_OldKeyInvalidated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	// Create device then generate first key so old key hash is stored.
	createdDev, err := service.CreateDevice(ctx, testAdminPrincipal(), "rotate-device", nil)
	is.NoErr(err)
	_, oldRawKey, err := service.RegenerateAPIKey(ctx, createdDev.ID)
	is.NoErr(err)
	is.True(oldRawKey != "")

	// Get the device
	var deviceID DeviceID
	for id := range mockRepo.devices {
		deviceID = id
		break
	}

	// Regenerate key
	_, newRawKey, err := service.RegenerateAPIKey(ctx, deviceID)
	is.NoErr(err)
	is.True(newRawKey != oldRawKey)

	// Old key should no longer be in the hash map
	oldHash := HashAPIKey(oldRawKey)
	_, oldKeyFound := mockRepo.apiKeysByHash[oldHash]
	is.True(!oldKeyFound)

	// New key should authenticate
	newHash := HashAPIKey(newRawKey)
	_, newKeyFound := mockRepo.apiKeysByHash[newHash]
	is.True(newKeyFound)
}

func TestService_GetAddressHistory_ValidInput(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	history, err := service.GetAddressHistory(ctx, AddressHistoryQuery{
		DeviceIDs:   []DeviceID{1},
		Granularity: timebucket.GranularityHour,
	})

	is.NoErr(err)
	is.True(history.Buckets != nil)
	is.True(history.Events != nil)
}

func TestService_GetAddressHistory_DefaultParams(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	// Empty query — Validate() should apply defaults
	history, err := service.GetAddressHistory(ctx, AddressHistoryQuery{})

	is.NoErr(err)
	is.True(history.Buckets != nil)
	is.True(history.Events != nil)
}

func TestService_UpdateDevice_RenamesDevice(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	d := &Device{ID: DeviceID(1), Name: "old", DeviceType: DeviceTypeStatic}
	mockRepo.devices[d.ID] = d

	svc := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	updated, err := svc.UpdateDevice(ctx, testAdminPrincipal(), d.ID, UpdateDeviceInput{Name: new("new-name")})

	is.NoErr(err)
	is.Equal(updated.Name, "new-name")
}

func TestService_UpdateDevice_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	svc := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})

	_, err := svc.UpdateDevice(ctx, testAdminPrincipal(), DeviceID(99), UpdateDeviceInput{})

	is.True(errors.Is(err, ErrDeviceNotFound))
}

func TestService_UpdateDevice_InvalidTypePropagated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	d := &Device{ID: DeviceID(1), Name: "d", DeviceType: DeviceTypeStatic}
	mockRepo.devices[d.ID] = d

	svc := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	_, err := svc.UpdateDevice(ctx, testAdminPrincipal(), d.ID, UpdateDeviceInput{DeviceType: new("robot")})

	is.True(errors.Is(err, ErrInvalidDeviceType))
}

func TestService_UpdateDevice_RepoErrorPropagated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	sentinel := errors.New("db gone")
	mockRepo := newMockRepository()
	d := &Device{ID: DeviceID(1), Name: "d", DeviceType: DeviceTypeStatic}
	mockRepo.devices[d.ID] = d
	mockRepo.updateDeviceErr = sentinel

	svc := NewService(mockRepo, slog.New(slog.DiscardHandler), netip.Addr{})
	_, err := svc.UpdateDevice(ctx, testAdminPrincipal(), d.ID, UpdateDeviceInput{})

	is.True(errors.Is(err, sentinel))
}

// mockRepository is a hand-rolled mock implementation of DeviceRepository
type mockRepository struct {
	devices           map[DeviceID]*Device
	addresses         map[AddressID]*Address
	deviceAddressByIP map[string]*Address
	apiKeysByHash     map[string]*Device
	getDeviceErr      error
	createDeviceErr   error
	createAddressErr  error
	getAddressByIPErr error
	enableAddressErr  error
	disableAddressErr error
	checkOwnershipErr error
	updateAPIKeyErr   error
	updateDeviceErr   error
	runInTxFn         func(repository) error
}

// Ensure mockRepository implements repository interface
// var _ Repository = (repository)(nil)
var _ repository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		devices:           make(map[DeviceID]*Device),
		addresses:         make(map[AddressID]*Address),
		deviceAddressByIP: make(map[string]*Address),
		apiKeysByHash:     make(map[string]*Device),
	}
}

func (m *mockRepository) GetDevice(ctx context.Context, id DeviceID) (*Device, error) {
	if m.getDeviceErr != nil {
		return nil, m.getDeviceErr
	}
	device, ok := m.devices[id]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

func (m *mockRepository) CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	if m.createDeviceErr != nil {
		return nil, m.createDeviceErr
	}
	device := &Device{
		ID:   DeviceID(len(m.devices) + 1),
		Name: params.Name,
		// No API key on creation — must be generated separately via UpsertAPIKey.
	}
	m.devices[device.ID] = device
	return device, nil
}

func (m *mockRepository) DeleteDevice(ctx context.Context, id DeviceID) error {
	if _, ok := m.devices[id]; !ok {
		return ErrDeviceNotFound
	}
	delete(m.devices, id)
	for k, v := range m.apiKeysByHash {
		if v.ID == id {
			delete(m.apiKeysByHash, k)
			break
		}
	}
	return nil
}

func (m *mockRepository) GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error) {
	device, ok := m.apiKeysByHash[keyHash]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

func (m *mockRepository) CreateAddress(ctx context.Context, params CreateAddressParams, source EventSource) (*Address, error) {
	if m.createAddressErr != nil {
		return nil, m.createAddressErr
	}
	now := time.Now().UTC()
	address := &Address{
		ID:        AddressID(len(m.addresses) + 1),
		DeviceID:  params.DeviceID,
		IP:        params.IP.String(),
		IsEnabled: true,
		Source:    EventSourceManual,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.addresses[address.ID] = address

	key := address.DeviceID.String() + ":" + address.IP
	m.deviceAddressByIP[key] = address

	return address, nil
}

func (m *mockRepository) GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip netip.Addr) (*Address, error) {
	if m.getAddressByIPErr != nil {
		return nil, m.getAddressByIPErr
	}
	key := deviceID.String() + ":" + ip.String()
	addr, ok := m.deviceAddressByIP[key]
	if !ok {
		return nil, ErrAddressNotFound
	}
	// Return full Address with status so service can call EnableAddress on it
	return addr, nil
}

func (m *mockRepository) DisableAddress(ctx context.Context, addressID AddressID) (*Address, error) {
	if m.disableAddressErr != nil {
		return nil, m.disableAddressErr
	}
	addr, ok := m.addresses[addressID]
	if !ok {
		return nil, ErrAddressNotFound
	}
	addr.IsEnabled = false
	return addr, nil
}

func (m *mockRepository) DisableAddresses(ctx context.Context, addressIDs []AddressID, source EventSource) ([]Address, error) {
	result := make([]Address, 0, len(addressIDs))
	for _, addressID := range addressIDs {
		addr, err := m.DisableAddress(ctx, addressID)
		if err != nil {
			return nil, err
		}
		addr.Source = source
		result = append(result, *addr)
	}
	return result, nil
}

func (m *mockRepository) EnableAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error) {
	if m.enableAddressErr != nil {
		return nil, m.enableAddressErr
	}
	addr, ok := m.addresses[addressID]
	if !ok {
		return nil, ErrAddressNotFound
	}
	addr.IsEnabled = true
	addr.Source = source
	return addr, nil
}

func (m *mockRepository) RefreshAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error) {
	return m.EnableAddress(ctx, addressID, source)
}

func (m *mockRepository) GetAddressByID(ctx context.Context, id AddressID) (*Address, error) {
	addr, ok := m.addresses[id]
	if !ok {
		return nil, ErrAddressNotFound
	}
	return addr, nil
}

func (m *mockRepository) CheckAddressOwnership(ctx context.Context, deviceID DeviceID, addressID AddressID) error {
	if m.checkOwnershipErr != nil {
		return m.checkOwnershipErr
	}
	addr, ok := m.addresses[addressID]
	if !ok || addr.DeviceID != deviceID {
		return ErrAddressNotOwnedByDevice
	}
	return nil
}

func (m *mockRepository) UpsertAPIKey(ctx context.Context, deviceID DeviceID, keyHash string, keyPrefix string) error {
	if m.updateAPIKeyErr != nil {
		return m.updateAPIKeyErr
	}
	device, ok := m.devices[deviceID]
	if !ok {
		return ErrDeviceNotFound
	}
	// Update stored state to reflect the new key
	device.KeyPrefix = &keyPrefix
	// Remove old hash entries for this device, add the new one
	for k, v := range m.apiKeysByHash {
		if v.ID == deviceID {
			delete(m.apiKeysByHash, k)
			break
		}
	}
	m.apiKeysByHash[keyHash] = device
	return nil
}

func (m *mockRepository) DeleteAPIKey(ctx context.Context, deviceID DeviceID) error {
	device, ok := m.devices[deviceID]
	if !ok {
		return ErrNoAPIKey
	}
	if device.KeyPrefix == nil {
		return ErrNoAPIKey
	}
	// Remove the key from the hash map
	for k, v := range m.apiKeysByHash {
		if v.ID == deviceID {
			delete(m.apiKeysByHash, k)
			break
		}
	}
	device.KeyPrefix = nil
	return nil
}

func (m *mockRepository) GetEnabledUniqueIPs(_ context.Context) ([]string, error) {
	ips := make([]string, 0)
	seen := map[string]bool{}
	for _, addr := range m.addresses {
		if addr.IsEnabled && !seen[addr.IP] {
			ips = append(ips, addr.IP)
			seen[addr.IP] = true
		}
	}
	return ips, nil
}

func (m *mockRepository) GetEnabledIPEntries(_ context.Context) ([]IPEntry, error) {
	var entries []IPEntry
	seen := map[string]bool{}
	for _, addr := range m.addresses {
		if addr.IsEnabled && !seen[addr.IP] {
			entries = append(entries, IPEntry{IP: addr.IP, DeviceID: addr.DeviceID, AddressID: addr.ID})
			seen[addr.IP] = true
		}
	}
	return entries, nil
}

func (m *mockRepository) GetAddressHistory(_ context.Context, _ AddressHistoryQuery) (AddressHistory, error) {
	return AddressHistory{Buckets: []AddressEventBucket{}, Events: []AddressStateChange{}}, nil
}

func (m *mockRepository) UpdateDevice(_ context.Context, device *Device) (*Device, error) {
	if m.updateDeviceErr != nil {
		return nil, m.updateDeviceErr
	}
	if _, ok := m.devices[device.ID]; !ok {
		return nil, ErrDeviceNotFound
	}
	m.devices[device.ID] = device
	return device, nil
}

func (m *mockRepository) RunInTx(ctx context.Context, fn func(repository) error) error {
	if m.runInTxFn != nil {
		return m.runInTxFn(m)
	}
	return fn(m)
}

func (m *mockRepository) GetEnabledAddressesForDevice(_ context.Context, deviceID DeviceID) ([]Address, error) {
	var result []Address
	for _, addr := range m.addresses {
		if addr.DeviceID == deviceID && addr.IsEnabled {
			result = append(result, *addr)
		}
	}
	// Sort by UpdatedAt DESC to match repository behavior
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].UpdatedAt.After(result[i].UpdatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}
