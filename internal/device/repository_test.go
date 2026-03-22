//go:build test

package device_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupTestDB(t *testing.T) *device.Repository {
	t.Helper()

	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	return device.NewRepository(db.DB())
}

func createTestDevice(t *testing.T, repo *device.Repository, ctx context.Context, name string) *device.Device {
	t.Helper()

	params, _, err := device.NewCreateDeviceParams(name)
	if err != nil {
		t.Fatalf("create device params %q: %v", name, err)
	}
	dev, err := repo.CreateDevice(ctx, params)
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

	params, _, err := device.NewCreateDeviceParams("test-device")
	is.NoErr(err)
	dev, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)
	is.Equal(dev.Name, "test-device")
	is.True(!dev.CreatedAt.IsZero())
}

func TestRepository_CreateDevice_DuplicateName(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("duplicate-name")
	is.NoErr(err)
	_, err = repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Try to create device with same name (active unique index)
	_, err = repo.CreateDevice(ctx, params)
	is.True(err != nil)
	is.True(errors.Is(err, device.ErrDuplicateDeviceName))
}

func TestRepository_CreateDevice_SameNameAfterSoftDelete(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "reused-name")
	err := repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	// Same name is allowed again
	params, _, err := device.NewCreateDeviceParams("reused-name")
	is.NoErr(err)
	second, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)
	is.True(second.ID != dev.ID)
	is.Equal(second.Name, "reused-name")
}

func TestRepository_DeleteDevice_Success(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "to-delete")
	err := repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	// Deleted device is hidden from GetDevice
	_, err = repo.GetDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	err := repo.DeleteDevice(ctx, device.DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_DeleteDevice_AlreadyDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "deleted-once")
	err := repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	// Second delete returns not found (idempotent 404)
	err = repo.DeleteDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDevice_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "hidden-after-delete")
	_, err := repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)

	err = repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	_, err = repo.GetDevice(ctx, dev.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDeviceByAPIKeyHash_HidesDeleted(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("apikey-device")
	is.NoErr(err)
	dev, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	_, err = repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.NoErr(err)

	err = repo.DeleteDevice(ctx, dev.ID)
	is.NoErr(err)

	_, err = repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("test-device")
	is.NoErr(err)
	created, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	got, err := repo.GetDevice(ctx, created.ID)
	is.NoErr(err)
	is.Equal(got.ID, created.ID)
	is.Equal(got.Name, "test-device")
	is.True(!got.CreatedAt.IsZero())
}

func TestRepository_GetDevice_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetDevice(ctx, device.DeviceID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_GetDeviceByAPIKeyHash_Success(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("lookup-device")
	is.NoErr(err)
	dev, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	found, err := repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.NoErr(err)
	is.True(found != nil)
	is.Equal(found.ID, dev.ID)
	is.Equal(found.Name, "lookup-device")
}

func TestRepository_GetDeviceByAPIKeyHash_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetDeviceByAPIKeyHash(ctx, "nonexistent-hash")
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_CreateDevice_InsertsAPIKeyRow(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	params, _, err := device.NewCreateDeviceParams("with-api-key")
	is.NoErr(err)
	dev, err := repo.CreateDevice(ctx, params)
	is.NoErr(err)

	// Verify key_prefix is returned via GetDevice
	updated, err := repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(updated.KeyPrefix, params.KeyPrefix)

	// Verify key_hash is stored: GetDeviceByAPIKeyHash must return the same device
	found, err := repo.GetDeviceByAPIKeyHash(ctx, params.KeyHash)
	is.NoErr(err)
	is.Equal(found.ID, dev.ID)
}

func TestRepository_UpdateAPIKey_Success(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	oldParams, _, err := device.NewCreateDeviceParams("regen-device")
	is.NoErr(err)
	dev, err := repo.CreateDevice(ctx, oldParams)
	is.NoErr(err)

	// Generate fresh key material via NewCreateDeviceParams (does not insert to DB)
	newKeyParams, _, err := device.NewCreateDeviceParams("unused-device-name")
	is.NoErr(err)

	err = repo.UpdateAPIKey(ctx, dev.ID, newKeyParams.KeyHash, newKeyParams.KeyPrefix)
	is.NoErr(err)

	// Old hash should no longer authenticate
	_, err = repo.GetDeviceByAPIKeyHash(ctx, oldParams.KeyHash)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)

	// New hash should authenticate
	found, err := repo.GetDeviceByAPIKeyHash(ctx, newKeyParams.KeyHash)
	is.NoErr(err)
	is.Equal(found.ID, dev.ID)

	// GetDevice returns the updated prefix
	updated, err := repo.GetDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(updated.KeyPrefix, newKeyParams.KeyPrefix)
}

func TestRepository_UpdateAPIKey_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	newKeyParams, _, err := device.NewCreateDeviceParams("unused-device-name")
	is.NoErr(err)

	err = repo.UpdateAPIKey(ctx, device.DeviceID(99999), newKeyParams.KeyHash, newKeyParams.KeyPrefix)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
}

func TestRepository_CreateAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")

	addr := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.100")
	is.Equal(addr.DeviceID, dev.ID)
	is.Equal(addr.IP, "192.168.1.100")
	is.True(!addr.CreatedAt.IsZero())
	is.True(addr.ID != 0)
}

func TestRepository_FindAddressForDeviceByIp(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")
	createdAddr := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.100")

	addr, err := repo.GetAddressForDeviceByIP(ctx, dev.ID, netip.MustParseAddr("192.168.1.100"))
	is.NoErr(err)
	is.Equal(addr.ID, createdAddr.ID)
	is.Equal(addr.DeviceID, dev.ID)
	is.Equal(addr.IP, "192.168.1.100")
}

func TestRepository_FindAddressForDeviceByIp_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")

	_, err := repo.GetAddressForDeviceByIP(ctx, dev.ID, netip.MustParseAddr("192.168.1.99"))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotFound)
}

func TestRepository_FindAddressForDeviceByIp_WrongDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repo, ctx, "device-1")
	dev2 := createTestDevice(t, repo, ctx, "device-2")
	createTestAddress(t, repo, ctx, dev1.ID, "192.168.1.100")

	_, err := repo.GetAddressForDeviceByIP(ctx, dev2.ID, netip.MustParseAddr("192.168.1.100"))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotFound)
}

func TestRepository_DisableAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.100")

	disabled, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(disabled.ID, addr.ID)
	is.True(!disabled.IsEnabled)
}

func TestRepository_EnableAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.100")

	_, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	enabled, err := repo.EnableAddress(ctx, addr.ID, device.EventSourceManual)
	is.NoErr(err)
	is.Equal(enabled.ID, addr.ID)
	is.True(enabled.IsEnabled)
}

func TestRepository_GetAddress(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.100")

	got, err := repo.GetAddress(ctx, addr.ID)
	is.NoErr(err)
	is.Equal(got.ID, addr.ID)
	is.Equal(got.DeviceID, dev.ID)
	is.Equal(got.IP, "192.168.1.100")
	is.True(got.IsEnabled)
	is.True(!got.CreatedAt.IsZero())
}

func TestRepository_GetAddress_NotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetAddress(ctx, device.AddressID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotFound)
}

func TestRepository_CheckAddressOwnership(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")
	addr := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.100")

	err := repo.CheckAddressOwnership(ctx, dev.ID, addr.ID)
	is.NoErr(err)
}

func TestRepository_CheckAddressOwnership_WrongDevice(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repo, ctx, "device-1")
	dev2 := createTestDevice(t, repo, ctx, "device-2")
	addr := createTestAddress(t, repo, ctx, dev1.ID, "192.168.1.100")

	err := repo.CheckAddressOwnership(ctx, dev2.ID, addr.ID)
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotOwnedByDevice)
}

func TestRepository_CheckAddressOwnership_AddressNotFound(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")

	err := repo.CheckAddressOwnership(ctx, dev.ID, device.AddressID(99999))
	is.True(err != nil)
	is.Equal(err, device.ErrAddressNotOwnedByDevice)
}

func TestRepository_GetEnabledUniqueIPs_Empty(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	ips, err := repo.GetEnabledIPEntries(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 0)
}

func TestRepository_GetEnabledUniqueIPs(t *testing.T) {
	is := is.New(t)

	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "test-device")
	_ = createTestAddress(t, repo, ctx, dev.ID, "192.168.1.1")
	addrToDisable := createTestAddress(t, repo, ctx, dev.ID, "192.168.1.2")
	_ = createTestAddress(t, repo, ctx, dev.ID, "192.168.1.3")

	_, err := repo.DisableAddress(ctx, addrToDisable.ID)
	is.NoErr(err)

	ips, err := repo.GetEnabledIPEntries(ctx)
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

	repo := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repo, ctx, "device-1")
	dev2 := createTestDevice(t, repo, ctx, "device-2")
	dev3 := createTestDevice(t, repo, ctx, "device-3")

	_ = createTestAddress(t, repo, ctx, dev1.ID, "192.168.1.100")
	_ = createTestAddress(t, repo, ctx, dev2.ID, "192.168.1.100") // same IP as dev1
	_ = createTestAddress(t, repo, ctx, dev3.ID, "192.168.1.200")

	ips, err := repo.GetEnabledIPEntries(ctx)
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
	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "history-device")

	// Create address → records an enable event
	addr := createTestAddress(t, repo, ctx, dev.ID, "10.0.0.1")

	// Disable → records a disable event
	_, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	// Re-enable → records an enable event
	_, err = repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: device.GranularityHour,
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
	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "empty-history")
	createTestAddress(t, repo, ctx, dev.ID, "10.0.0.1")

	// Query a time range far in the past where no events exist
	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	history, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: device.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)
	is.Equal(len(history.Buckets), 0)
	is.Equal(len(history.Events), 0)
	is.Equal(history.TotalEvents, 0)
}

func TestRepository_GetAddressHistory_DayGranularity(t *testing.T) {
	is := is.New(t)
	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "day-history")
	createTestAddress(t, repo, ctx, dev.ID, "10.0.0.1")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []device.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: device.GranularityDay,
		Limit:       50,
	})
	is.NoErr(err)

	// Should have exactly 1 bucket (all events on the same day)
	is.Equal(len(history.Buckets), 1)
	is.True(history.Buckets[0].EventCount >= 1)
}

func TestRepository_GetAddressHistory_AllDevices(t *testing.T) {
	is := is.New(t)
	repo := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repo, ctx, "dev1")
	dev2 := createTestDevice(t, repo, ctx, "dev2")
	createTestAddress(t, repo, ctx, dev1.ID, "10.0.0.1")
	createTestAddress(t, repo, ctx, dev2.ID, "10.0.0.2")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// No device filter — should return events from both devices
	history, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: device.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)

	is.Equal(len(history.Events), 2)
	is.Equal(history.TotalEvents, 2)
}

func TestRepository_GetAddressHistory_FilterBySource(t *testing.T) {
	is := is.New(t)
	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "source-filter")
	addr := createTestAddress(t, repo, ctx, dev.ID, "10.0.0.1")

	// Disable and re-enable to create heartbeat event
	_, err := repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	_, err = repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)
	source := string(device.EventSourceHeartbeat)

	history, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: device.GranularityHour,
		Source:      &source,
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
	repo := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repo, ctx, "pagination")
	addr := createTestAddress(t, repo, ctx, dev.ID, "10.0.0.1")

	// Create several events: enable, disable x3
	for i := 0; i < 3; i++ {
		_, err := repo.DisableAddress(ctx, addr.ID)
		is.NoErr(err)
		_, err = repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
		is.NoErr(err)
	}

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// Page 1: limit 3
	page1, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: device.GranularityHour,
		Limit:       3,
	})
	is.NoErr(err)
	is.Equal(len(page1.Events), 3)
	is.True(page1.TotalEvents > 3) // more events exist

	// Page 2: use cursor from last event of page 1
	cursor := page1.Events[len(page1.Events)-1].ID
	page2, err := repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: device.GranularityHour,
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
