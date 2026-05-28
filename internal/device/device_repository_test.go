//go:build test

package device_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

type testFixture struct {
	repo    *device.Repository
	db      *database.DB
	ownerID ids.UserID
}

func setupTestDB(t *testing.T) testFixture {
	t.Helper()

	sqlite, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	sqlxDB := sqlite.DB()
	var ownerID ids.UserID
	if err := sqlxDB.QueryRowxContext(t.Context(),
		`INSERT INTO users (username, display_name, password_hash, role) VALUES ('testadmin', 'Test Admin', 'x', 'admin') RETURNING id`,
	).Scan(&ownerID); err != nil {
		t.Fatalf("setupTestDB: insert test user: %v", err)
	}

	return testFixture{
		repo:    device.NewRepository(sqlxDB),
		db:      sqlxDB,
		ownerID: ownerID,
	}
}

func createTestDevice(t *testing.T, fix testFixture, ctx context.Context, name string) *device.Device {
	t.Helper()

	dev, err := fix.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: name, OwnerID: fix.ownerID})
	if err != nil {
		t.Fatalf("create device %q: %v", name, err)
	}
	return dev
}

func createTestAddress(t *testing.T, repo *device.Repository, ctx context.Context, deviceID ids.DeviceID, ip string) *device.Address {
	t.Helper()

	params, err := device.NewCreateAddressParams(deviceID, ip, netip.Addr{})
	if err != nil {
		t.Fatalf("create address params %q: %v", ip, err)
	}
	created, err := repo.CreateAddress(ctx, params, device.EventSourceManual)
	if err != nil {
		t.Fatalf("persist address %q: %v", ip, err)
	}
	return created
}

func TestRepository_CreateDevice(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "test-device", OwnerID: repos.ownerID})
	is.NoErr(err)
	is.Equal(dev.Name, "test-device")
	is.True(!dev.CreatedAt.IsZero())
}

func TestRepository_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	params := device.CreateDeviceParams{Name: "duplicate-name", OwnerID: repos.ownerID}
	_, err := repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)

	_, err = repos.repo.CreateDevice(ctx, params)
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDuplicateDeviceName))
}

func TestRepository_CreateDevice_SameNameAfterSoftDelete(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "reused-name")
	err := repos.repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	second, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "reused-name", OwnerID: repos.ownerID})
	is.NoErr(err)
	is.True(second.ID != dev.ID)
	is.Equal(second.Name, "reused-name")
}

func TestRepository_DeleteDevice_Success(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "to-delete")
	err := repos.repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	_, err = repos.repo.GetDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	err := repos.repo.DeleteDevice(ctx, ids.DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_AlreadyDeleted(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "deleted-once")
	err := repos.repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	err = repos.repo.DeleteDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDevice_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "hidden-after-delete")
	_, err := repos.repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)

	err = repos.repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	_, err = repos.repo.GetDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDeviceByAPIKeyHash_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "apikey-device", OwnerID: repos.ownerID})
	is.NoErr(err)

	_, keyHash, keyPrefix, err := device.GenerateAPIKey()
	is.NoErr(err)
	err = repos.repo.UpsertAPIKey(ctx, dev.ID, keyHash, keyPrefix)
	is.NoErr(err)

	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.NoErr(err)

	err = repos.repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDevice(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	created, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "test-device", OwnerID: repos.ownerID})
	is.NoErr(err)

	got, err := repos.repo.GetDevice(ctx, created.ID)
	is.NoErr(err)
	is.Equal(got.ID, created.ID)
	is.Equal(got.Name, "test-device")
	is.True(!got.CreatedAt.IsZero())
	is.True(got.KeyPrefix == nil)
}

func TestRepository_GetDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	_, err := repos.repo.GetDevice(ctx, ids.DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDeviceByAPIKeyHash_Success(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "lookup-device", OwnerID: repos.ownerID})
	is.NoErr(err)

	_, keyHash, keyPrefix, err := device.GenerateAPIKey()
	is.NoErr(err)
	err = repos.repo.UpsertAPIKey(ctx, dev.ID, keyHash, keyPrefix)
	is.NoErr(err)

	found, err := repos.repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.NoErr(err)
	is.True(found != nil)
	is.Equal(found.ID, dev.ID)
	is.Equal(found.Name, "lookup-device")
}

func TestRepository_GetDeviceByAPIKeyHash_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	_, err := repos.repo.GetDeviceByAPIKeyHash(ctx, "nonexistent-hash")
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_CreateDevice_NoAPIKeyRow(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "no-api-key", OwnerID: repos.ownerID})
	is.NoErr(err)

	fetched, err := repos.repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.True(fetched.KeyPrefix == nil)
}

func TestRepository_UpsertAPIKey_Success(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "regen-device", OwnerID: repos.ownerID})
	is.NoErr(err)

	_, oldHash, oldPrefix, err := device.GenerateAPIKey()
	is.NoErr(err)
	err = repos.repo.UpsertAPIKey(ctx, dev.ID, oldHash, oldPrefix)
	is.NoErr(err)

	found, err := repos.repo.GetDeviceByAPIKeyHash(ctx, oldHash)
	is.NoErr(err)
	is.Equal(found.ID, dev.ID)
	is.True(found.KeyPrefix != nil)
	is.Equal(*found.KeyPrefix, oldPrefix)

	_, newHash, newPrefix, err := device.GenerateAPIKey()
	is.NoErr(err)
	err = repos.repo.UpsertAPIKey(ctx, dev.ID, newHash, newPrefix)
	is.NoErr(err)

	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, oldHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)

	updated, err := repos.repo.GetDeviceByAPIKeyHash(ctx, newHash)
	is.NoErr(err)
	is.Equal(updated.ID, dev.ID)
	is.True(updated.KeyPrefix != nil)
	is.Equal(*updated.KeyPrefix, newPrefix)
}

func TestRepository_UpsertAPIKey_NonExistentDevice(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	_, keyHash, keyPrefix, err := device.GenerateAPIKey()
	is.NoErr(err)

	err = repos.repo.UpsertAPIKey(ctx, ids.DeviceID(99999), keyHash, keyPrefix)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_DeleteAPIKey_Success(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "delete-key-device", OwnerID: repos.ownerID})
	is.NoErr(err)

	_, keyHash, keyPrefix, err := device.GenerateAPIKey()
	is.NoErr(err)
	err = repos.repo.UpsertAPIKey(ctx, dev.ID, keyHash, keyPrefix)
	is.NoErr(err)

	err = repos.repo.DeleteAPIKey(ctx, dev.ID)
	is.NoErr(err)

	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)

	fetched, err := repos.repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.True(fetched.KeyPrefix == nil)
}

func TestRepository_DeleteAPIKey_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev, err := repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "no-key-device", OwnerID: repos.ownerID})
	is.NoErr(err)

	err = repos.repo.DeleteAPIKey(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrNoAPIKey)
}

func TestRepository_CreateDevice_DefaultsForNewFields(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "defaults-device")

	is.Equal(dev.DeviceType, device.DeviceTypeStatic)
	is.True(dev.Description == nil)
	is.True(dev.Icon == nil)
	is.True(!dev.UpdatedAt.IsZero())
}

func TestRepository_UpdateDevice_Rename(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "original")
	originalUpdatedAt := dev.UpdatedAt

	// Give time so updated_at can advance
	time.Sleep(2 * time.Millisecond)

	dev.Name = "renamed"
	updated, err := repos.repo.UpdateDevice(ctx, dev)

	is.NoErr(err)
	is.Equal(updated.Name, "renamed")
	// Note: SQLite CURRENT_TIMESTAMP has second-level granularity; a sub-second
	// sleep cannot verify advancement. The assertion checks only that updated_at
	// was not moved backward.
	is.True(!updated.UpdatedAt.Before(originalUpdatedAt))
}

func TestRepository_UpdateDevice_SetAllFields(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "full-update")
	dev.DeviceType = device.DeviceTypeMobile
	dev.Description = new("a note")
	dev.Icon = new("IconRouter")

	updated, err := repos.repo.UpdateDevice(ctx, dev)

	is.NoErr(err)
	is.Equal(updated.DeviceType, device.DeviceTypeMobile)
	is.True(updated.Description != nil)
	is.Equal(*updated.Description, "a note")
	is.True(updated.Icon != nil)
	is.Equal(*updated.Icon, "IconRouter")
}

func TestRepository_UpdateDevice_ClearDescription(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "clear-desc")
	dev.Description = new("initial")
	dev, err := repos.repo.UpdateDevice(ctx, dev)
	is.NoErr(err)
	is.True(dev.Description != nil)

	dev.Description = nil
	updated, err := repos.repo.UpdateDevice(ctx, dev)

	is.NoErr(err)
	is.True(updated.Description == nil)
}

func TestRepository_UpdateDevice_NotFound(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	ghost := &device.Device{ID: ids.DeviceID(9999), Name: "ghost", DeviceType: device.DeviceTypeStatic}
	_, err := repos.repo.UpdateDevice(ctx, ghost)

	is.True(errors.Is(err, device.ErrDeviceNotFound))
}

func TestRepository_UpdateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	createTestDevice(t, repos, ctx, "taken")
	dev := createTestDevice(t, repos, ctx, "to-rename")
	dev.Name = "taken"

	_, err := repos.repo.UpdateDevice(ctx, dev)

	is.True(errors.Is(err, device.ErrDuplicateDeviceName))
}

func TestRepository_GetDeviceIDsByOwner_ReturnsOwnedDevices(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repos, ctx, "device-a")
	dev2 := createTestDevice(t, repos, ctx, "device-b")

	deviceIDs, err := repos.repo.GetDeviceIDsByOwner(ctx, repos.ownerID)
	is.NoErr(err)
	is.Equal(len(deviceIDs), 2)

	got := map[ids.DeviceID]bool{deviceIDs[0]: true, deviceIDs[1]: true}
	is.True(got[dev1.ID])
	is.True(got[dev2.ID])
}

func TestRepository_GetDeviceIDsByOwner_ExcludesOtherOwners(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	var otherOwnerID ids.UserID
	err := repos.db.QueryRowxContext(ctx,
		`INSERT INTO users (username, display_name, password_hash, role) VALUES ('other', 'Other', 'x', 'user') RETURNING id`,
	).Scan(&otherOwnerID)
	is.NoErr(err)

	createTestDevice(t, repos, ctx, "main-device")
	_, err = repos.repo.CreateDevice(ctx, device.CreateDeviceParams{Name: "other-device", OwnerID: otherOwnerID})
	is.NoErr(err)

	deviceIDs, err := repos.repo.GetDeviceIDsByOwner(ctx, repos.ownerID)
	is.NoErr(err)
	is.Equal(len(deviceIDs), 1)
}

func TestRepository_GetDeviceIDsByOwner_ExcludesDeleted(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	active := createTestDevice(t, repos, ctx, "active")
	deleted := createTestDevice(t, repos, ctx, "to-delete")
	err := repos.repo.DeleteDevice(ctx, deleted.ID)
	is.NoErr(err)

	deviceIDs, err := repos.repo.GetDeviceIDsByOwner(ctx, repos.ownerID)
	is.NoErr(err)
	is.Equal(len(deviceIDs), 1)
	is.Equal(deviceIDs[0], active.ID)
}

func TestRepository_GetDeviceIDsByOwner_EmptyWhenNoDevices(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	deviceIDs, err := repos.repo.GetDeviceIDsByOwner(ctx, repos.ownerID)
	is.NoErr(err)
	is.Equal(len(deviceIDs), 0)
}
