//go:build test

package device_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
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

// DeleteAddressEventsOlderThan

func TestRepository_DeleteAddressEventsOlderThan_RemovesOldRows(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)
	ctx := context.Background()

	dev := createTestDevice(t, repos, ctx, "retention-device")
	addr := createTestAddress(t, repos.repo, ctx, dev.ID, "10.99.0.1")

	// Insert two extra address_events backdated to 48 hours ago.
	old := time.Now().UTC().Add(-48 * time.Hour)
	for i := 0; i < 2; i++ {
		if _, err := repos.db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'manual', ?)`,
			addr.ID, old,
		); err != nil {
			t.Fatalf("insert old event: %v", err)
		}
	}

	cutoff := time.Now().UTC().Add(-24 * time.Hour)

	deleted, err := repos.repo.DeleteAddressEventsOlderThan(ctx, cutoff)
	is.NoErr(err)
	is.Equal(deleted, int64(2))

	// A second call with the same cutoff should delete nothing.
	deleted2, err := repos.repo.DeleteAddressEventsOlderThan(ctx, cutoff)
	is.NoErr(err)
	is.Equal(deleted2, int64(0))
}

func TestRepository_DeleteAddressEventsOlderThan_EmptyTable_ReturnsZero(t *testing.T) {
	is := is.New(t)
	repos := setupTestDB(t)

	deleted, err := repos.repo.DeleteAddressEventsOlderThan(context.Background(), time.Now())
	is.NoErr(err)
	is.Equal(deleted, int64(0))
}
