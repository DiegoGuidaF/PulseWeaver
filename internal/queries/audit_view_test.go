//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

func TestRepository_ListAuditLogStatsByCountry(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	// Insert audit events with GeoIP data via the audit repository.
	err := repos.audit.BatchInsert(ctx, []policy.DecisionEvent{
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
			DenyReason: ptrDenyReason(policy.DenyReasonIPNotRegistered),
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

	since := time.Now().UTC().Add(-1 * time.Hour)
	stats, err := repos.queries.ListAuditLogStatsByCountry(ctx, since)
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

func TestRepository_ListAuditLogStatsByCountry_Empty(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)

	since := time.Now().UTC().Add(-1 * time.Hour)
	stats, err := repos.queries.ListAuditLogStatsByCountry(t.Context(), since)
	is.NoErr(err)
	is.Equal(len(stats), 0)
}

func TestRepository_ListAuditLogStatsByCountry_SinceFilter(t *testing.T) {
	is := is.New(t)
	repos := setupRepos(t)
	ctx := t.Context()

	// Insert an old event (created 2 hours ago).
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	err := repos.audit.BatchInsert(ctx, []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: oldTime,
			Headers:   map[string][]string{},
			GeoIP:     geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA"},
		},
	})
	is.NoErr(err)

	// Query with since = 1 hour ago — should exclude the old event.
	since := time.Now().UTC().Add(-1 * time.Hour)
	stats, err := repos.queries.ListAuditLogStatsByCountry(ctx, since)
	is.NoErr(err)
	is.Equal(len(stats), 0)
}

func ptrDenyReason(r policy.DenyReason) *policy.DenyReason {
	return &r
}
