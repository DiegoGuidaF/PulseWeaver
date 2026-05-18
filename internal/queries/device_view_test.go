//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

func TestRepository_GetDevices_EmptySlice(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	devices, err := repos.queries.GetDevices(t.Context(), nil)
	is.NoErr(err)
	is.Equal(len(devices), 0)
}

func TestRepository_GetDevices_ReturnsFields(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos, "device-fields")

	devices, err := repos.queries.GetDevices(t.Context(), nil)
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

	createDevice(t, repos, "device-no-addresses")

	devices, err := repos.queries.GetDevices(t.Context(), nil)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].AddressCount, 0)
}

func TestRepository_GetDevices_AddressCountOnlyCountsEnabledAddresses(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	dev := createDevice(t, repos, "device-count")
	addr1 := createAddress(t, repos.devices, dev.ID, "10.0.0.10")
	createAddress(t, repos.devices, dev.ID, "10.0.0.11")

	_, err := repos.devices.DisableAddress(t.Context(), addr1.ID)
	is.NoErr(err)

	devices, err := repos.queries.GetDevices(t.Context(), nil)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].AddressCount, 1)
}

func TestRepository_GetDevices_OrderedByCreatedAtDesc(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	olderTime := time.Now().UTC().Add(-2 * time.Second).Truncate(time.Second)
	newerTime := time.Now().UTC().Add(-1 * time.Second).Truncate(time.Second)

	insertDevice := func(name, prefix, hash string, createdAt time.Time) ids.DeviceID {
		t.Helper()

		var id ids.DeviceID
		err := repos.db.GetContext(t.Context(), &id,
			`INSERT INTO devices (name, created_at, owner_id) VALUES (?, ?, ?) RETURNING id`,
			name, createdAt, 1,
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

	devices, err := repos.queries.GetDevices(t.Context(), nil)
	is.NoErr(err)
	is.Equal(len(devices), 2)
	is.Equal(devices[0].ID, newID)
	is.Equal(devices[1].ID, oldID)
}
