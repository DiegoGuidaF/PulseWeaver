//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/matryer/is"
)

func TestRepository_ListAccessLogStatsByCountry(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	// Insert access events with GeoIP data via the access repository.
	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
			GeoIP:     geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 15169, ASNOrg: "Google LLC"},
		},
		{
			ClientIP:   "8.8.4.4",
			Outcome:    false,
			DenyReason: new(policy.DenyReasonIPNotRegistered),
			CreatedAt:  time.Now().UTC(),
			Headers:    map[string][]string{},
			GeoIP:      geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 15169, ASNOrg: "Google LLC"},
		},
		{
			ClientIP:  "1.1.1.1",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
			GeoIP:     geoip.Result{CountryCode: "AU", CountryName: "Australia", ContinentCode: "OC", ASN: 13335, ASNOrg: "Cloudflare"},
		},
		{
			// No GeoIP — should not appear in stats.
			ClientIP:  "192.168.1.1",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
	})
	is.NoErr(err)

	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour)
	stats, err := repos.queries.ListAccessLogStatsByCountry(ctx, from, now)
	is.NoErr(err)

	// Should have 2 countries (US and AU); private IP excluded.
	is.Equal(len(stats), 2)

	// US should be first (2 total > 1 total).
	is.Equal(stats[0].CountryCode, "US")
	is.Equal(stats[0].CountryName, "United States")
	is.Equal(stats[0].ContinentCode, "NA")
	is.Equal(stats[0].Total, int64(2))
	is.Equal(stats[0].Allowed, int64(1))
	is.Equal(stats[0].Denied, int64(1))

	// AU second.
	is.Equal(stats[1].CountryCode, "AU")
	is.Equal(stats[1].CountryName, "Australia")
	is.Equal(stats[1].ContinentCode, "OC")
	is.Equal(stats[1].Total, int64(1))
	is.Equal(stats[1].Allowed, int64(1))
	is.Equal(stats[1].Denied, int64(0))
}

func TestRepository_ListaccessLogStatsByCountry_Empty(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour)
	stats, err := repos.queries.ListAccessLogStatsByCountry(t.Context(), from, now)
	is.NoErr(err)
	is.Equal(len(stats), 0)
}

func TestRepository_ListaccessLogStatsByCountry_FromFilter(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	// Insert an old event (created 2 hours ago).
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: oldTime,
			Headers:   map[string][]string{},
			GeoIP:     geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA"},
		},
	})
	is.NoErr(err)

	// Query with from = 1 hour ago — should exclude the old event.
	now := time.Now().UTC()
	from := now.Add(-1 * time.Hour)
	stats, err := repos.queries.ListAccessLogStatsByCountry(ctx, from, now)
	is.NoErr(err)
	is.Equal(len(stats), 0)
}

func TestRepository_ListaccessLogStatsByCountry_ToFilter(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	now := time.Now().UTC()

	// Insert a recent event.
	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: now,
			Headers:   map[string][]string{},
			GeoIP:     geoip.Result{CountryCode: "DE", CountryName: "Germany", ContinentCode: "EU"},
		},
	})
	is.NoErr(err)

	// Query with to = 1 hour ago — should exclude the event created at now.
	from := now.Add(-2 * time.Hour)
	to := now.Add(-1 * time.Hour)
	stats, err := repos.queries.ListAccessLogStatsByCountry(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(stats), 0)

	// Query with to = now — should include the event.
	stats, err = repos.queries.ListAccessLogStatsByCountry(ctx, from, now)
	is.NoErr(err)
	is.Equal(len(stats), 1)
	is.Equal(stats[0].CountryCode, "DE")
}

// ── ListAccessLog ─────────────────────────────────────────────────────────────

func TestListAccessLog_Empty(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	rows, total, err := repos.queries.ListAccessLog(t.Context(), queries.AccessLogQuery{
		From:  time.Now().UTC().Add(-time.Hour),
		To:    time.Now().UTC(),
		Limit: 50,
	})
	is.NoErr(err)
	is.Equal(total, 0)
	is.Equal(len(rows), 0)
}

func TestListAccessLog_WithContributor(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "access-log-dev")
	addr := createAddress(t, repos.devices, dev.ID, "10.1.1.1")

	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP: "10.1.1.1",
			Outcome:  true,
			IPContributors: []policy.IPContributor{
				{DeviceID: dev.ID, AddressID: addr.ID, UserID: repos.testOwnerID},
			},
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
	})
	is.NoErr(err)

	rows, total, err := repos.queries.ListAccessLog(ctx, queries.AccessLogQuery{
		From:  time.Now().UTC().Add(-time.Hour),
		To:    time.Now().UTC(),
		Limit: 50,
	})
	is.NoErr(err)
	is.Equal(total, 1)
	is.Equal(len(rows), 1)

	row := rows[0]
	is.Equal(row.ClientIP, "10.1.1.1")
	is.Equal(row.Outcome, true)
	is.True(row.DeviceID != nil)
	is.Equal(*row.DeviceID, dev.ID)
	is.True(row.AddressID != nil)
	is.Equal(*row.AddressID, addr.ID)
	is.True(row.DeviceName != nil)
	is.Equal(*row.DeviceName, dev.Name)
}

func TestListAccessLog_NoContributor(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "9.9.9.9", Outcome: false, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	})
	is.NoErr(err)

	rows, total, err := repos.queries.ListAccessLog(ctx, queries.AccessLogQuery{
		From:  time.Now().UTC().Add(-time.Hour),
		To:    time.Now().UTC(),
		Limit: 50,
	})
	is.NoErr(err)
	is.Equal(total, 1)
	is.Equal(len(rows), 1)

	row := rows[0]
	is.Equal(row.ClientIP, "9.9.9.9")
	is.True(row.DeviceID == nil)
	is.True(row.AddressID == nil)
	is.True(row.DeviceName == nil)
}

// TestListAccessLog_NoFanOut is the critical regression test: multiple access_log
// entries whose contributors share the same user_id must not produce duplicate rows.
// The old query joined on user_id alone without constraining by access_log_id, causing
// each entry to expand into N rows where N is the total number of contributors with
// that user_id.
func TestListAccessLog_NoFanOut(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev1 := createDevice(t, repos, "fan-out-dev-1")
	dev2 := createDevice(t, repos, "fan-out-dev-2")
	addr1 := createAddress(t, repos.devices, dev1.ID, "10.2.1.1")
	addr2 := createAddress(t, repos.devices, dev2.ID, "10.2.1.2")

	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP: "10.2.1.1",
			Outcome:  true,
			IPContributors: []policy.IPContributor{
				{DeviceID: dev1.ID, AddressID: addr1.ID, UserID: repos.testOwnerID},
			},
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
		{
			ClientIP: "10.2.1.2",
			Outcome:  true,
			IPContributors: []policy.IPContributor{
				{DeviceID: dev2.ID, AddressID: addr2.ID, UserID: repos.testOwnerID},
			},
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
	})
	is.NoErr(err)

	rows, total, err := repos.queries.ListAccessLog(ctx, queries.AccessLogQuery{
		From:  time.Now().UTC().Add(-time.Hour),
		To:    time.Now().UTC(),
		Limit: 50,
	})
	is.NoErr(err)
	is.Equal(total, 2)
	is.Equal(len(rows), 2) // must be exactly 2, not 4
}

func TestListAccessLog_MultipleContributors(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev1 := createDevice(t, repos, "multi-dev-1")
	dev2 := createDevice(t, repos, "multi-dev-2")
	addr1 := createAddress(t, repos.devices, dev1.ID, "10.3.1.1")
	addr2 := createAddress(t, repos.devices, dev2.ID, "10.3.1.2")

	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP: "10.3.1.1",
			Outcome:  true,
			IPContributors: []policy.IPContributor{
				{DeviceID: dev1.ID, AddressID: addr1.ID, UserID: repos.testOwnerID},
				{DeviceID: dev2.ID, AddressID: addr2.ID, UserID: repos.testOwnerID},
			},
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
	})
	is.NoErr(err)

	rows, total, err := repos.queries.ListAccessLog(ctx, queries.AccessLogQuery{
		From:  time.Now().UTC().Add(-time.Hour),
		To:    time.Now().UTC(),
		Limit: 50,
	})
	is.NoErr(err)
	is.Equal(total, 1)
	is.Equal(len(rows), 1) // exactly one row for one access_log entry
	is.True(rows[0].DeviceID != nil)
}

func TestListAccessLog_FilterDeviceID(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dev := createDevice(t, repos, "filter-dev")
	addr := createAddress(t, repos.devices, dev.ID, "10.4.1.1")

	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP: "10.4.1.1",
			Outcome:  true,
			IPContributors: []policy.IPContributor{
				{DeviceID: dev.ID, AddressID: addr.ID, UserID: repos.testOwnerID},
			},
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
		{ClientIP: "9.9.9.9", Outcome: false, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	})
	is.NoErr(err)

	devID := dev.ID
	rows, total, err := repos.queries.ListAccessLog(ctx, queries.AccessLogQuery{
		DeviceID: &devID,
		From:     time.Now().UTC().Add(-time.Hour),
		To:       time.Now().UTC(),
		Limit:    50,
	})
	is.NoErr(err)
	is.Equal(total, 1)
	is.Equal(len(rows), 1)
	is.True(rows[0].DeviceID != nil)
	is.Equal(*rows[0].DeviceID, dev.ID)
}

func TestListAccessLog_PaginationBeforeID(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "3.3.3.3", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	})
	is.NoErr(err)

	q := queries.AccessLogQuery{
		From:  time.Now().UTC().Add(-time.Hour),
		To:    time.Now().UTC(),
		Limit: 2,
	}

	page1, total1, err := repos.queries.ListAccessLog(ctx, q)
	is.NoErr(err)
	is.Equal(total1, 3)
	is.Equal(len(page1), 2)

	cursor := page1[len(page1)-1].ID
	q.BeforeID = &cursor

	page2, total2, err := repos.queries.ListAccessLog(ctx, q)
	is.NoErr(err)
	is.Equal(total2, 3) // total reflects all matching rows, cursor does not affect it
	is.Equal(len(page2), 1)
	for _, row := range page2 {
		is.True(row.ID < cursor)
	}
}
