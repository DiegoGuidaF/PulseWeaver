//go:build test

package lease_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/jmoiron/sqlx"
	"github.com/matryer/is"
)

func setupLeaseTestDB(t *testing.T) (*lease.Repository, *sqlx.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return lease.NewRepository(db.DB()), db.DB()
}

func insertDevice(t *testing.T, db *sqlx.DB, name string) *device.Device {
	t.Helper()
	params, _, err := device.NewCreateDeviceParams(name)
	if err != nil {
		t.Fatalf("NewCreateDeviceParams: %v", err)
	}
	dev, err := device.NewRepository(db).CreateDevice(context.Background(), params)
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	return dev
}

func insertAddress(t *testing.T, db *sqlx.DB, deviceID device.DeviceID, ip string) *device.Address {
	t.Helper()
	params, err := device.NewCreateAddressParams(deviceID, ip, netip.Addr{})
	if err != nil {
		t.Fatalf("NewCreateAddressParams: %v", err)
	}
	addr, err := device.NewRepository(db).CreateAddress(context.Background(), params, device.EventSourceManual)
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	return addr
}

// UpsertAddressLease

func TestRepository_UpsertAddressLease_CreatesNewLease(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-create")
	addr := insertAddress(t, db, dev.ID, "10.0.0.1")

	expiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	input := &lease.AddressLease{
		AddressID: addr.ID,
		DeviceID:  dev.ID,
		ExpiresAt: &expiresAt,
	}

	got, err := repo.UpsertAddressLease(context.Background(), input)

	is.NoErr(err)
	is.True(got != nil)
	is.Equal(got.AddressID, addr.ID)
	is.Equal(got.DeviceID, dev.ID)
	is.True(got.ExpiresAt != nil)
	is.True(got.ExpiresAt.UTC().Truncate(time.Second).Equal(expiresAt))
}

func TestRepository_UpsertAddressLease_UpdatesExistingLease(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-update")
	addr := insertAddress(t, db, dev.ID, "10.0.0.2")

	first := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{
		AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: &first,
	})
	is.NoErr(err)

	second := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	got, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{
		AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: &second,
	})

	is.NoErr(err)
	is.True(got.ExpiresAt.UTC().Truncate(time.Second).Equal(second))
}

func TestRepository_UpsertAddressLease_NilExpiresAt_Allowed(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-nil-expiry")
	addr := insertAddress(t, db, dev.ID, "10.0.0.3")

	got, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{
		AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: nil,
	})

	is.NoErr(err)
	is.True(got != nil)
	is.True(got.ExpiresAt == nil)
}

// GetExpiredAddressIDs

func TestRepository_GetExpiredAddressIDs_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupLeaseTestDB(t)

	ids, err := repo.GetExpiredAddressIDs(context.Background())

	is.NoErr(err)
	is.Equal(len(ids), 0)
}

func TestRepository_GetExpiredAddressIDs_PastExpiry_ReturnsID(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-expired")
	addr := insertAddress(t, db, dev.ID, "10.0.1.1")

	past := time.Now().UTC().Add(-time.Minute)
	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{
		AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: &past,
	})
	is.NoErr(err)

	ids, err := repo.GetExpiredAddressIDs(context.Background())

	is.NoErr(err)
	is.Equal(len(ids), 1)
	is.Equal(ids[0], addr.ID)
}

func TestRepository_GetExpiredAddressIDs_FutureExpiry_NotReturned(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-future")
	addr := insertAddress(t, db, dev.ID, "10.0.2.1")

	future := time.Now().UTC().Add(time.Hour)
	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{
		AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: &future,
	})
	is.NoErr(err)

	ids, err := repo.GetExpiredAddressIDs(context.Background())

	is.NoErr(err)
	is.Equal(len(ids), 0)
}

func TestRepository_GetExpiredAddressIDs_NilExpiry_NotReturned(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-no-expiry")
	addr := insertAddress(t, db, dev.ID, "10.0.3.1")

	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{
		AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: nil,
	})
	is.NoErr(err)

	ids, err := repo.GetExpiredAddressIDs(context.Background())

	is.NoErr(err)
	is.Equal(len(ids), 0)
}

// SetDeviceAddressLeasesExpiry

func TestRepository_SetDeviceAddressLeasesExpiry_UpdatesAllLeasesForDevice(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-set-expiry")
	addr1 := insertAddress(t, db, dev.ID, "10.1.0.1")
	addr2 := insertAddress(t, db, dev.ID, "10.1.0.2")

	past := time.Now().UTC().Add(-time.Minute)
	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{AddressID: addr1.ID, DeviceID: dev.ID, ExpiresAt: &past})
	is.NoErr(err)
	_, err = repo.UpsertAddressLease(context.Background(), &lease.AddressLease{AddressID: addr2.ID, DeviceID: dev.ID, ExpiresAt: &past})
	is.NoErr(err)

	// Both should be expired before the update.
	expired, err := repo.GetExpiredAddressIDs(context.Background())
	is.NoErr(err)
	is.Equal(len(expired), 2)

	future := time.Now().UTC().Add(time.Hour)
	err = repo.SetDeviceAddressLeasesExpiry(context.Background(), dev.ID, &future, time.Now().UTC())
	is.NoErr(err)

	// After the update, none should be expired.
	expired, err = repo.GetExpiredAddressIDs(context.Background())
	is.NoErr(err)
	is.Equal(len(expired), 0)
}

func TestRepository_SetDeviceAddressLeasesExpiry_NilExpiry_ClearsExpiry(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev := insertDevice(t, db, "lease-device-clear-expiry")
	addr := insertAddress(t, db, dev.ID, "10.2.0.1")

	past := time.Now().UTC().Add(-time.Minute)
	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{AddressID: addr.ID, DeviceID: dev.ID, ExpiresAt: &past})
	is.NoErr(err)

	err = repo.SetDeviceAddressLeasesExpiry(context.Background(), dev.ID, nil, time.Now().UTC())
	is.NoErr(err)

	// Nil expiry means no expiration — should not appear in expired list.
	expired, err := repo.GetExpiredAddressIDs(context.Background())
	is.NoErr(err)
	is.Equal(len(expired), 0)
}

func TestRepository_SetDeviceAddressLeasesExpiry_OnlyAffectsTargetDevice(t *testing.T) {
	is := is.New(t)
	repo, db := setupLeaseTestDB(t)
	dev1 := insertDevice(t, db, "lease-device-isolation-1")
	dev2 := insertDevice(t, db, "lease-device-isolation-2")
	addr1 := insertAddress(t, db, dev1.ID, "10.3.0.1")
	addr2 := insertAddress(t, db, dev2.ID, "10.3.0.2")

	past := time.Now().UTC().Add(-time.Minute)
	_, err := repo.UpsertAddressLease(context.Background(), &lease.AddressLease{AddressID: addr1.ID, DeviceID: dev1.ID, ExpiresAt: &past})
	is.NoErr(err)
	_, err = repo.UpsertAddressLease(context.Background(), &lease.AddressLease{AddressID: addr2.ID, DeviceID: dev2.ID, ExpiresAt: &past})
	is.NoErr(err)

	// Update only dev1's leases to a future time.
	future := time.Now().UTC().Add(time.Hour)
	err = repo.SetDeviceAddressLeasesExpiry(context.Background(), dev1.ID, &future, time.Now().UTC())
	is.NoErr(err)

	// dev2's lease should still be expired.
	expired, err := repo.GetExpiredAddressIDs(context.Background())
	is.NoErr(err)
	is.Equal(len(expired), 1)
	is.Equal(expired[0], addr2.ID)
}
