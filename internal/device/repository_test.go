package device

import (
	"context"
	"fmt"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"github.com/matryer/is"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()

	conf := config.ConfDB{
		Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
		Debug: false,
	}

	db, err := database.NewSQLite(conf)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewRepository(db.DB())
}

func TestRepository_CreateDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	device, err := repo.CreateDevice(ctx, "test-device")
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

	// Create test data
	_, err := repo.CreateDevice(ctx, "device-1")
	is.NoErr(err)
	_, err = repo.CreateDevice(ctx, "device-2")
	is.NoErr(err)

	// Get all devices
	devices, err := repo.GetDevices(ctx)
	is.NoErr(err)
	is.Equal(len(devices), 2) // Should have 2 devices
}

func TestRepository_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create first device
	_, err := repo.CreateDevice(ctx, "duplicate-name")
	is.NoErr(err)

	// Try to create device with same name
	_, err = repo.CreateDevice(ctx, "duplicate-name")
	is.True(err != nil) // Should error (UNIQUE constraint)
}

func TestRepository_DatabaseIsolation(t *testing.T) {

	// Test 1: Create 1 device
	t.Run("test1", func(t *testing.T) {
		is := is.New(t)

		repo := setupTestDB(t)
		ctx := context.Background()

		repo.CreateDevice(ctx, "device-1")

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

func TestRepository_GetDeviceByID(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	createdDevice, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Get device by ID
	device, err := repo.GetDeviceByID(ctx, createdDevice.ID)
	is.NoErr(err)
	is.Equal(device.ID, createdDevice.ID)
	is.Equal(device.Name, "test-device")
	is.True(!device.CreatedAt.IsZero())
}

func TestRepository_GetDeviceByID_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Try to get non-existent device
	_, err := repo.GetDeviceByID(ctx, DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, ErrDeviceNotFound)
}

func TestRepository_CreateAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device first
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Create an address
	address, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)
	is.Equal(address.DeviceId, device.ID)
	is.Equal(address.IP, "192.168.1.100")
	is.True(!address.CreatedAt.IsZero())
	is.True(address.ID != 0)
}

func TestRepository_CreateAddress_IPv6(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device first
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Create an IPv6 address
	address, err := repo.CreateAddress(ctx, device.ID, "2001:db8::1")
	is.NoErr(err)
	is.Equal(address.DeviceId, device.ID)
	is.Equal(address.IP, "2001:db8::1")
	is.True(!address.CreatedAt.IsZero())
}

func TestRepository_FindAddressForDeviceByIp(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	createdAddr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Find the address
	address, err := repo.GetAddressForDeviceByIp(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)
	is.Equal(address.Id, createdAddr.ID)
	is.Equal(address.DeviceId, device.ID)
	is.Equal(address.IP, "192.168.1.100")
}

func TestRepository_FindAddressForDeviceByIp_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Try to find non-existent address
	_, err = repo.GetAddressForDeviceByIp(ctx, device.ID, "192.168.1.999")
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_FindAddressForDeviceByIp_WrongDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create two devices
	device1, err := repo.CreateDevice(ctx, "device-1")
	is.NoErr(err)
	device2, err := repo.CreateDevice(ctx, "device-2")
	is.NoErr(err)

	// Create address for device1
	_, err = repo.CreateAddress(ctx, device1.ID, "192.168.1.100")
	is.NoErr(err)

	// Try to find address for device2 (should not find it)
	_, err = repo.GetAddressForDeviceByIp(ctx, device2.ID, "192.168.1.100")
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_ListAddresses(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Create multiple addresses
	addr1, err := repo.CreateAddress(ctx, device.ID, "192.168.1.1")
	is.NoErr(err)
	addr2, err := repo.CreateAddress(ctx, device.ID, "192.168.1.2")
	is.NoErr(err)

	// Enable addresses (they need status records)
	_, err = repo.EnableAddress(ctx, addr1.ID)
	is.NoErr(err)
	_, err = repo.EnableAddress(ctx, addr2.ID)
	is.NoErr(err)

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
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

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
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	addr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Enable address first
	_, err = repo.EnableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Disable address
	disabledAddr, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(disabledAddr.Id, addr.ID)
	is.True(!disabledAddr.Status) // Should be disabled
}

func TestRepository_EnableAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	addr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Enable address
	enabledAddr, err := repo.EnableAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(enabledAddr.Id, addr.ID)
	is.True(enabledAddr.Status) // Should be enabled
}

func TestRepository_EnableAddress_ReEnable(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	addr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Enable address
	_, err = repo.EnableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Disable address
	_, err = repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Re-enable address
	enabledAddr, err := repo.EnableAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(enabledAddr.Id, addr.ID)
	is.True(enabledAddr.Status) // Should be enabled again
}

func TestRepository_GetAddressWithStatus(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	addr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Enable address
	_, err = repo.EnableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Get address with status
	addrWithStatus, err := repo.GetAddressWithStatus(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(addrWithStatus.Id, addr.ID)
	is.Equal(addrWithStatus.DeviceId, device.ID)
	is.Equal(addrWithStatus.IP, "192.168.1.100")
	is.True(addrWithStatus.Status) // Should be enabled
	is.True(!addrWithStatus.CreatedAt.IsZero())
}

func TestRepository_GetAddressWithStatus_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Try to get non-existent address
	_, err := repo.GetAddressWithStatus(ctx, AddressID(99999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_GetAddressByID(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	createdAddr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Get address by ID
	address, err := repo.GetAddressByID(ctx, createdAddr.ID)
	is.NoErr(err)
	is.Equal(address.ID, createdAddr.ID)
	is.Equal(address.DeviceId, device.ID)
	is.Equal(address.IP, "192.168.1.100")
	is.True(!address.CreatedAt.IsZero())
}

func TestRepository_GetAddressByID_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Try to get non-existent address
	_, err := repo.GetAddressByID(ctx, AddressID(99999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotFound)
}

func TestRepository_CheckAddressOwnership(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device and address
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)
	addr, err := repo.CreateAddress(ctx, device.ID, "192.168.1.100")
	is.NoErr(err)

	// Check ownership (should succeed)
	err = repo.CheckAddressOwnership(ctx, device.ID, addr.ID)
	is.NoErr(err)
}

func TestRepository_CheckAddressOwnership_WrongDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create two devices
	device1, err := repo.CreateDevice(ctx, "device-1")
	is.NoErr(err)
	device2, err := repo.CreateDevice(ctx, "device-2")
	is.NoErr(err)

	// Create address for device1
	addr, err := repo.CreateAddress(ctx, device1.ID, "192.168.1.100")
	is.NoErr(err)

	// Check ownership with device2 (should fail)
	err = repo.CheckAddressOwnership(ctx, device2.ID, addr.ID)
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
}

func TestRepository_CheckAddressOwnership_AddressNotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Check ownership of non-existent address (should fail)
	err = repo.CheckAddressOwnership(ctx, device.ID, AddressID(99999))
	is.True(err != nil)
	is.Equal(err, ErrAddressNotOwnedByDevice)
}

func TestRepository_RunInTx(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Run operations in transaction
	err = repo.RunInTx(ctx, func(tx DeviceRepository) error {
		// Create address in transaction
		addr, err := tx.CreateAddress(ctx, device.ID, "192.168.1.100")
		if err != nil {
			return err
		}

		// Enable address in transaction
		_, err = tx.EnableAddress(ctx, addr.ID)
		return err
	})
	is.NoErr(err)

	// Verify address was created and enabled
	addresses, err := repo.ListAddresses(ctx, device.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 1)
	is.Equal(addresses[0].IP, "192.168.1.100")
	is.True(addresses[0].Status) // Should be enabled
}

func TestRepository_RunInTx_Rollback(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	// Create a device
	device, err := repo.CreateDevice(ctx, "test-device")
	is.NoErr(err)

	// Run operations in transaction that will fail
	testError := fmt.Errorf("test error")
	err = repo.RunInTx(ctx, func(tx DeviceRepository) error {
		// Create address in transaction
		_, err := tx.CreateAddress(ctx, device.ID, "192.168.1.100")
		if err != nil {
			return err
		}

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
