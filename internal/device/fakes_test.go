//go:build test

package device_test

import (
	"context"
	"net/netip"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type testAddressObserver struct {
	events []device.AddressEvent
}

func (o *testAddressObserver) OnAddressEvent(_ context.Context, event device.AddressEvent) {
	o.events = append(o.events, event)
}

// mockRepository is a hand-rolled mock implementation of the repository interface
type mockRepository struct {
	devices           map[ids.DeviceID]*device.Device
	addresses         map[ids.AddressID]*device.Address
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
		devices:           make(map[ids.DeviceID]*device.Device),
		addresses:         make(map[ids.AddressID]*device.Address),
		deviceAddressByIP: make(map[string]*device.Address),
		apiKeysByHash:     make(map[string]*device.Device),
	}
}

func (m *mockRepository) GetDevice(ctx context.Context, id ids.DeviceID) (*device.Device, error) {
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
		ID:   ids.DeviceID(len(m.devices) + 1),
		Name: params.Name,
		// No API key on creation — must be generated separately via UpsertAPIKey.
	}
	m.devices[dev.ID] = dev
	return dev, nil
}

func (m *mockRepository) DeleteDevice(ctx context.Context, id ids.DeviceID) error {
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

func (m *mockRepository) SetDeviceDisabled(ctx context.Context, id ids.DeviceID, disabled bool) error {
	dev, ok := m.devices[id]
	if !ok {
		return device.ErrDeviceNotFound
	}
	if disabled {
		now := time.Now().UTC()
		dev.DisabledAt = &now
	} else {
		dev.DisabledAt = nil
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
		ID:        ids.AddressID(len(m.addresses) + 1),
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

func (m *mockRepository) GetAddressForDeviceByIP(ctx context.Context, deviceID ids.DeviceID, ip netip.Addr) (*device.Address, error) {
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

func (m *mockRepository) DisableAddress(ctx context.Context, addressID ids.AddressID) (*device.Address, error) {
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

func (m *mockRepository) DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source device.EventSource) ([]device.Address, error) {
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

func (m *mockRepository) EnableAddress(ctx context.Context, addressID ids.AddressID, source device.EventSource) (*device.Address, error) {
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

func (m *mockRepository) RefreshAddress(ctx context.Context, addressID ids.AddressID, source device.EventSource) (*device.Address, error) {
	return m.EnableAddress(ctx, addressID, source)
}

func (m *mockRepository) CheckAddressOwnership(ctx context.Context, deviceID ids.DeviceID, addressID ids.AddressID) error {
	if m.checkOwnershipErr != nil {
		return m.checkOwnershipErr
	}
	addr, ok := m.addresses[addressID]
	if !ok || addr.DeviceID != deviceID {
		return device.ErrAddressNotOwnedByDevice
	}
	return nil
}

func (m *mockRepository) UpsertAPIKey(ctx context.Context, deviceID ids.DeviceID, keyHash string, keyPrefix string) error {
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

func (m *mockRepository) DeleteAPIKey(ctx context.Context, deviceID ids.DeviceID) error {
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

func (m *mockRepository) GetEnabledAddressesForDevice(_ context.Context, deviceID ids.DeviceID) ([]device.Address, error) {
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

func (m *mockRepository) GetDeviceIDsByOwner(_ context.Context, ownerID ids.UserID) ([]ids.DeviceID, error) {
	var result []ids.DeviceID
	for _, dev := range m.devices {
		if dev.OwnerID == ownerID {
			result = append(result, dev.ID)
		}
	}
	return result, nil
}
