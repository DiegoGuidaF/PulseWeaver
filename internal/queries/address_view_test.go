//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/matryer/is"
)

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
			`INSERT INTO addresses (device_id, ip, created_at, is_enabled, source, updated_at) VALUES (?, ?, ?, 1, 'manual', ?) RETURNING id`,
			dev.ID, ip, createdAt, createdAt,
		)
		if err != nil {
			t.Fatalf("insert address %q: %v", ip, err)
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
