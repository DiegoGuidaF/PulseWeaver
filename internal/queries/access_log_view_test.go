//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// ListAccessLog itself is exercised end-to-end in handler_access_log_test.go against
// the full seeded world — filters, sort, keyset pagination, and the multi-contributor
// assembly are too data-complex to test honestly against hand-built rows. The country
// rollup below is a simpler single-purpose aggregation, kept as a focused repository test.

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
