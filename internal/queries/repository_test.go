//go:build test

package queries_test

import (
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/lease"
	"github.com/DiegoGuidaF/WallyDex/internal/queries"
	"github.com/DiegoGuidaF/WallyDex/internal/testdb"
	"github.com/jmoiron/sqlx"
	"github.com/matryer/is"
)

// testRepos groups all repositories used by the queries package tests.
type testRepos struct {
	queries *queries.Repository
	devices *device.Repository
	leases  *lease.Repository
	db      *sqlx.DB
}

// setupRepos creates an in-memory SQLite DB and returns all repositories sharing it.
func setupRepos(t *testing.T) testRepos {
	t.Helper()

	dbWrapper, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	sqlxDB := dbWrapper.DB()
	return testRepos{
		queries: queries.NewRepository(sqlxDB),
		devices: device.NewRepository(sqlxDB),
		leases:  lease.NewRepository(sqlxDB),
		db:      sqlxDB,
	}
}

// createDevice is a test helper that inserts a device using the device repository.
func createDevice(t *testing.T, repo *device.Repository, name string) *device.Device {
	t.Helper()

	params, _, err := device.NewCreateDeviceParams(name)
	if err != nil {
		t.Fatalf("NewCreateDeviceParams(%q): %v", name, err)
	}
	dev, err := repo.CreateDevice(t.Context(), params)
	if err != nil {
		t.Fatalf("CreateDevice(%q): %v", name, err)
	}
	return dev
}

// createAddress is a test helper that inserts an address for a device using the device repository.
func createAddress(t *testing.T, repo *device.Repository, deviceID device.DeviceID, ip string) *device.Address {
	t.Helper()

	params, err := device.NewCreateAddressParams(deviceID, ip, netip.Addr{})
	if err != nil {
		t.Fatalf("NewCreateAddressParams(%q): %v", ip, err)
	}
	addr, err := repo.CreateAddress(t.Context(), params)
	if err != nil {
		t.Fatalf("CreateAddress(%q): %v", ip, err)
	}
	return addr
}

func TestRepository_DeviceExists_ExistingDevice(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "existing-device")

	exists, err := repos.queries.DeviceExists(t.Context(), dev.ID)
	is.NoErr(err)
	is.True(exists)
}

func TestRepository_DeviceExists_NonExistentDevice(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	exists, err := repos.queries.DeviceExists(t.Context(), device.DeviceID(99999))
	is.NoErr(err)
	is.True(!exists)
}

func TestRepository_DeviceExists_SoftDeletedDevice(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "to-delete")
	err := repos.devices.DeleteDevice(t.Context(), dev.ID)
	is.NoErr(err)

	exists, err := repos.queries.DeviceExists(t.Context(), dev.ID)
	is.NoErr(err)
	is.True(!exists)
}

func TestRepository_GetDeviceAddresses_EmptyForNoAddresses(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "no-addresses")

	addresses, err := repos.queries.GetDeviceAddresses(t.Context(), dev.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 0)
}

func TestRepository_GetDeviceAddresses_CorrectFields(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "field-check-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	addresses, err := repos.queries.GetDeviceAddresses(t.Context(), dev.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 1)

	got := addresses[0]
	is.Equal(got.ID, addr.ID)
	is.Equal(got.DeviceID, dev.ID)
	is.Equal(got.IP, "10.0.0.1")
	is.True(got.IsEnabled)
	is.Equal(got.Source, string(device.EventSourceManual))
	is.True(!got.CreatedAt.IsZero())
	is.True(!got.UpdatedAt.IsZero())
}

func TestRepository_GetDeviceAddresses_ExpiresAtPopulatedWithActiveLease(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "lease-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.1.2.3")

	futureExpiry := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	addressLease := &lease.AddressLease{
		AddressID: addr.ID,
		DeviceID:  dev.ID,
		ExpiresAt: &futureExpiry,
	}
	_, err := repos.leases.UpsertAddressLease(t.Context(), addressLease)
	is.NoErr(err)

	addresses, err := repos.queries.GetDeviceAddresses(t.Context(), dev.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 1)
	is.True(addresses[0].ExpiresAt != nil)
	// Compare truncated to second to avoid sub-second precision differences from SQLite.
	is.True(addresses[0].ExpiresAt.UTC().Truncate(time.Second).Equal(futureExpiry))
}

func TestRepository_GetDeviceAddresses_ExpiresAtNilWhenNoLease(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "no-lease-device")
	createAddress(t, repos.devices, dev.ID, "192.168.100.1")

	addresses, err := repos.queries.GetDeviceAddresses(t.Context(), dev.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 1)
	is.True(addresses[0].ExpiresAt == nil)
}

func TestRepository_GetDeviceAddresses_OrderedByCreatedAtDesc(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "ordering-device")

	// Insert the addresses row directly with explicit created_at values so the
	// ORDER BY created_at DESC behaviour is testable even when all three inserts
	// happen within the same wall-clock second (SQLite CURRENT_TIMESTAMP has
	// second precision). We then add the required address_current_state row so
	// that the JOIN inside GetDeviceAddresses succeeds.
	oldest := time.Now().UTC().Add(-2 * time.Second).Truncate(time.Second)
	middle := time.Now().UTC().Add(-1 * time.Second).Truncate(time.Second)
	newest := time.Now().UTC().Truncate(time.Second)

	insertAddr := func(ip string, createdAt time.Time) device.AddressID {
		t.Helper()
		var id device.AddressID
		err := repos.db.GetContext(t.Context(), &id,
			`INSERT INTO addresses (device_id, ip, created_at) VALUES (?, ?, ?) RETURNING id`,
			dev.ID, ip, createdAt,
		)
		if err != nil {
			t.Fatalf("insert address %q: %v", ip, err)
		}
		_, err = repos.db.ExecContext(t.Context(),
			`INSERT INTO address_current_state (address_id, is_enabled, source, updated_at) VALUES (?, 1, 'manual', ?)`,
			id, createdAt,
		)
		if err != nil {
			t.Fatalf("insert address_current_state for %q: %v", ip, err)
		}
		return id
	}

	idOldest := insertAddr("172.16.0.1", oldest)
	idMiddle := insertAddr("172.16.0.2", middle)
	idNewest := insertAddr("172.16.0.3", newest)

	addresses, err := repos.queries.GetDeviceAddresses(t.Context(), dev.ID)
	is.NoErr(err)
	is.Equal(len(addresses), 3)

	// Newest created_at should appear first in DESC order.
	is.Equal(addresses[0].ID, idNewest)
	is.Equal(addresses[1].ID, idMiddle)
	is.Equal(addresses[2].ID, idOldest)
}

func TestRepository_GetDevices_EmptySlice(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	devices, err := repos.queries.GetDevices(t.Context())
	is.NoErr(err)
	is.Equal(len(devices), 0)
}

func TestRepository_GetDevices_ReturnsFields(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "device-fields")

	devices, err := repos.queries.GetDevices(t.Context())
	is.NoErr(err)
	is.Equal(len(devices), 1)

	got := devices[0]
	is.Equal(got.ID, dev.ID)
	is.Equal(got.Name, dev.Name)
	is.Equal(got.KeyPrefix, dev.KeyPrefix)
	is.Equal(got.AddressCount, 0)
	is.True(!got.CreatedAt.IsZero())
}

func TestRepository_GetDevices_AddressCountZeroWhenNoAddresses(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	createDevice(t, repos.devices, "device-no-addresses")

	devices, err := repos.queries.GetDevices(t.Context())
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].AddressCount, 0)
}

func TestRepository_GetDevices_AddressCountOnlyCountsEnabledAddresses(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos.devices, "device-count")
	addr1 := createAddress(t, repos.devices, dev.ID, "10.0.0.10")
	createAddress(t, repos.devices, dev.ID, "10.0.0.11")

	_, err := repos.devices.DisableAddress(t.Context(), addr1.ID)
	is.NoErr(err)

	devices, err := repos.queries.GetDevices(t.Context())
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].AddressCount, 1)
}

func TestRepository_GetDevices_OrderedByCreatedAtDesc(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	olderTime := time.Now().UTC().Add(-2 * time.Second).Truncate(time.Second)
	newerTime := time.Now().UTC().Add(-1 * time.Second).Truncate(time.Second)

	insertDevice := func(name, prefix, hash string, createdAt time.Time) device.DeviceID {
		t.Helper()

		var id device.DeviceID
		err := repos.db.GetContext(t.Context(), &id,
			`INSERT INTO devices (name, created_at) VALUES (?, ?) RETURNING id`,
			name, createdAt,
		)
		if err != nil {
			t.Fatalf("insert device %q: %v", name, err)
		}

		_, err = repos.db.ExecContext(t.Context(),
			`INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at) VALUES (?, ?, ?, ?)`,
			id, prefix, hash, createdAt,
		)
		if err != nil {
			t.Fatalf("insert api key for %q: %v", name, err)
		}

		return id
	}

	oldID := insertDevice("older-device", "wdk_oldaaaa", "hash-old", olderTime)
	newID := insertDevice("newer-device", "wdk_newbbbb", "hash-new", newerTime)

	devices, err := repos.queries.GetDevices(t.Context())
	is.NoErr(err)
	is.Equal(len(devices), 2)
	is.Equal(devices[0].ID, newID)
	is.Equal(devices[1].ID, oldID)
}
