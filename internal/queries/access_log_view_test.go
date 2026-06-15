//go:build test

package queries_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
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

// TestDashboardWidgets_CrossWidgetConsistency_WideWindow guards the F18
// invariant: for the same window every dashboard widget answers from the same
// source. For a window > dashboard.RawWindowThreshold all widgets take the
// aggregate path, so after a rollup catch-up the summary stats, the traffic
// series sums, and the country stats must all describe the same seeded rows.
func TestDashboardWidgets_CrossWidgetConsistency_WideWindow(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	dashRepo := dashboard.NewRepository(repos.db, nil)

	usGeo := geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA"}
	auGeo := geoip.Result{CountryCode: "AU", CountryName: "Australia", ContinentCode: "OC"}
	appHost := "app.example.com"
	otherHost := "other.example.com"

	// All rows sit in complete past hours: the in-flight hour is never rolled
	// up, so rows there would be invisible to every aggregate-path widget.
	oldHour := time.Now().UTC().Truncate(time.Hour).Add(-30 * time.Hour)
	recentHour := time.Now().UTC().Truncate(time.Hour).Add(-2 * time.Hour)
	err := repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "8.8.8.8", TargetHost: &appHost, Outcome: true, CreatedAt: oldHour.Add(5 * time.Minute), Headers: map[string][]string{}, GeoIP: usGeo},
		{ClientIP: "8.8.8.8", TargetHost: &appHost, Outcome: true, CreatedAt: oldHour.Add(10 * time.Minute), Headers: map[string][]string{}, GeoIP: usGeo},
		{ClientIP: "8.8.4.4", TargetHost: &appHost, Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), CreatedAt: oldHour.Add(15 * time.Minute), Headers: map[string][]string{}, GeoIP: usGeo},
		{ClientIP: "1.1.1.1", TargetHost: &otherHost, Outcome: true, CreatedAt: recentHour.Add(5 * time.Minute), Headers: map[string][]string{}, GeoIP: auGeo},
		{ClientIP: "1.1.1.1", TargetHost: &otherHost, Outcome: false, DenyReason: new(policy.DenyReasonHostNotAllowed), CreatedAt: recentHour.Add(10 * time.Minute), Headers: map[string][]string{}, GeoIP: auGeo},
	})
	is.NoErr(err)

	is.NoErr(dashRepo.NewRollupJob(slog.New(slog.DiscardHandler)).Run(ctx))

	to := time.Now().UTC()
	from := to.Add(-48 * time.Hour)
	is.True(to.Sub(from) > dashboard.RawWindowThreshold) // all widgets on the aggregate path

	stats, err := dashRepo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.TotalRequests, int64(5))
	is.Equal(stats.AllowedCount, int64(3))
	is.Equal(stats.DeniedCount, int64(2))

	series, err := dashRepo.GetTrafficSeries(ctx, from, to)
	is.NoErr(err)
	var seriesAllowed, seriesDenied int64
	for _, bucket := range series {
		seriesAllowed += bucket.AllowCount
		seriesDenied += bucket.DenyCount
	}
	is.Equal(seriesAllowed, stats.AllowedCount)
	is.Equal(seriesDenied, stats.DeniedCount)

	countries, err := repos.queries.ListAccessLogStatsByCountry(ctx, from, to)
	is.NoErr(err)
	var countryTotal, countryAllowed, countryDenied int64
	for _, c := range countries {
		countryTotal += c.Total
		countryAllowed += c.Allowed
		countryDenied += c.Denied
	}
	is.Equal(countryTotal, stats.TotalRequests)
	is.Equal(countryAllowed, stats.AllowedCount)
	is.Equal(countryDenied, stats.DeniedCount)

	is.Equal(len(countries), 2) // US first (3 > 2)
	is.Equal(countries[0].CountryCode, "US")
	is.Equal(countries[0].Total, int64(3))
	is.Equal(countries[1].CountryCode, "AU")
	is.Equal(countries[1].Total, int64(2))
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
