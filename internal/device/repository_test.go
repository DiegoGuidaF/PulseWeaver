//go:build test

package device_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/matryer/is"
)

type testFixture struct {
	repo    *device.Repository
	ownerID auth.UserID
}

func setupTestDB(t *testing.T) testFixture {
	t.Helper()

	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	sqlxDB := db.DB()
	var ownerID auth.UserID
	if err := sqlxDB.QueryRowx(
		`INSERT INTO users (username, display_name, password_hash, role) VALUES ('testadmin', 'Test Admin', 'x', 'admin') RETURNING id`,
	).Scan(&ownerID); err != nil {
		t.Fatalf("setupTestDB: insert test user: %v", err)
	}

	return testFixture{
		repo:    device.NewRepository(sqlxDB),
		ownerID: ownerID,
	}
}

func createTestDevice(t *testing.T, fix testFixture, ctx context.Context, name string) *device.Device {
	t.Helper()

	params, _, err := device.NewCreateDeviceParams(name, fix.ownerID)
	if err != nil {
		t.Fatalf("create device params %q: %v", name, err)
	}
	dev, err := fix.repo.CreateDevice(ctx, params)
	if err != nil {
		t.Fatalf("create device %q: %v", name, err)
	}
	return dev
}

func createTestAddress(t *testing.T, repo *device.Repository, ctx context.Context, deviceID device.DeviceID, ip string) *device.Address {
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

	params, _, err := device.NewCreateDeviceParams("test-device", repos.ownerID)
	is.NoErr(err)
	dev, err := repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)
	is.Equal(dev.Name, "test-device")
	is.True(!dev.CreatedAt.IsZero())
}

func TestRepository_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("duplicate-name", repos.ownerID)
	is.NoErr(err)
	_, err = repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Try to create device with same name (active unique index)
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

	// Same name is allowed again
	params, _, err := device.NewCreateDeviceParams("reused-name", repos.ownerID)
	is.NoErr(err)
	second, err := repos.repo.CreateDevice(ctx, params)
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

	// Deleted device is hidden from GetDevice
	_, err = repos.repo.GetDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	err := repos.repo.DeleteDevice(ctx, device.DeviceID(99999))
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

	// Second delete returns not found (idempotent 404)
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

	params, _, err := device.NewCreateDeviceParams("apikey-device", repos.ownerID)
	is.NoErr(err)
	dev, err := repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)

	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.NoErr(err)

	err = repos.repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDevice(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("test-device", repos.ownerID)
	is.NoErr(err)
	created, err := repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)

	got, err := repos.repo.GetDevice(ctx, created.ID)
	is.NoErr(err)
	is.Equal(got.ID, created.ID)
	is.Equal(got.Name, "test-device")
	is.True(!got.CreatedAt.IsZero())
}

func TestRepository_GetDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	_, err := repos.repo.GetDevice(ctx, device.DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDeviceByAPIKeyHash_Success(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("lookup-device", repos.ownerID)
	is.NoErr(err)
	dev, err := repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)

	found, err := repos.repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
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

func TestRepository_CreateDevice_InsertsAPIKeyRow(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("with-api-key", repos.ownerID)
	is.NoErr(err)
	dev, err := repos.repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Verify key_prefix is returned via GetDevice
	updated, err := repos.repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(updated.KeyPrefix, params.KeyPrefix)

	// Verify key_hash is stored: GetDeviceByAPIKeyHash must return the same device
	found, err := repos.repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.NoErr(err)
	is.Equal(found.ID, dev.ID)
}

func TestRepository_UpdateAPIKey_Success(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	oldParams, _, err := device.NewCreateDeviceParams("regen-device", repos.ownerID)
	is.NoErr(err)
	dev, err := repos.repo.CreateDevice(ctx, oldParams)
	is.NoErr(err)

	// Generate fresh key material via NewCreateDeviceParams (does not insert to DB)
	newKeyParams, _, err := device.NewCreateDeviceParams("unused-device-name", repos.ownerID)
	is.NoErr(err)

	err = repos.repo.UpdateAPIKey(ctx, dev.ID, newKeyParams.KeyHash, newKeyParams.KeyPrefix)
	is.NoErr(err)

	// Old hash should no longer authenticate
	_, err = repos.repo.GetDeviceByAPIKeyHash(ctx, oldParams.KeyHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)

	// New hash should authenticate
	found, err := repos.repo.GetDeviceByAPIKeyHash(ctx, newKeyParams.KeyHash)
	is.NoErr(err)
	is.Equal(found.ID, dev.ID)

	// GetDevice returns the updated prefix
	updated, err := repos.repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(updated.KeyPrefix, newKeyParams.KeyPrefix)
}

func TestRepository_UpdateAPIKey_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	newKeyParams, _, err := device.NewCreateDeviceParams("unused-device-name", repos.ownerID)
	is.NoErr(err)

	err = repos.repo.UpdateAPIKey(ctx, device.DeviceID(99999), newKeyParams.KeyHash, newKeyParams.KeyPrefix)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_CreateAddress(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")

	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.100")
	is.Equal(addr.DeviceID, dev.ID)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(!addr.CreatedAt.IsZero())
	is.True(addr.ID != 0)
}

func TestRepository_CreateAddress_SetsInitialState(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	// Address should be enabled with the correct source from creation, not from a subsequent update
	is.True(addr.IsEnabled)
	is.Equal(string(addr.Source), string(device.EventSourceManual))

	// created_at and updated_at must match — the address was created once, not created then updated
	is.Equal(addr.CreatedAt, addr.UpdatedAt)
}

func TestRepository_FindAddressForDeviceByIp(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	createdAddr := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.100")

	addr, err := repos.repo.GetAddressForDeviceByIP(ctx, dev.ID, netip.MustParseAddr("192.168.1.100"))
	is.NoErr(err)
	is.Equal(addr.ID, createdAddr.ID)
	is.Equal(addr.DeviceID, dev.ID)
	is.Equal(addr.IP, "192.168.1.100")
}

func TestRepository_FindAddressForDeviceByIp_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")

	_, err := repos.repo.GetAddressForDeviceByIP(ctx, dev.ID, netip.MustParseAddr("192.168.1.99"))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotFound)
}

func TestRepository_FindAddressForDeviceByIp_WrongDevice(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repos, ctx, "device-1")
	dev2 := createTestDevice(t, repos, ctx, "device-2")
	createTestAddress(t, repos.repo, ctx, dev1.ID, "192.168.1.100")

	_, err := repos.repo.GetAddressForDeviceByIP(ctx, dev2.ID, netip.MustParseAddr("192.168.1.100"))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotFound)
}

func TestRepository_DisableAddress(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.100")

	disabled, err := repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(disabled.ID, addr.ID)
	is.True(!disabled.IsEnabled)
}

func TestRepository_EnableAddress(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.100")

	_, err := repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	enabled, err := repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceManual)
	is.NoErr(err)
	is.Equal(enabled.ID, addr.ID)
	is.True(enabled.IsEnabled)
}

func TestRepository_GetAddress(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.100")

	got, err := repos.repo.GetAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(got.ID, addr.ID)
	is.Equal(got.DeviceID, dev.ID)
	is.Equal(got.IP, "192.168.1.100")
	is.True(got.IsEnabled)
	is.True(!got.CreatedAt.IsZero())
}

func TestRepository_GetAddress_NotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	_, err := repos.repo.GetAddress(ctx, device.AddressID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotFound)
}

func TestRepository_CheckAddressOwnership(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.100")

	err := repos.repo.CheckAddressOwnership(ctx, dev.ID, addr.ID)
	is.NoErr(err)
}

func TestRepository_CheckAddressOwnership_WrongDevice(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repos, ctx, "device-1")
	dev2 := createTestDevice(t, repos, ctx, "device-2")
	addr := createTestAddress(t, repos.repo, ctx, dev1.ID, "192.168.1.100")

	err := repos.repo.CheckAddressOwnership(ctx, dev2.ID, addr.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotOwnedByDevice)
}

func TestRepository_CheckAddressOwnership_AddressNotFound(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")

	err := repos.repo.CheckAddressOwnership(ctx, dev.ID, device.AddressID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotOwnedByDevice)
}

func TestRepository_GetEnabledUniqueIPs_Empty(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	ips, err := repos.repo.GetEnabledIPEntries(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 0)
}

func TestRepository_GetEnabledUniqueIPs(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "test-device")
	_ = createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.1")
	addrToDisable := createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.2")
	_ = createTestAddress(t, repos.repo, ctx, dev.ID, "192.168.1.3")

	_, err := repos.repo.DisableAddress(ctx, addrToDisable.ID)
	is.NoErr(err)

	ips, err := repos.repo.GetEnabledIPEntries(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 2)

	ipMap := make(map[string]bool)
	for _, ip := range ips {
		ipMap[ip.IP] = true
	}
	is.True(ipMap["192.168.1.1"])
	is.True(ipMap["192.168.1.3"])
	is.True(!ipMap["192.168.1.2"])
}

func TestRepository_GetEnabledUniqueIPs_Deduplicates(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repos, ctx, "device-1")
	dev2 := createTestDevice(t, repos, ctx, "device-2")
	dev3 := createTestDevice(t, repos, ctx, "device-3")

	_ = createTestAddress(t, repos.repo, ctx, dev1.ID, "192.168.1.100")
	_ = createTestAddress(t, repos.repo, ctx, dev2.ID, "192.168.1.100") // same IP as dev1
	_ = createTestAddress(t, repos.repo, ctx, dev3.ID, "192.168.1.200")

	ips, err := repos.repo.GetEnabledIPEntries(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 2) // 192.168.1.100 appears only once

	ipMap := make(map[string]bool)
	for _, ip := range ips {
		ipMap[ip.IP] = true
	}
	is.True(ipMap["192.168.1.100"])
	is.True(ipMap["192.168.1.200"])
}

func TestRepository_GetAddressHistory_ReturnsBucketsAndEvents(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "history-device")

	// Create address → records an enable event
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	// Disable → records a disable event
	_, err := repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Re-enable → records an enable event
	_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)

	// Should have 1 bucket (all events in the same hour)
	is.True(len(history.Buckets) >= 1)

	// Should have 3 events: create(enable), disable, enable
	is.Equal(len(history.Events), 3)
	is.Equal(history.TotalEvents, 3)

	// Events are ordered DESC (most recent first)
	is.True(history.Events[0].IsEnabled)                                            // re-enable
	is.Equal(string(history.Events[0].Source), string(device.EventSourceHeartbeat)) // heartbeat source
	is.True(!history.Events[1].IsEnabled)                                           // disable
	is.True(history.Events[2].IsEnabled)                                            // initial create

	// Events include device info
	is.Equal(history.Events[0].DeviceID, dev.ID)
	is.Equal(history.Events[0].DeviceName, "history-device")
}

func TestRepository_GetAddressHistory_EmptyRange(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "empty-history")
	createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	// Query a time range far in the past where no events exist
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)
	is.Equal(len(history.Buckets), 0)
	is.Equal(len(history.Events), 0)
	is.Equal(history.TotalEvents, 0)
}

func TestRepository_GetAddressHistory_DayGranularity(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "day-history")
	createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityDay,
		Limit:       50,
	})
	is.NoErr(err)

	// Should have exactly 1 bucket (all events on the same day)
	is.Equal(len(history.Buckets), 1)
	is.True(history.Buckets[0].EventCount >= 1)
}

func TestRepository_GetAddressHistory_AllDevices(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repos, ctx, "dev1")
	dev2 := createTestDevice(t, repos, ctx, "dev2")
	createTestAddress(t, repos.repo, ctx, dev1.ID, "10.0.0.1")
	createTestAddress(t, repos.repo, ctx, dev2.ID, "10.0.0.2")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// No device filter — should return events from both devices
	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)

	is.Equal(len(history.Events), 2)
	is.Equal(history.TotalEvents, 2)
}

func TestRepository_GetAddressHistory_FilterBySource(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "source-filter")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	// Disable and re-enable to create heartbeat event
	_, err := repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)
	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Source:      new(string(device.EventSourceHeartbeat)),
		Limit:       50,
	})
	is.NoErr(err)

	// Only heartbeat events (the re-enable + initial create is "manual", so only 1 heartbeat)
	for _, e := range history.Events {
		is.Equal(string(e.Source), string(device.EventSourceHeartbeat))
	}
}

func TestRepository_GetAddressHistory_EventsPagination(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "pagination")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	// Create several events: enable, disable x3
	for i := 0; i < 3; i++ {
		_, err := repos.repo.DisableAddress(ctx, addr.ID)
		is.NoErr(err)
		_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
		is.NoErr(err)
	}

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// Page 1: limit 3
	page1, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       3,
	})
	is.NoErr(err)
	is.Equal(len(page1.Events), 3)
	is.True(page1.TotalEvents > 3) // more events exist

	// Page 2: use cursor from last event of page 1
	cursor := page1.Events[len(page1.Events)-1].ID
	page2, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		BeforeID:    &cursor,
		Limit:       3,
	})
	is.NoErr(err)
	is.True(len(page2.Events) > 0)

	// Page 2 events should have lower IDs than cursor
	for _, e := range page2.Events {
		is.True(e.ID < cursor)
	}
}

func TestRepository_GetAddressHistory_StateChangesOnly(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "state-changes")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	// Simulate heartbeat refreshes (enable on already-enabled address)
	_, err := repos.repo.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)
	_, err = repos.repo.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	// Disable → actual state change
	_, err = repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Re-enable → actual state change
	_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// IncludeAll = true → should return all 5 events
	allHistory, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.Equal(allHistory.TotalEvents, 5) // create + 2 refreshes + disable + enable

	// IncludeAll = false (default) → should return only state changes
	changesHistory, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  false,
	})
	is.NoErr(err)
	is.Equal(changesHistory.TotalEvents, 3) // create, disable, enable

	// Verify the state changes are correct (most recent first)
	is.True(changesHistory.Events[0].IsEnabled)  // re-enable
	is.True(!changesHistory.Events[1].IsEnabled) // disable
	is.True(changesHistory.Events[2].IsEnabled)  // initial create

	// Buckets should still include all events regardless of IncludeAll
	is.Equal(allHistory.Buckets[0].EventCount, changesHistory.Buckets[0].EventCount)
}

func TestRepository_GetEnabledAddressesForDevice_ReturnsOnlyEnabled(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()
	dev := createTestDevice(t, repos, ctx, "enabled-filter-device")

	// Create two addresses, disable one
	addr1 := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")
	addr2 := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.2")

	_, err := repos.repo.DisableAddress(ctx, addr2.ID)
	is.NoErr(err)

	enabled, err := repos.repo.GetEnabledAddressesForDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(len(enabled), 1)
	is.Equal(enabled[0].ID, addr1.ID)
}

func TestRepository_GetEnabledAddressesForDevice_OrderedByUpdatedAtDesc(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()
	dev := createTestDevice(t, repos, ctx, "order-device")

	addr1 := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")
	addr2 := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.2")

	// Refresh addr1 so it has a more recent updated_at
	_, err := repos.repo.RefreshAddress(ctx, addr1.ID, device.EventSourceManual)
	is.NoErr(err)

	enabled, err := repos.repo.GetEnabledAddressesForDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(len(enabled), 2)
	// addr1 was refreshed more recently, should be first
	is.Equal(enabled[0].ID, addr1.ID)
	is.Equal(enabled[1].ID, addr2.ID)
}

func TestRepository_GetEnabledAddressesForDevice_EmptyWhenNone(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()
	dev := createTestDevice(t, repos, ctx, "empty-device")

	enabled, err := repos.repo.GetEnabledAddressesForDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(len(enabled), 0)
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

	// Now clear it
	dev.Description = nil
	updated, err := repos.repo.UpdateDevice(ctx, dev)

	is.NoErr(err)
	is.True(updated.Description == nil)
}

func TestRepository_UpdateDevice_NotFound(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	ghost := &device.Device{ID: device.DeviceID(9999), Name: "ghost", DeviceType: device.DeviceTypeStatic}
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
