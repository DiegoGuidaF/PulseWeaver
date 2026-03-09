//go:build test

package device

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/WallyDex/internal/testdb"
	"github.com/matryer/is"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()

	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	return NewRepository(db.DB())
}

func createTestDevice(t *testing.T, repo repository, ctx context.Context, name string) *Device {
	t.Helper()

	params, _, err := NewCreateDeviceParams(name)
	if err != nil {
		t.Fatalf("create device params %q: %v", name, err)
	}
	device, err := repo.CreateDevice(ctx, params)
	if err != nil {
		t.Fatalf("create device %q: %v", name, err)
	}
	return device
}

func createTestAddress(t *testing.T, repo repository, ctx context.Context, deviceID DeviceID, ip string) *Address {
	t.Helper()

	params, err := NewCreateAddressParams(deviceID, ip, netip.Addr{})
	if err != nil {
		t.Fatalf("create address params %q: %v", ip, err)
	}
	created, err := repo.CreateAddress(ctx, params)
	if err != nil {
		t.Fatalf("persist address %q: %v", ip, err)
	}
	return created
}

func TestRepository_CreateDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, _, err := NewCreateDeviceParams("test-device")
	is.NoErr(err)
	device, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)
	is.Equal(device.Name, "test-device")
	is.True(!device.CreatedAt.IsZero())
}

func TestRepository_GetDevices_Empty(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	devices, err := repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 0) // Should be empty
}

func TestRepository_GetDevices_Multiple(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create test data (CreateDevice now creates API key; devices appear in GetDevices JOIN)
	createTestDevice(t, repo, ctx, "device-1")
	createTestDevice(t, repo, ctx, "device-2")

	// Get all devices
	devices, err := repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 2) // Should have 2 devices
	is.Equal(devices[0].KeyPrefix != "", true)
	is.Equal(devices[1].KeyPrefix != "", true)
}

func TestRepository_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create first device
	params, _, err := NewCreateDeviceParams("duplicate-name")
	is.NoErr(err)
	_, err = repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Try to create device with same name (active unique index)
	_, err = repo.CreateDevice(ctx, params)
	is.True(err != nil)
	is.True(errors.Is(err, ErrDuplicateDeviceName))
}

func TestRepository_CreateDevice_SameNameAfterSoftDelete(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create device
	device := createTestDevice(t, repo, ctx, "reused-name")
	// Soft-delete it
	err := repo.DeleteDevice(ctx, device.ID)
	is.NoErr(err)

	// Same name is allowed again
	params, _, err := NewCreateDeviceParams("reused-name")
	is.NoErr(err)
	second, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)
	is.True(second.ID != device.ID)
	is.Equal(second.Name, "reused-name")
}

func TestRepository_DeleteDevice_Success(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	device := createTestDevice(t, repo, ctx, "to-delete")
	err := repo.DeleteDevice(ctx, device.ID)
	is.NoErr(err)

	// Deleted device is hidden from GetDevice
	_, err = repo.GetDevice(ctx, device.ID)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	err := repo.DeleteDevice(ctx, DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_AlreadyDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	device := createTestDevice(t, repo, ctx, "deleted-once")
	err := repo.DeleteDevice(ctx, device.ID)
	is.NoErr(err)

	// Second delete returns not found (idempotent 404)
	err = repo.DeleteDevice(ctx, device.ID)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_GetDevice_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	device := createTestDevice(t, repo, ctx, "hidden-after-delete")
	_, err := repo.GetDevice(ctx, device.ID)
	is.NoErr(err)

	err = repo.DeleteDevice(ctx, device.ID)
	is.NoErr(err)

	_, err = repo.GetDevice(ctx, device.ID)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_GetDevices_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	createTestDevice(t, repo, ctx, "device-1")
	device2 := createTestDevice(t, repo, ctx, "device-2")

	devices, err := repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 2)

	err = repo.DeleteDevice(ctx, device2.ID)
	is.NoErr(err)

	devices, err = repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].Name, "device-1")
}

func TestRepository_GetDeviceByAPIKeyHash_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, rawKey, err := NewCreateDeviceParams("apikey-device")
	is.NoErr(err)
	device, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	keyHash := hashAPIKey(rawKey)
	_, err = repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.NoErr(err)

	err = repo.DeleteDevice(ctx, device.ID)
	is.NoErr(err)

	_, err = repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_DatabaseIsolation(t *testing.T) {

	// Test 1: Create 1 device with API key
	t.Run("test1", func(t *testing.T) {
		is := is.New(t)

		repo := setupTestDB(t)
		ctx := context.Background()

		createTestDevice(t, repo, ctx, "device-1")

		devices, err := repo.GetDevices(ctx)
		is.NoErr(err)
		is.Equal(len(devices), 1) // Should have 1 device
	})

	// Test 2: Should have 0 devices (fresh DB)
	t.Run("test2", func(t *testing.T) {
		is := is.New(t)

		repo := setupTestDB(t)
		ctx := context.Background()

		devices, err := repo.GetDevices(ctx)
		is.NoErr(err)
		is.Equal(len(devices), 0) // Should be empty (isolated from test1)
	})
}

func TestRepository_GetDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	params, _, err := NewCreateDeviceParams("test-device")
	is.NoErr(err)
	createdDevice, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Get device by ID
	device, err := repo.GetDevice(ctx, createdDevice.ID)
	is.NoErr(err)
	is.Equal(device.ID, createdDevice.ID)
	is.Equal(device.Name, "test-device")
	is.True(!device.CreatedAt.IsZero())
}

func TestRepository_GetDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Try to get non-existent device
	_, err := repo.GetDevice(ctx, DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_GetDeviceByAPIKeyHash_Success(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create device via CreateDevice
	params, rawKey, err := NewCreateDeviceParams("lookup-device")
	is.NoErr(err)
	device, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	keyHash := hashAPIKey(rawKey)
	found, err := repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.NoErr(err)
	is.True(found != nil)
	is.Equal(found.ID, device.ID)
	is.Equal(found.Name, "lookup-device")
}

func TestRepository_GetDeviceByAPIKeyHash_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetDeviceByAPIKeyHash(ctx, "nonexistent-hash")
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_CreateDevice_InsertsAPIKeyRow(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create device using params so we have the key metadata and raw key
	params, rawKey, err := NewCreateDeviceParams("with-api-key")
	is.NoErr(err)
	device, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Verify there is exactly one API key row for this device with the expected fields
	type dbAPIKey struct {
		DeviceID  DeviceID `db:"device_id"`
		KeyPrefix string   `db:"key_prefix"`
		KeyHash   string   `db:"key_hash"`
	}

	var row dbAPIKey
	err = repo.rootDB.GetContext(ctx, &row, `
		SELECT device_id, key_prefix, key_hash
		FROM device_api_keys
		WHERE device_id = ?
	`, device.ID)
	is.NoErr(err)

	is.Equal(row.DeviceID, device.ID)
	is.Equal(row.KeyPrefix, params.KeyPrefix)
	is.Equal(row.KeyHash, params.KeyHash)

	// Sanity check: stored hash matches hashing the raw key returned from NewCreateDeviceParams
	is.Equal(row.KeyHash, hashAPIKey(rawKey))
}

func TestRepository_CreateAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device first
	device := createTestDevice(t, repo, ctx, "test-device")

	// Create an address
	address := createTestAddress(t, repo, ctx, device.ID, "192.168.1.100")
	is.Equal(address.DeviceID, device.ID)
	is.Equal(address.IP, "192.168.1.100")
	is.True(!address.CreatedAt.IsZero())
	is.True(address.ID != 0)
}

func TestRepository_FindAddressForDeviceByIp(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device := createTestDevice(t, repo, ctx, "test-device")
	createdAddr := createTestAddress(t, repo, ctx, device.ID, "192.168.1.100")

	// Find the address
	address, err := repo.GetAddressForDeviceByIP(ctx, device.ID, netip.MustParseAddr("192.168.1.100"))
	is.NoErr(err)
	is.Equal(address.ID, createdAddr.ID)
	is.Equal(address.DeviceID, device.ID)
	is.Equal(address.IP, "192.168.1.100")
}

func TestRepository_FindAddressForDeviceByIp_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device := createTestDevice(t, repo, ctx, "test-device")

	// Try to find non-existent address
	_, err := repo.GetAddressForDeviceByIP(ctx, device.ID, netip.MustParseAddr("192.168.1.99"))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_FindAddressForDeviceByIp_WrongDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create two devices
	device1 := createTestDevice(t, repo, ctx, "device-1")
	device2 := createTestDevice(t, repo, ctx, "device-2")

	// Create address for device1
	createTestAddress(t, repo, ctx, device1.ID, "192.168.1.100")

	// Try to find address for device2 (should not find it)
	_, err := repo.GetAddressForDeviceByIP(ctx, device2.ID, netip.MustParseAddr("192.168.1.100"))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_ListAddresses(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device := createTestDevice(t, repo, ctx, "test-device")

	// Create multiple addresses
	createTestAddress(t, repo, ctx, device.ID, "192.168.1.1")
	createTestAddress(t, repo, ctx, device.ID, "192.168.1.2")

	// List addresses
	addresses, err := repo.ListAddresses(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 2)

	// Verify addresses are returned
	ips := make(map[string]bool)
	for _, addr := range addresses {
		ips[addr.IP] = true
	}
	is.True(ips["192.168.1.1"])
	is.True(ips["192.168.1.2"])
}

func TestRepository_ListAddresses_Empty(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device := createTestDevice(t, repo, ctx, "test-device")

	// List addresses (should be empty)
	addresses, err := repo.ListAddresses(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 0)
}

func TestRepository_DisableAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, device.ID, "192.168.1.100")

	// Disable address
	disabledAddr, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(disabledAddr.ID, addr.ID)
	is.True(!disabledAddr.IsEnabled) // Should be disabled
}

func TestRepository_EnableAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, device.ID, "192.168.1.100")

	// Disable address
	_, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Enable address
	enabledAddr, err := repo.EnableAddress(ctx, addr.ID, EventSourceManual)
	is.NoErr(err)
	is.Equal(enabledAddr.ID, addr.ID)
	is.True(enabledAddr.IsEnabled) // Should be enabled
}

func TestRepository_GetAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, device.ID, "192.168.1.100")

	// Get address with status
	addrWithStatus, err := repo.GetAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(addrWithStatus.ID, addr.ID)
	is.Equal(addrWithStatus.DeviceID, device.ID)
	is.Equal(addrWithStatus.IP, "192.168.1.100")
	is.True(addrWithStatus.IsEnabled) // Should be enabled
	is.True(!addrWithStatus.CreatedAt.IsZero())
}

func TestRepository_GetAddress_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Try to get non-existent address
	_, err := repo.GetAddress(ctx, AddressID(99999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_CheckAddressOwnership(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, device.ID, "192.168.1.100")

	// Check ownership (should succeed)
	err := repo.CheckAddressOwnership(ctx, device.ID, addr.ID)
	is.NoErr(err)
}

func TestRepository_CheckAddressOwnership_WrongDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create two devices
	device1 := createTestDevice(t, repo, ctx, "device-1")
	device2 := createTestDevice(t, repo, ctx, "device-2")

	// Create address for device1
	addr := createTestAddress(t, repo, ctx, device1.ID, "192.168.1.100")

	// Check ownership with device2 (should fail)
	err := repo.CheckAddressOwnership(ctx, device2.ID, addr.ID)
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
}

func TestRepository_CheckAddressOwnership_AddressNotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device := createTestDevice(t, repo, ctx, "test-device")

	// Check ownership of non-existent address (should fail)
	err := repo.CheckAddressOwnership(ctx, device.ID, AddressID(99999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
}

func TestRepository_RunInTx(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device := createTestDevice(t, repo, ctx, "test-device")

	// Run operations in transaction
	err := repo.RunInTx(ctx, func(tx repository) error {
		// Create address in transaction (CreateAddress already records status as enabled)
		_ = createTestAddress(t, tx, ctx, device.ID, "192.168.1.100")
		return nil
	})
	is.NoErr(err)

	// Verify address was created and enabled
	addresses, err := repo.ListAddresses(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 1)
	is.Equal(addresses[0].IP, "192.168.1.100")
	is.True(addresses[0].IsEnabled) // Should be enabled
}

func TestRepository_RunInTx_Rollback(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device := createTestDevice(t, repo, ctx, "test-device")

	// Run operations in transaction that will fail
	testError := fmt.Errorf("test error")
	err := repo.RunInTx(ctx, func(tx repository) error {
		// Create address in transaction
		createTestAddress(t, tx, ctx, device.ID, "192.168.1.100")

		// Return error to trigger rollback
		return testError
	})
	is.True(err != nil) // Transaction should fail
	is.Equal(err, testError)

	// Verify address was NOT created (transaction rolled back)
	addresses, err := repo.ListAddresses(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 0) // Should be empty due to rollback
}

func TestRepository_GetEnabledUniqueIPs_Empty(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Get enabled IPs when none exist
	ips, err := repo.GetEnabledUniqueIPs(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 0) // Should be empty
}

func TestRepository_GetEnabledUniqueIPs(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create device
	device := createTestDevice(t, repo, ctx, "test-device")

	// Create addresses
	_ = createTestAddress(t, repo, ctx, device.ID, "192.168.1.1")
	addrToDisable := createTestAddress(t, repo, ctx, device.ID, "192.168.1.2")
	_ = createTestAddress(t, repo, ctx, device.ID, "192.168.1.3")

	// Disable addr2
	_, err := repo.DisableAddress(ctx, addrToDisable.ID)
	is.NoErr(err)

	// Get enabled IPs
	ips, err := repo.GetEnabledUniqueIPs(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 2) // Should have 2 enabled IPs

	// Verify correct IPs are returned
	ipMap := make(map[string]bool)
	for _, ip := range ips {
		ipMap[ip] = true
	}
	is.True(ipMap["192.168.1.1"])
	is.True(ipMap["192.168.1.3"])
	is.True(!ipMap["192.168.1.2"]) // Disabled address should not be included
}

func TestRepository_GetEnabledUniqueIPs_Deduplicates(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create multiple devices
	device1 := createTestDevice(t, repo, ctx, "device-1")
	device2 := createTestDevice(t, repo, ctx, "device-2")
	device3 := createTestDevice(t, repo, ctx, "device-3")

	// Create addresses with duplicate IPs across different devices
	_ = createTestAddress(t, repo, ctx, device1.ID, "192.168.1.100")
	_ = createTestAddress(t, repo, ctx, device2.ID, "192.168.1.100") // Same IP as addr1
	_ = createTestAddress(t, repo, ctx, device3.ID, "192.168.1.200")

	// Get enabled IPs
	ips, err := repo.GetEnabledUniqueIPs(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 2) // Should deduplicate: 192.168.1.100 appears only once

	// Verify correct IPs are returned (deduplicated)
	ipMap := make(map[string]bool)
	for _, ip := range ips {
		ipMap[ip] = true
	}
	is.True(ipMap["192.168.1.100"])
	is.True(ipMap["192.168.1.200"])
}
