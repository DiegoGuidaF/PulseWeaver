//go:build test

package device_test

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/matryer/is"
)

func TestService_RegisterAddressActivity_NewAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	service := newService(mockRepo)

	addr, eventType, err := service.RegisterAddressActivity(ctx, dev.ID, "192.168.1.100", device.EventSourceManual)
	is.NoErr(err)
	is.Equal(eventType, device.EventTypeAddressCreated)
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.IsEnabled)
}

func newService(mockRepo *mockRepository) *device.Service {
	return newServiceWithTrustedProxy(mockRepo, netip.Addr{})
}

func newServiceWithTrustedProxy(mockRepo *mockRepository, proxy netip.Addr) *device.Service {
	return device.NewService(mockRepo, testutils.NoopTransactor{}, slog.New(slog.DiscardHandler), proxy)
}

func TestService_RegisterAddressActivity_ExistingAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	existingAddr := &device.Address{
		ID:        device.AddressID(1),
		DeviceID:  dev.ID,
		IP:        "192.168.1.100",
		IsEnabled: false,
	}
	key := dev.ID.String() + ":192.168.1.100"
	mockRepo.addresses[existingAddr.ID] = existingAddr
	mockRepo.deviceAddressByIP[key] = existingAddr

	service := newService(mockRepo)

	addr, eventType, err := service.RegisterAddressActivity(ctx, dev.ID, "192.168.1.100", device.EventSourceManual)
	is.NoErr(err)
	is.Equal(eventType, device.EventTypeAddressEnabled) // Address already existed, we just enabled it
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.IsEnabled) // Should be enabled
}

func TestService_RegisterAddressActivity_ExistingEnabledAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	existingAddr := &device.Address{
		ID:        device.AddressID(1),
		DeviceID:  dev.ID,
		IP:        "192.168.1.100",
		IsEnabled: true, // already enabled
	}
	key := dev.ID.String() + ":192.168.1.100"
	mockRepo.addresses[existingAddr.ID] = existingAddr
	mockRepo.deviceAddressByIP[key] = existingAddr

	service := newService(mockRepo)
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	addr, eventType, err := service.RegisterAddressActivity(ctx, dev.ID, "192.168.1.100", device.EventSourceHeartbeat)
	is.NoErr(err)
	is.Equal(eventType, device.EventTypeAddressRefreshed)
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.IsEnabled)

	is.Equal(len(observer.events), 1)
	event := observer.events[0]
	is.Equal(event.Type, device.EventTypeAddressRefreshed)
	is.Equal(event.AddressID, addr.ID)
	is.Equal(event.DeviceID, dev.ID)
	is.True(!event.OccurredAt.IsZero())
}

func TestService_RegisterAddressActivity_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceErr = device.ErrDeviceNotFound

	service := newService(mockRepo)

	addr, eventType, err := service.RegisterAddressActivity(ctx, device.DeviceID(999), "192.168.1.100", device.EventSourceManual)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
	is.True(addr == nil)
	is.Equal(eventType, device.EventType(""))
}

func TestService_RegisterAddressActivity_RejectsTrustedProxyIP(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	service := newServiceWithTrustedProxy(mockRepo, netip.MustParseAddr("10.1.2.3"))

	addr, eventType, err := service.RegisterAddressActivity(ctx, dev.ID, "10.1.2.3", device.EventSourceHeartbeat)
	is.True(errors.Is(err, device.ErrTrustedProxyIPRejected))
	is.True(addr == nil)
	is.Equal(eventType, device.EventType(""))
}

func TestService_DisableAddress_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	address := &device.Address{
		ID:        device.AddressID(1),
		DeviceID:  dev.ID,
		IP:        "192.168.1.100",
		IsEnabled: true,
	}
	mockRepo.addresses[address.ID] = address

	service := newService(mockRepo)

	disabledAddr, err := service.DisableAddress(ctx, dev.ID, address.ID)
	is.NoErr(err)
	is.True(disabledAddr != nil)
	is.True(!disabledAddr.IsEnabled)
}

func TestService_DisableAddress_OwnershipValidation(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device1 := &device.Device{ID: device.DeviceID(1), Name: "device-1"}
	device2 := &device.Device{ID: device.DeviceID(2), Name: "device-2"}
	mockRepo.devices[device1.ID] = device1
	mockRepo.devices[device2.ID] = device2

	address := &device.Address{
		ID:        device.AddressID(1),
		DeviceID:  device1.ID,
		IP:        "192.168.1.100",
		IsEnabled: true,
	}
	mockRepo.addresses[address.ID] = address

	service := newService(mockRepo)

	// Try to disable address using wrong device ID
	disabledAddr, err := service.DisableAddress(ctx, device2.ID, address.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotOwnedByDevice)
	is.True(disabledAddr == nil)
}

func TestService_DisableAddress_AddressNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev
	mockRepo.checkOwnershipErr = device.ErrAddressNotOwnedByDevice

	service := newService(mockRepo)

	disabledAddr, err := service.DisableAddress(ctx, dev.ID, device.AddressID(999))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotOwnedByDevice)
	is.True(disabledAddr == nil)
}

func testAdminPrincipal() *auth.Principal {
	return auth.NewPrincipal(auth.UserID(1), auth.SessionID(0), auth.AdminRole)
}

func TestService_CreateDevice_ReturnsDevice(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

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
	service := newService(mockRepo)

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
	service := newService(mockRepo)

	_, err := service.Authenticate(ctx, "invalid-no-prefix")
	is.True(err != nil)
	is.Equal(err, device.ErrInvalidAPIKey)

	_, err = service.Authenticate(ctx, "wdk") // too short
	is.True(err != nil)
	is.Equal(err, device.ErrInvalidAPIKey)
}

func TestService_Authenticate_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	// No devices/keys in mock; valid format but unknown key
	rawKey := device.APIKeyPrefix + "unknownkey123456789012345678901234"
	service := newService(mockRepo)

	_, err := service.Authenticate(ctx, rawKey)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestService_GetDevice_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &device.Device{ID: device.DeviceID(1), Name: "single-device"}
	mockRepo.devices[device.ID] = device
	service := newService(mockRepo)

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
	mockRepo.getDeviceErr = device.ErrDeviceNotFound
	service := newService(mockRepo)

	got, err := service.GetDevice(ctx, device.DeviceID(999))
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
	is.True(got == nil)
}

func TestService_GetDevice_RepoError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	repoErr := errors.New("db error")
	mockRepo.getDeviceErr = repoErr
	service := newService(mockRepo)

	got, err := service.GetDevice(ctx, device.DeviceID(1))
	is.True(err != nil)
	is.Equal(err, repoErr)
	is.True(got == nil)
}

func TestService_DeleteDevice_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.devices[device.DeviceID(1)] = &device.Device{ID: device.DeviceID(1), Name: "to-delete"}
	service := newService(mockRepo)

	err := service.DeleteDevice(ctx, device.DeviceID(1))
	is.NoErr(err)
	// Mock removes from map
	_, ok := mockRepo.devices[device.DeviceID(1)]
	is.True(!ok)
}

func TestService_DeleteDevice_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	err := service.DeleteDevice(ctx, device.DeviceID(999))
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDeviceNotFound))
}

type testAddressObserver struct {
	events []device.AddressEvent
}

func (o *testAddressObserver) OnAddressEvent(_ context.Context, event device.AddressEvent) {
	o.events = append(o.events, event)
}

func TestService_RegisterAddressActivity_NotifiesObserverOnNewAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	service := newService(mockRepo)
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	addr, eventType, err := service.RegisterAddressActivity(ctx, dev.ID, "192.168.1.100", device.EventSourceManual)
	is.NoErr(err)
	is.Equal(eventType, device.EventTypeAddressCreated)
	is.True(addr != nil)

	is.Equal(len(observer.events), 1)
	event := observer.events[0]
	is.Equal(event.Type, device.EventTypeAddressCreated)
	is.Equal(event.AddressID, addr.ID)
	is.Equal(event.DeviceID, dev.ID)
	is.True(!event.OccurredAt.IsZero())
}

func TestService_DisableAddress_NotifiesObserver(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "test-device"}
	mockRepo.devices[dev.ID] = dev

	address := &device.Address{
		ID:        device.AddressID(1),
		DeviceID:  dev.ID,
		IP:        "192.168.1.100",
		IsEnabled: true,
	}
	mockRepo.addresses[address.ID] = address

	service := newService(mockRepo)
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	disabledAddr, err := service.DisableAddress(ctx, dev.ID, address.ID)
	is.NoErr(err)
	is.True(disabledAddr != nil)

	is.Equal(len(observer.events), 1)
	event := observer.events[0]
	is.Equal(event.Type, device.EventTypeAddressDisabled)
	is.Equal(event.AddressID, disabledAddr.ID)
	is.Equal(event.DeviceID, dev.ID)
	is.True(!event.OccurredAt.IsZero())
}

func TestService_DisableAddresses_NotifiesObserverPerAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()

	address1 := &device.Address{
		ID:        device.AddressID(1),
		DeviceID:  device.DeviceID(1),
		IP:        "192.168.1.1",
		IsEnabled: true,
	}
	address2 := &device.Address{
		ID:        device.AddressID(2),
		DeviceID:  device.DeviceID(2),
		IP:        "192.168.1.2",
		IsEnabled: true,
	}
	mockRepo.addresses[address1.ID] = address1
	mockRepo.addresses[address2.ID] = address2

	service := newService(mockRepo)
	observer := &testAddressObserver{}
	service.AddAddressObserver(observer)

	err := service.DisableAddresses(ctx, []device.AddressID{address1.ID, address2.ID}, device.EventSourceManual)
	is.NoErr(err)

	is.Equal(len(observer.events), 2)
	seen := map[device.AddressID]bool{}
	for _, event := range observer.events {
		is.Equal(event.Type, device.EventTypeAddressDisabled)
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
	mockRepo.createDeviceErr = device.ErrDuplicateDeviceName
	service := newService(mockRepo)

	dev, err := service.CreateDevice(ctx, testAdminPrincipal(), "dup-name", nil)
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDuplicateDeviceName))
	is.True(dev == nil)
}

func TestService_DisableAddress_DeviceDeleted(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	// Device not in map simulates deleted device; GetDevice returns  device.ErrDeviceNotFound
	mockRepo.getDeviceErr = device.ErrDeviceNotFound
	service := newService(mockRepo)

	addr, err := service.DisableAddress(ctx, device.DeviceID(1), device.AddressID(1))
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDeviceNotFound))
	is.True(addr == nil)
}

func TestService_RegenerateAPIKey_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	oldPrefix := "wdk_oldpre"
	mockRepo := newMockRepository()
	dev := &device.Device{ID: device.DeviceID(1), Name: "regen-device", KeyPrefix: &oldPrefix}
	mockRepo.devices[dev.ID] = dev
	service := newService(mockRepo)

	updatedDevice, rawKey, err := service.RegenerateAPIKey(ctx, dev.ID)
	is.NoErr(err)
	is.True(updatedDevice != nil)
	is.Equal(updatedDevice.ID, dev.ID)
	is.True(len(rawKey) > len(device.APIKeyPrefix))
	is.Equal(rawKey[:len(device.APIKeyPrefix)], device.APIKeyPrefix)
	// New prefix should be stored and differ from the old one
	is.True(updatedDevice.KeyPrefix != nil && *updatedDevice.KeyPrefix != oldPrefix)
}

func TestService_RegenerateAPIKey_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceErr = device.ErrDeviceNotFound
	service := newService(mockRepo)

	updatedDevice, rawKey, err := service.RegenerateAPIKey(ctx, device.DeviceID(999))
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDeviceNotFound))
	is.True(updatedDevice == nil)
	is.Equal(rawKey, "")
}

func TestService_RegenerateAPIKey_OldKeyInvalidated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	// Create device then generate first key so old key hash is stored.
	createdDev, err := service.CreateDevice(ctx, testAdminPrincipal(), "rotate-device", nil)
	is.NoErr(err)
	_, oldRawKey, err := service.RegenerateAPIKey(ctx, createdDev.ID)
	is.NoErr(err)
	is.True(oldRawKey != "")

	// Get the device
	var deviceID device.DeviceID
	for id := range mockRepo.devices {
		deviceID = id
		break
	}

	// Regenerate key
	_, newRawKey, err := service.RegenerateAPIKey(ctx, deviceID)
	is.NoErr(err)
	is.True(newRawKey != oldRawKey)

	// Old key should no longer be in the hash map
	oldHash := device.HashAPIKey(oldRawKey)
	_, oldKeyFound := mockRepo.apiKeysByHash[oldHash]
	is.True(!oldKeyFound)

	// New key should authenticate
	newHash := device.HashAPIKey(newRawKey)
	_, newKeyFound := mockRepo.apiKeysByHash[newHash]
	is.True(newKeyFound)
}

func TestService_GetAddressHistory_ValidInput(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	history, err := service.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{1},
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
	service := newService(mockRepo)

	// Empty query — Validate() should apply defaults
	history, err := service.GetAddressHistory(ctx, device.AddressHistoryQuery{})

	is.NoErr(err)
	is.True(history.Buckets != nil)
	is.True(history.Events != nil)
}

func TestService_UpdateDevice_RenamesDevice(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	d := &device.Device{ID: device.DeviceID(1), Name: "old", DeviceType: device.DeviceTypeStatic}
	mockRepo.devices[d.ID] = d

	svc := newService(mockRepo)
	updated, err := svc.UpdateDevice(ctx, testAdminPrincipal(), d.ID, device.UpdateDeviceInput{Name: new("new-name")})

	is.NoErr(err)
	is.Equal(updated.Name, "new-name")
}

func TestService_UpdateDevice_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	svc := newService(mockRepo)

	_, err := svc.UpdateDevice(ctx, testAdminPrincipal(), device.DeviceID(99), device.UpdateDeviceInput{})

	is.True(errors.Is(err, device.ErrDeviceNotFound))
}

func TestService_UpdateDevice_InvalidTypePropagated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	d := &device.Device{ID: device.DeviceID(1), Name: "d", DeviceType: device.DeviceTypeStatic}
	mockRepo.devices[d.ID] = d

	svc := newService(mockRepo)
	_, err := svc.UpdateDevice(ctx, testAdminPrincipal(), d.ID, device.UpdateDeviceInput{DeviceType: new("robot")})

	is.True(errors.Is(err, device.ErrInvalidDeviceType))
}

func TestService_UpdateDevice_RepoErrorPropagated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	sentinel := errors.New("db gone")
	mockRepo := newMockRepository()
	d := &device.Device{ID: device.DeviceID(1), Name: "d", DeviceType: device.DeviceTypeStatic}
	mockRepo.devices[d.ID] = d
	mockRepo.updateDeviceErr = sentinel

	svc := newService(mockRepo)
	_, err := svc.UpdateDevice(ctx, testAdminPrincipal(), d.ID, device.UpdateDeviceInput{})

	is.True(errors.Is(err, sentinel))
}

// mockRepository is a hand-rolled mock implementation of DeviceRepository
type mockRepository struct {
	devices           map[device.DeviceID]*device.Device
	addresses         map[device.AddressID]*device.Address
	deviceAddressByIP map[string]*device.Address
	apiKeysByHash     map[string]*device.Device
	getDeviceErr      error
	createDeviceErr   error
	createAddressErr  error
	getAddressByIPErr error
	enableAddressErr  error
	disableAddressErr error
	checkOwnershipErr error
	updateAPIKeyErr   error
	updateDeviceErr   error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		devices:           make(map[device.DeviceID]*device.Device),
		addresses:         make(map[device.AddressID]*device.Address),
		deviceAddressByIP: make(map[string]*device.Address),
		apiKeysByHash:     make(map[string]*device.Device),
	}
}

func (m *mockRepository) GetDevice(ctx context.Context, id device.DeviceID) (*device.Device, error) {
	if m.getDeviceErr != nil {
		return nil, m.getDeviceErr
	}
	dev, ok := m.devices[id]
	if !ok {
		return nil, device.ErrDeviceNotFound
	}
	return dev, nil
}

func (m *mockRepository) CreateDevice(ctx context.Context, params device.CreateDeviceParams) (*device.Device, error) {
	if m.createDeviceErr != nil {
		return nil, m.createDeviceErr
	}
	dev := &device.Device{
		ID:   device.DeviceID(len(m.devices) + 1),
		Name: params.Name,
		// No API key on creation — must be generated separately via UpsertAPIKey.
	}
	m.devices[dev.ID] = dev
	return dev, nil
}

func (m *mockRepository) DeleteDevice(ctx context.Context, id device.DeviceID) error {
	if _, ok := m.devices[id]; !ok {
		return device.ErrDeviceNotFound
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

func (m *mockRepository) GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*device.Device, error) {
	dev, ok := m.apiKeysByHash[keyHash]
	if !ok {
		return nil, device.ErrDeviceNotFound
	}
	return dev, nil
}

func (m *mockRepository) CreateAddress(ctx context.Context, params device.CreateAddressParams, source device.EventSource) (*device.Address, error) {
	if m.createAddressErr != nil {
		return nil, m.createAddressErr
	}
	now := time.Now().UTC()
	address := &device.Address{
		ID:        device.AddressID(len(m.addresses) + 1),
		DeviceID:  params.DeviceID,
		IP:        params.IP.String(),
		IsEnabled: true,
		Source:    device.EventSourceManual,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.addresses[address.ID] = address

	key := address.DeviceID.String() + ":" + address.IP
	m.deviceAddressByIP[key] = address

	return address, nil
}

func (m *mockRepository) GetAddressForDeviceByIP(ctx context.Context, deviceID device.DeviceID, ip netip.Addr) (*device.Address, error) {
	if m.getAddressByIPErr != nil {
		return nil, m.getAddressByIPErr
	}
	key := deviceID.String() + ":" + ip.String()
	addr, ok := m.deviceAddressByIP[key]
	if !ok {
		return nil, device.ErrAddressNotFound
	}
	// Return full Address with status so service can call EnableAddress on it
	return addr, nil
}

func (m *mockRepository) DisableAddress(ctx context.Context, addressID device.AddressID) (*device.Address, error) {
	if m.disableAddressErr != nil {
		return nil, m.disableAddressErr
	}
	addr, ok := m.addresses[addressID]
	if !ok {
		return nil, device.ErrAddressNotFound
	}
	addr.IsEnabled = false
	return addr, nil
}

func (m *mockRepository) DisableAddresses(ctx context.Context, addressIDs []device.AddressID, source device.EventSource) ([]device.Address, error) {
	result := make([]device.Address, 0, len(addressIDs))
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

func (m *mockRepository) EnableAddress(ctx context.Context, addressID device.AddressID, source device.EventSource) (*device.Address, error) {
	if m.enableAddressErr != nil {
		return nil, m.enableAddressErr
	}
	addr, ok := m.addresses[addressID]
	if !ok {
		return nil, device.ErrAddressNotFound
	}
	addr.IsEnabled = true
	addr.Source = source
	return addr, nil
}

func (m *mockRepository) RefreshAddress(ctx context.Context, addressID device.AddressID, source device.EventSource) (*device.Address, error) {
	return m.EnableAddress(ctx, addressID, source)
}

func (m *mockRepository) GetAddressByID(ctx context.Context, id device.AddressID) (*device.Address, error) {
	addr, ok := m.addresses[id]
	if !ok {
		return nil, device.ErrAddressNotFound
	}
	return addr, nil
}

func (m *mockRepository) CheckAddressOwnership(ctx context.Context, deviceID device.DeviceID, addressID device.AddressID) error {
	if m.checkOwnershipErr != nil {
		return m.checkOwnershipErr
	}
	addr, ok := m.addresses[addressID]
	if !ok || addr.DeviceID != deviceID {
		return device.ErrAddressNotOwnedByDevice
	}
	return nil
}

func (m *mockRepository) UpsertAPIKey(ctx context.Context, deviceID device.DeviceID, keyHash string, keyPrefix string) error {
	if m.updateAPIKeyErr != nil {
		return m.updateAPIKeyErr
	}
	dev, ok := m.devices[deviceID]
	if !ok {
		return device.ErrDeviceNotFound
	}
	// Update stored state to reflect the new key
	dev.KeyPrefix = &keyPrefix
	// Remove old hash entries for this device, add the new one
	for k, v := range m.apiKeysByHash {
		if v.ID == deviceID {
			delete(m.apiKeysByHash, k)
			break
		}
	}
	m.apiKeysByHash[keyHash] = dev
	return nil
}

func (m *mockRepository) DeleteAPIKey(ctx context.Context, deviceID device.DeviceID) error {
	dev, ok := m.devices[deviceID]
	if !ok {
		return device.ErrNoAPIKey
	}
	if dev.KeyPrefix == nil {
		return device.ErrNoAPIKey
	}
	// Remove the key from the hash map
	for k, v := range m.apiKeysByHash {
		if v.ID == deviceID {
			delete(m.apiKeysByHash, k)
			break
		}
	}
	dev.KeyPrefix = nil
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

func (m *mockRepository) GetEnabledIPEntries(_ context.Context) ([]device.IPEntry, error) {
	var entries []device.IPEntry
	seen := map[string]bool{}
	for _, addr := range m.addresses {
		if addr.IsEnabled && !seen[addr.IP] {
			entries = append(entries, device.IPEntry{IP: addr.IP, DeviceID: addr.DeviceID, AddressID: addr.ID})
			seen[addr.IP] = true
		}
	}
	return entries, nil
}

func (m *mockRepository) GetAddressHistory(_ context.Context, _ device.AddressHistoryQuery) (device.AddressHistory, error) {
	return device.AddressHistory{Buckets: []device.AddressEventBucket{}, Events: []device.AddressStateChange{}}, nil
}

func (m *mockRepository) UpdateDevice(_ context.Context, dev *device.Device) (*device.Device, error) {
	if m.updateDeviceErr != nil {
		return nil, m.updateDeviceErr
	}
	if _, ok := m.devices[dev.ID]; !ok {
		return nil, device.ErrDeviceNotFound
	}
	m.devices[dev.ID] = dev
	return dev, nil
}

func (m *mockRepository) GetEnabledAddressesForDevice(_ context.Context, deviceID device.DeviceID) ([]device.Address, error) {
	var result []device.Address
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
