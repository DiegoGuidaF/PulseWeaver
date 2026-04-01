//go:build test

package queries_test

import (
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/jmoiron/sqlx"
	"github.com/matryer/is"
)

// testRepos groups all repositories used by the queries package tests.
type testRepos struct {
	queries   *queries.Repository
	devices   *device.Repository
	leases    *lease.Repository
	accessLog *accesslog.Repository
	db        *sqlx.DB
}

// setupRepos creates an in-memory SQLite DB and returns all repositories sharing it.
func setupRepos(t *testing.T) testRepos {
	t.Helper()

	dbWrapper, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	sqlxDB := dbWrapper.DB()
	return testRepos{
		queries:   queries.NewRepository(sqlxDB),
		devices:   device.NewRepository(sqlxDB),
		leases:    lease.NewRepository(sqlxDB),
		accessLog: accesslog.NewRepository(sqlxDB),
		db:        sqlxDB,
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
	addr, err := repo.CreateAddress(t.Context(), params, device.EventSourceManual)
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
