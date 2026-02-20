package device

import (
	"context"
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestService_AssignAddress_NewAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	service := NewService(mockRepo)

	addr, wasCreated, err := service.AssignAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)
	is.True(wasCreated)
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.Status)
}

func TestService_AssignAddress_ExistingAddress(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	existingAddr := &AddressWithStatus{
		ID:       AddressID(1),
		DeviceID: device.ID,
		IP:       "192.168.1.100",
		Status:   false,
	}
	key := device.ID.String() + ":192.168.1.100"
	mockRepo.deviceByIP[key] = existingAddr
	mockRepo.addressesWithStatus[existingAddr.ID] = existingAddr

	service := NewService(mockRepo)

	addr, wasCreated, err := service.AssignAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)
	is.True(!wasCreated) // Should not be created, just enabled
	is.True(addr != nil)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(addr.Status) // Should be enabled
}

func TestService_AssignAddress_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceByIDErr = ErrDeviceNotFound

	service := NewService(mockRepo)

	addr, wasCreated, err := service.AssignAddress(ctx, DeviceID(999), "192.168.1.100")
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
	is.True(addr == nil)
	is.True(!wasCreated)
}

func TestService_AssignAddress_TransactionRollback(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	// Simulate transaction failure
	testErr := errors.New("transaction error")
	mockRepo.runInTxFn = func(repo repository) error {
		// Try to create address
		addr, _ := NewAddress(device.ID, "192.168.1.100")
		_, err := repo.CreateAddress(ctx, addr)
		if err != nil {
			return err
		}
		// Return error to trigger rollback
		return testErr
	}

	service := NewService(mockRepo)

	addr, wasCreated, err := service.AssignAddress(ctx, device.ID, "192.168.1.100")
	is.True(err != nil)
	is.Equal(err, testErr)
	is.True(addr == nil)
	is.True(!wasCreated)
}

func TestService_DisableAddress_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	address := &Address{ID: AddressID(1), DeviceID: device.ID, IP: "192.168.1.100"}
	mockRepo.addresses[address.ID] = address

	addressWithStatus := &AddressWithStatus{
		ID:       address.ID,
		DeviceID: device.ID,
		IP:       "192.168.1.100",
		Status:   true,
	}
	mockRepo.addressesWithStatus[address.ID] = addressWithStatus

	service := NewService(mockRepo)

	disabledAddr, err := service.DisableAddress(ctx, device.ID, address.ID)
	is.NoErr(err)
	is.True(disabledAddr != nil)
	is.True(!disabledAddr.Status)
}

func TestService_DisableAddress_OwnershipValidation(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device1 := &Device{ID: DeviceID(1), Name: "device-1"}
	device2 := &Device{ID: DeviceID(2), Name: "device-2"}
	mockRepo.devices[device1.ID] = device1
	mockRepo.devices[device2.ID] = device2

	address := &Address{ID: AddressID(1), DeviceID: device1.ID, IP: "192.168.1.100"}
	mockRepo.addresses[address.ID] = address

	addressWithStatus := &AddressWithStatus{
		ID:       address.ID,
		DeviceID: device1.ID,
		IP:       "192.168.1.100",
		Status:   true,
	}
	mockRepo.addressesWithStatus[address.ID] = addressWithStatus

	service := NewService(mockRepo)

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

	service := NewService(mockRepo)

	disabledAddr, err := service.DisableAddress(ctx, device.ID, AddressID(999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
	is.True(disabledAddr == nil)
}

func TestService_GetAddressesForDevice_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	addr1 := &AddressWithStatus{
		ID:       AddressID(1),
		DeviceID: device.ID,
		IP:       "192.168.1.1",
		Status:   true,
	}
	addr2 := &AddressWithStatus{
		ID:       AddressID(2),
		DeviceID: device.ID,
		IP:       "192.168.1.2",
		Status:   false,
	}
	mockRepo.addressesWithStatus[addr1.ID] = addr1
	mockRepo.addressesWithStatus[addr2.ID] = addr2

	service := NewService(mockRepo)

	addresses, err := service.GetAddressesForDevice(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 2)
}

func TestService_GetAddressesForDevice_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getDeviceByIDErr = ErrDeviceNotFound

	service := NewService(mockRepo)

	addresses, err := service.GetAddressesForDevice(ctx, DeviceID(999))
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
	is.True(addresses == nil)
}

func TestService_GetAddressesForDevice_Empty(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	device := &Device{ID: DeviceID(1), Name: "test-device"}
	mockRepo.devices[device.ID] = device

	service := NewService(mockRepo)

	addresses, err := service.GetAddressesForDevice(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 0)
}

func TestService_CreateDevice_ReturnsDeviceAndRawKey(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo)

	deviceWithPrefix, rawKey, err := service.CreateDevice(ctx, "my-device")
	is.NoErr(err)
	is.True(deviceWithPrefix != nil)
	is.Equal(deviceWithPrefix.Name, "my-device")
	is.True(deviceWithPrefix.ID != 0)
	is.True(deviceWithPrefix.KeyPrefix != "")
	is.True(len(rawKey) > len(APIKeyPrefix))
	is.Equal(rawKey[:len(APIKeyPrefix)], APIKeyPrefix)
}

func TestService_Authenticate_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo)

	// Create device via service so API key is stored in mock
	deviceWithPrefix, rawKey, err := service.CreateDevice(ctx, "auth-device")
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
	service := NewService(mockRepo)

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
	service := NewService(mockRepo)

	_, err := service.Authenticate(ctx, rawKey)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestService_GetDevices_ReturnsListWithPrefix(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.devices[DeviceID(1)] = &Device{ID: DeviceID(1), Name: "d1"}
	mockRepo.devices[DeviceID(2)] = &Device{ID: DeviceID(2), Name: "d2"}
	service := NewService(mockRepo)

	list, err := service.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(list), 2)
	for i := range list {
		is.True(list[i].KeyPrefix != "")
		is.Equal(list[i].KeyPrefix, "wdk_xxxxxxxx")
	}
}

// mockRepository is a hand-rolled mock implementation of DeviceRepository
type mockRepository struct {
	devices             map[DeviceID]*Device
	addresses           map[AddressID]*Address
	addressesWithStatus map[AddressID]*AddressWithStatus
	deviceByIP          map[string]*AddressWithStatus // key: "deviceID:ip"
	apiKeysByHash       map[string]*Device            // keyHash -> device (for GetDeviceByAPIKeyHash)
	getDeviceByIDErr    error
	createAddressErr    error
	getAddressByIPErr   error
	enableAddressErr    error
	disableAddressErr   error
	listAddressesErr    error
	checkOwnershipErr   error
	runInTxFn           func(repository) error
}

// Ensure mockRepository implements repository interface
// var _ Repository = (repository)(nil)
var _ repository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		devices:             make(map[DeviceID]*Device),
		addresses:           make(map[AddressID]*Address),
		addressesWithStatus: make(map[AddressID]*AddressWithStatus),
		deviceByIP:          make(map[string]*AddressWithStatus),
		apiKeysByHash:       make(map[string]*Device),
	}
}

func (m *mockRepository) GetDeviceByID(ctx context.Context, id DeviceID) (*Device, error) {
	if m.getDeviceByIDErr != nil {
		return nil, m.getDeviceByIDErr
	}
	device, ok := m.devices[id]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

func (m *mockRepository) CreateDevice(ctx context.Context, device *Device) (*Device, error) {
	device.ID = DeviceID(len(m.devices) + 1)
	m.devices[device.ID] = device
	return device, nil
}

func (m *mockRepository) GetDevices(ctx context.Context) ([]DeviceWithAPIKeyPrefix, error) {
	devices := make([]DeviceWithAPIKeyPrefix, 0, len(m.devices))
	for _, d := range m.devices {
		devices = append(devices, DeviceWithAPIKeyPrefix{Device: *d, KeyPrefix: "wdk_xxxxxxxx"})
	}
	return devices, nil
}

func (m *mockRepository) CreateDeviceAPIKey(ctx context.Context, apiKey *APIKey) (*APIKey, error) {
	device, ok := m.devices[apiKey.DeviceID]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	m.apiKeysByHash[apiKey.KeyHash] = device
	return apiKey, nil
}

func (m *mockRepository) GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error) {
	device, ok := m.apiKeysByHash[keyHash]
	if !ok {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

func (m *mockRepository) CreateAddress(ctx context.Context, address *Address) (*Address, error) {
	if m.createAddressErr != nil {
		return nil, m.createAddressErr
	}
	address.ID = AddressID(len(m.addresses) + 1)
	m.addresses[address.ID] = address
	return address, nil
}

func (m *mockRepository) GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip string) (*AddressWithStatus, error) {
	if m.getAddressByIPErr != nil {
		return nil, m.getAddressByIPErr
	}
	key := deviceID.String() + ":" + ip
	addr, ok := m.deviceByIP[key]
	if !ok {
		return nil, ErrAddressNotFound
	}
	return addr, nil
}

func (m *mockRepository) ListAddresses(ctx context.Context, deviceID DeviceID) ([]AddressWithStatus, error) {
	if m.listAddressesErr != nil {
		return nil, m.listAddressesErr
	}
	addresses := make([]AddressWithStatus, 0)
	for _, addr := range m.addressesWithStatus {
		if addr.DeviceID == deviceID {
			addresses = append(addresses, *addr)
		}
	}
	return addresses, nil
}

func (m *mockRepository) DisableAddress(ctx context.Context, addressID AddressID) (*AddressWithStatus, error) {
	if m.disableAddressErr != nil {
		return nil, m.disableAddressErr
	}
	addr, ok := m.addressesWithStatus[addressID]
	if !ok {
		return nil, ErrAddressNotFound
	}
	addr.Status = false
	return addr, nil
}

func (m *mockRepository) EnableAddress(ctx context.Context, addressID AddressID) (*AddressWithStatus, error) {
	if m.enableAddressErr != nil {
		return nil, m.enableAddressErr
	}
	addr, ok := m.addressesWithStatus[addressID]
	if !ok {
		// Create new address with status if it doesn't exist
		baseAddr, ok := m.addresses[addressID]
		if !ok {
			return nil, ErrAddressNotFound
		}
		addr = &AddressWithStatus{
			ID:       baseAddr.ID,
			DeviceID: baseAddr.DeviceID,
			IP:       baseAddr.IP,
			Status:   true,
		}
		m.addressesWithStatus[addressID] = addr
	} else {
		addr.Status = true
	}
	return addr, nil
}

func (m *mockRepository) GetAddressWithStatus(ctx context.Context, addressID AddressID) (*AddressWithStatus, error) {
	addr, ok := m.addressesWithStatus[addressID]
	if !ok {
		return nil, ErrAddressNotFound
	}
	return addr, nil
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
	addr, ok := m.addressesWithStatus[addressID]
	if !ok {
		addr2, ok2 := m.addresses[addressID]
		if !ok2 {
			return ErrAddressNotOwnedByDevice
		}
		if addr2.DeviceID != deviceID {
			return ErrAddressNotOwnedByDevice
		}
		return nil
	}
	if addr.DeviceID != deviceID {
		return ErrAddressNotOwnedByDevice
	}
	return nil
}

func (m *mockRepository) RunInTx(ctx context.Context, fn func(repository) error) error {
	if m.runInTxFn != nil {
		return m.runInTxFn(m)
	}
	return fn(m)
}
