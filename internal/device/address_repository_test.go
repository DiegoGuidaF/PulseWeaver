//go:build test

package device_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/matryer/is"
)

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

	_, err := repos.repo.GetAddress(ctx, ids.AddressID(99999))
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

	err := repos.repo.CheckAddressOwnership(ctx, dev.ID, ids.AddressID(99999))
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

func TestRepository_GetEnabledIPEntries_ReturnsAllRows_SharedIPNotDeduplicated(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()

	dev1 := createTestDevice(t, repos, ctx, "device-1")
	dev2 := createTestDevice(t, repos, ctx, "device-2")
	dev3 := createTestDevice(t, repos, ctx, "device-3")

	_ = createTestAddress(t, repos.repo, ctx, dev1.ID, "192.168.1.100")
	_ = createTestAddress(t, repos.repo, ctx, dev2.ID, "192.168.1.100")
	_ = createTestAddress(t, repos.repo, ctx, dev3.ID, "192.168.1.200")

	ips, err := repos.repo.GetEnabledIPEntries(ctx)
	is.NoErr(err)
	is.Equal(len(ips), 3)

	ipCount := make(map[string]int)
	for _, ip := range ips {
		ipCount[ip.IP]++
	}
	is.Equal(ipCount["192.168.1.100"], 2)
	is.Equal(ipCount["192.168.1.200"], 1)
}

func TestRepository_GetAddressHistory_ReturnsBucketsAndEvents(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "history-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	_, err := repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)

	is.True(len(history.Buckets) >= 1)
	is.Equal(len(history.Events), 3)
	is.Equal(history.TotalEvents, 3)

	is.True(history.Events[0].IsEnabled)
	is.Equal(string(history.Events[0].Source), string(device.EventSourceHeartbeat))
	is.True(!history.Events[1].IsEnabled)
	is.True(history.Events[2].IsEnabled)

	is.Equal(history.Events[0].DeviceID, dev.ID)
	is.Equal(history.Events[0].DeviceName, "history-device")
}

func TestRepository_GetAddressHistory_EmptyRange(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "empty-history")
	createTestAddress(t, repos.repo, ctx, dev.ID, "10.0.0.1")

	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	history, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
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
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityDay,
		Limit:       50,
	})
	is.NoErr(err)
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

	for i := 0; i < 3; i++ {
		_, err := repos.repo.DisableAddress(ctx, addr.ID)
		is.NoErr(err)
		_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
		is.NoErr(err)
	}

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	page1, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       3,
	})
	is.NoErr(err)
	is.Equal(len(page1.Events), 3)
	is.True(page1.TotalEvents > 3)

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

	_, err := repos.repo.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)
	_, err = repos.repo.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	_, err = repos.repo.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	_, err = repos.repo.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	allHistory, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.Equal(allHistory.TotalEvents, 5)

	changesHistory, err := repos.repo.GetAddressHistory(ctx, device.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  false,
	})
	is.NoErr(err)
	is.Equal(changesHistory.TotalEvents, 3)

	is.True(changesHistory.Events[0].IsEnabled)
	is.True(!changesHistory.Events[1].IsEnabled)
	is.True(changesHistory.Events[2].IsEnabled)

	is.Equal(allHistory.Buckets[0].EventCount, changesHistory.Buckets[0].EventCount)
}

func TestRepository_GetEnabledAddressesForDevice_ReturnsOnlyEnabled(t *testing.T) {
	is := is.New(t)

	repos := setupTestDB(t)
	ctx := context.Background()
	dev := createTestDevice(t, repos, ctx, "enabled-filter-device")

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

	_, err := repos.repo.RefreshAddress(ctx, addr1.ID, device.EventSourceManual)
	is.NoErr(err)

	enabled, err := repos.repo.GetEnabledAddressesForDevice(ctx, dev.ID)
	is.NoErr(err)
	is.Equal(len(enabled), 2)
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
