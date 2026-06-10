//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/matryer/is"
)

func TestRepository_GetAddressHistory_EmptyRange(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "empty-history")
	createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	from := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
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
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "day-history")
	createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
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

func TestRepository_GetAddressHistory_FilterBySource(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "source-filter")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	_, err := repos.devices.DisableAddress(ctx, addr.ID)
	is.NoErr(err)
	_, err = repos.devices.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)
	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
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

// TestRepository_GetAddressHistory_FilterIPEscapesWildcards verifies the IP filter
// escapes LIKE wildcards (ADR-007 / PW-65): a `_` in the input must match literally
// rather than as a single-char wildcard. Without escaping, "0_1" would match the
// "0.1" substring of a stored IP.
func TestRepository_GetAddressHistory_FilterIPEscapesWildcards(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "ip-escape")
	_ = createAddress(t, repos.devices, dev.ID, "10.0.1.2")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	// Literal substring matches the stored IP.
	literal := "0.1"
	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		IP:          &literal,
		IncludeAll:  true,
		Limit:       50,
	})
	is.NoErr(err)
	is.True(len(history.Events) > 0)

	// "0_1" must NOT act as a wildcard (would otherwise match the "0.1" substring).
	wildcard := "0_1"
	history, err = repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		IP:          &wildcard,
		IncludeAll:  true,
		Limit:       50,
	})
	is.NoErr(err)
	is.Equal(len(history.Events), 0)
}

func TestRepository_GetAddressHistory_StateChangesOnly(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "state-changes")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	_, err := repos.devices.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)
	_, err = repos.devices.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	_, err = repos.devices.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	_, err = repos.devices.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	allHistory, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.Equal(allHistory.TotalEvents, 5)

	changesHistory, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
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

// --- Bucket active_count and gap_count semantics ---

// TestGetAddressHistory_BucketActiveCount_EndOfBucketState verifies that
// active_count reflects the state at the END of the bucket (last event per
// address), not "was ever enabled in the bucket".
func TestGetAddressHistory_BucketActiveCount_EndOfBucketState(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "gap-test-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.1") // creates heartbeat event

	// Expiry after creation → last event in bucket is disabled.
	_, err := repos.devices.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.True(len(history.Buckets) >= 1)
	// Last event was is_enabled=0 (DisableAddress) → active_count must be 0.
	is.Equal(history.Buckets[0].ActiveCount, 0)
	// An expiry occurred → gap_count must be 1.
	is.Equal(history.Buckets[0].GapCount, 1)
}

// TestGetAddressHistory_BucketActiveCount_RecoveredAddress verifies that when an
// address goes heartbeat→expiry→heartbeat in the same bucket, active_count=1
// (recovered) and gap_count=1 (the expiry happened).
func TestGetAddressHistory_BucketActiveCount_RecoveredAddress(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "recovered-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.2") // heartbeat

	_, err := repos.devices.DisableAddress(ctx, addr.ID) // expiry
	is.NoErr(err)
	_, err = repos.devices.EnableAddress(ctx, addr.ID, device.EventSourceHeartbeat) // heartbeat recovery
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.True(len(history.Buckets) >= 1)
	// Last event was is_enabled=1 (recovery) → active_count=1.
	is.Equal(history.Buckets[0].ActiveCount, 1)
	// Expiry occurred → gap_count=1.
	is.Equal(history.Buckets[0].GapCount, 1)
}

// TestGetAddressHistory_BucketGapCount_NoGapWhenOnlyHeartbeats verifies that
// gap_count=0 when an address only has heartbeat events in the bucket.
func TestGetAddressHistory_BucketGapCount_NoGapWhenOnlyHeartbeats(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "heartbeat-only-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.3") // heartbeat

	_, err := repos.devices.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat) // another heartbeat
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.True(len(history.Buckets) >= 1)
	// Only heartbeats → active at end, no gap.
	is.Equal(history.Buckets[0].ActiveCount, 1)
	is.Equal(history.Buckets[0].GapCount, 0)
}

// TestGetAddressHistory_BucketCounts_MultipleAddresses verifies that
// active_count and gap_count are computed independently per address.
func TestGetAddressHistory_BucketCounts_MultipleAddresses(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "multi-addr-device")
	addrA := createAddress(t, repos.devices, dev.ID, "10.0.1.1") // heartbeat only
	addrB := createAddress(t, repos.devices, dev.ID, "10.0.1.2") // heartbeat + expiry

	_, err := repos.devices.RefreshAddress(ctx, addrA.ID, device.EventSourceHeartbeat)
	is.NoErr(err)
	_, err = repos.devices.DisableAddress(ctx, addrB.ID)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.True(len(history.Buckets) >= 1)
	// addrA: active at end; addrB: disabled at end.
	is.Equal(history.Buckets[0].ActiveCount, 1)
	// Only addrB had an expiry.
	is.Equal(history.Buckets[0].GapCount, 1)
}

// --- New enrichment fields: time_gap_seconds, ip_changed, is_refresh, ttl_seconds ---

// TestRepository_GetAddressHistory_FirstEventHasNoComparison verifies that the
// earliest event for a device within the queried range has a null time gap and
// false change flags — there is nothing to compare it against.
func TestRepository_GetAddressHistory_FirstEventHasNoComparison(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "first-event-device")
	createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)
	is.Equal(len(history.Events), 1)

	first := history.Events[0]
	is.True(first.TimeGapSeconds == nil)
	is.True(!first.IPChanged)
	is.True(!first.IsRefresh)
}

// TestRepository_GetAddressHistory_RefreshDetection verifies that a repeated
// heartbeat on the same address (same IP, same enabled state) is reported as a
// refresh, while a state-changing event is not.
func TestRepository_GetAddressHistory_RefreshDetection(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "refresh-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	_, err := repos.devices.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)
	_, err = repos.devices.DisableAddress(ctx, addr.ID)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.Equal(len(history.Events), 3)

	// Returned newest-first: [disable, refresh, create].
	is.True(!history.Events[0].IsRefresh) // status changed (enabled → disabled)
	is.True(history.Events[1].IsRefresh)  // same IP, same enabled state as creation
	is.True(!history.Events[2].IsRefresh) // first event for the device, nothing to compare
}

// TestRepository_GetAddressHistory_IPChangedAcrossDeviceAddresses verifies that
// ip_changed is computed device-wide: switching from one of the device's
// addresses to another is flagged as a change, even though each address keeps
// a fixed IP.
func TestRepository_GetAddressHistory_IPChangedAcrossDeviceAddresses(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "hopping-device")
	addrA := createAddress(t, repos.devices, dev.ID, "10.0.0.1")
	_ = createAddress(t, repos.devices, dev.ID, "10.0.0.2")

	// Heartbeat returns to addrA's IP after addrB was created.
	_, err := repos.devices.RefreshAddress(ctx, addrA.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.Equal(len(history.Events), 3)

	// Newest-first: [refresh on 10.0.0.1, create 10.0.0.2, create 10.0.0.1].
	is.Equal(history.Events[0].IP, "10.0.0.1")
	is.True(history.Events[0].IPChanged) // previous device event was for 10.0.0.2

	is.Equal(history.Events[1].IP, "10.0.0.2")
	is.True(history.Events[1].IPChanged) // previous device event was for 10.0.0.1

	is.Equal(history.Events[2].IP, "10.0.0.1")
	is.True(!history.Events[2].IPChanged) // first event for the device
}

// TestRepository_GetAddressHistory_TimeGapComputed verifies time_gap_seconds is
// the elapsed time since the previous device event, scoped per device.
func TestRepository_GetAddressHistory_TimeGapComputed(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "time-gap-device")
	addr := createAddress(t, repos.devices, dev.ID, "10.0.0.1")

	_, err := repos.devices.RefreshAddress(ctx, addr.ID, device.EventSourceHeartbeat)
	is.NoErr(err)

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		DeviceIDs:   []ids.DeviceID{dev.ID},
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
		IncludeAll:  true,
	})
	is.NoErr(err)
	is.Equal(len(history.Events), 2)

	is.True(history.Events[0].TimeGapSeconds != nil)
	is.True(*history.Events[0].TimeGapSeconds >= 0)
	is.True(history.Events[1].TimeGapSeconds == nil)
}

// TestRepository_GetAddressHistory_TTLSecondsFromLeaseRule verifies that TTLSeconds
// reflects the device's enabled lease rule, and is null when no rule is configured.
func TestRepository_GetAddressHistory_TTLSecondsFromLeaseRule(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	withTTL := createDevice(t, repos, "ttl-device")
	createAddress(t, repos.devices, withTTL.ID, "10.0.0.1")

	const ttlSeconds = 3600
	config, err := rule.NewDeviceAddressLeaseConfig(ttlSeconds)
	is.NoErr(err)
	_, err = repos.rules.EnableDeviceAddressLeaseRuleConfig(ctx, withTTL.ID, config)
	is.NoErr(err)

	withoutTTL := createDevice(t, repos, "no-ttl-device")
	createAddress(t, repos.devices, withoutTTL.ID, "10.0.0.2")

	from := time.Now().UTC().Add(-1 * time.Hour)
	to := time.Now().UTC().Add(1 * time.Hour)

	history, err := repos.queries.GetAddressHistory(ctx, queries.AddressHistoryQuery{
		From:        from,
		To:          to,
		Granularity: timebucket.GranularityHour,
		Limit:       50,
	})
	is.NoErr(err)
	is.Equal(len(history.Events), 2)

	byDevice := make(map[ids.DeviceID]*int64, 2)
	for _, e := range history.Events {
		byDevice[e.DeviceID] = e.TTLSeconds
	}

	is.True(byDevice[withTTL.ID] != nil)
	is.Equal(*byDevice[withTTL.ID], int64(ttlSeconds))
	is.True(byDevice[withoutTTL.ID] == nil)
}
