//go:build test

package device_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
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

func TestService_DisableAddress_DeviceDeleted(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	// Device not in map simulates deleted device; GetDevice returns device.ErrDeviceNotFound
	mockRepo.getDeviceErr = device.ErrDeviceNotFound
	service := newService(mockRepo)

	addr, err := service.DisableAddress(ctx, device.DeviceID(1), device.AddressID(1))
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDeviceNotFound))
	is.True(addr == nil)
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
