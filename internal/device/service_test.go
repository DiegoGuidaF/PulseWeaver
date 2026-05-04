//go:build test

package device_test

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func newService(mockRepo *mockRepository) *device.Service {
	return newServiceWithTrustedProxy(mockRepo, netip.Addr{})
}

func newServiceWithTrustedProxy(mockRepo *mockRepository, proxy netip.Addr) *device.Service {
	return device.NewService(mockRepo, testutils.NoopTransactor{}, slog.New(slog.DiscardHandler), proxy)
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
	dev := &device.Device{ID: device.DeviceID(1), Name: "single-device"}
	mockRepo.devices[dev.ID] = dev
	service := newService(mockRepo)

	got, err := service.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.True(got != nil)
	is.Equal(got.ID, dev.ID)
	is.Equal(got.Name, dev.Name)
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

func TestService_DeleteDevice_ShouldDisableEnabledAddresses(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	addressIDEnabled := device.AddressID(1)
	addressIDDisabled := device.AddressID(2)

	d := &device.Device{ID: device.DeviceID(1), Name: "to-delete"}
	mockRepo.devices[d.ID] = d
	mockRepo.addresses[addressIDEnabled] = &device.Address{ID: addressIDEnabled, DeviceID: d.ID, IsEnabled: true}
	mockRepo.addresses[addressIDDisabled] = &device.Address{ID: addressIDDisabled, DeviceID: d.ID, IsEnabled: false}
	service := newService(mockRepo)

	err := service.DeleteDevice(ctx, d.ID)
	is.NoErr(err)

	// Mock has disabled enabled addresses
	is.Equal(mockRepo.addresses[addressIDEnabled].IsEnabled, false)
	is.Equal(mockRepo.addresses[addressIDDisabled].IsEnabled, false)
}

func TestService_DeleteDevice_ShouldRemoveAnyAPIKey(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	apiKey := "anapikey"

	d := &device.Device{ID: device.DeviceID(1), Name: "to-delete"}
	mockRepo.devices[d.ID] = d
	mockRepo.apiKeysByHash[apiKey] = d
	service := newService(mockRepo)

	err := service.DeleteDevice(ctx, d.ID)
	is.NoErr(err)

	// Mock has disabled enabled addresses
	_, ok := mockRepo.apiKeysByHash[apiKey]
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

func TestService_UpdateDevice_RenamesDevice(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	d := &device.Device{ID: device.DeviceID(1), Name: "old", DeviceType: device.DeviceTypeStatic}
	mockRepo.devices[d.ID] = d

	svc := newService(mockRepo)
	updated, err := svc.UpdateDevice(ctx, d.ID, device.UpdateDeviceInput{Name: new("new-name")})

	is.NoErr(err)
	is.Equal(updated.Name, "new-name")
}

func TestService_UpdateDevice_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	svc := newService(mockRepo)

	_, err := svc.UpdateDevice(ctx, device.DeviceID(99), device.UpdateDeviceInput{})

	is.True(errors.Is(err, device.ErrDeviceNotFound))
}

func TestService_UpdateDevice_InvalidTypePropagated(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	d := &device.Device{ID: device.DeviceID(1), Name: "d", DeviceType: device.DeviceTypeStatic}
	mockRepo.devices[d.ID] = d

	svc := newService(mockRepo)
	_, err := svc.UpdateDevice(ctx, d.ID, device.UpdateDeviceInput{DeviceType: new("robot")})

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
	_, err := svc.UpdateDevice(ctx, d.ID, device.UpdateDeviceInput{})

	is.True(errors.Is(err, sentinel))
}
