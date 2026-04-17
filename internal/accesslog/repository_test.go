//go:build test

package accesslog_test

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupTestRepo(t *testing.T) *accesslog.Repository {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return accesslog.NewRepository(db.DB())
}

func TestRepository_BatchInsert_EmptyBatch(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)

	err := repo.BatchInsert(context.Background(), nil)
	is.NoErr(err)

	err = repo.BatchInsert(context.Background(), []policy.DecisionEvent{})
	is.NoErr(err)
}

func TestRepository_BatchInsert_AllowEvent(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{
			ClientIP:   "1.2.3.4",
			Outcome:    true,
			DenyReason: nil,
			DeviceID:   nil,
			AddressID:  nil,
			CreatedAt:  time.Now().UTC(),
			TargetHost: new("example.com"),
			TargetURI:  new("/api"),
			HTTPMethod: new("GET"),
			XFFChain:   new("1.2.3.4"),
			Headers:    map[string][]string{"User-Agent": {"Mozilla/5.0"}},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Allow events must not appear as deny reasons.
	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}

func TestRepository_BatchInsert_DenyEvent(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{
			ClientIP:   "10.0.0.1",
			Outcome:    false,
			DenyReason: new(policy.DenyReasonIPNotRegistered),
			CreatedAt:  time.Now().UTC(),
			Headers:    map[string][]string{},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 1)
	is.Equal(reasons[0], string(policy.DenyReasonIPNotRegistered))
}

func TestRepository_BatchInsert_MultipleEvents(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: new(policy.DenyReasonNoDeviceMatch), CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "3.3.3.3", Outcome: true, DenyReason: nil, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Both deny reasons stored; allow event excluded.
	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 2)
}

func TestRepository_ListDenyReasons_Empty(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)

	reasons, err := repo.ListDenyReasons(context.Background())
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}

func TestRepository_ListDenyReasons_ReturnsSortedDistinct(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	r1 := policy.DenyReasonIPNotRegistered
	r2 := policy.DenyReasonNoDeviceMatch

	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // duplicate
		{ClientIP: "3.3.3.3", Outcome: false, DenyReason: &r2, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 2)
	is.Equal(reasons[0], string(r1))
	is.Equal(reasons[1], string(r2))
}

func TestRepository_ListDenyReasons_ExcludesAllowEvents(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: true, DenyReason: nil, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}

// GeoIP persistence tests

func TestRepository_BatchInsert_WithGeoIPData(t *testing.T) {
	is := is.New(t)

	dbWrapper, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	repo := accesslog.NewRepository(dbWrapper.DB())
	ctx := context.Background()

	// Insert event with full GeoIP data.
	events := []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
			GeoIP: geoip.Result{
				CountryCode:   "US",
				CountryName:   "United States",
				ContinentCode: "NA",
				ASN:           15169,
				ASNOrg:        "Google LLC",
			},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Verify the geoip row was persisted with correct values.
	var countryCode, countryName, continentCode, asnOrg string
	var asn int
	err = dbWrapper.DB().QueryRowxContext(t.Context(),
		`SELECT country_code, country_name, continent_code, asn, asn_org
		 FROM access_log_geoip LIMIT 1`,
	).Scan(&countryCode, &countryName, &continentCode, &asn, &asnOrg)
	is.NoErr(err)
	is.Equal(countryCode, "US")
	is.Equal(countryName, "United States")
	is.Equal(continentCode, "NA")
	is.Equal(asn, 15169)
	is.Equal(asnOrg, "Google LLC")
}

func TestRepository_BatchInsert_GeoIPCascadeDelete(t *testing.T) {
	is := is.New(t)

	dbWrapper, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	repo := accesslog.NewRepository(dbWrapper.DB())
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
			GeoIP: geoip.Result{
				CountryCode:   "US",
				CountryName:   "United States",
				ContinentCode: "NA",
				ASN:           15169,
				ASNOrg:        "Google LLC",
			},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Delete the access log row — geoip row should cascade.
	_, err = dbWrapper.DB().ExecContext(t.Context(), `DELETE FROM access_log`)
	is.NoErr(err)

	var count int
	err = dbWrapper.DB().QueryRowxContext(t.Context(), `SELECT COUNT(*) FROM access_log_geoip`).Scan(&count)
	is.NoErr(err)
	is.Equal(count, 0)
}

func TestRepository_BatchInsert_WithEmptyGeoIP(t *testing.T) {
	is := is.New(t)

	dbWrapper, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	repo := accesslog.NewRepository(dbWrapper.DB())
	ctx := context.Background()

	// Private IP — GeoIP.IsEmpty() == true, no geoip row should be written.
	events := []policy.DecisionEvent{
		{
			ClientIP:  "192.168.1.1",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Verify no geoip row exists.
	var count int
	err = dbWrapper.DB().QueryRowxContext(t.Context(), `SELECT COUNT(*) FROM access_log_geoip`).Scan(&count)
	is.NoErr(err)
	is.Equal(count, 0)
}

func TestRepository_BatchInsert_MixedGeoIP(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	// One event with GeoIP, one without — both access rows must be written.
	events := []policy.DecisionEvent{
		{
			ClientIP:  "8.8.8.8",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
			GeoIP:     geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 15169, ASNOrg: "Google LLC"},
		},
		{
			ClientIP:  "192.168.1.1",
			Outcome:   true,
			CreatedAt: time.Now().UTC(),
			Headers:   map[string][]string{},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Both access rows exist (deny reasons list is empty since all allowed).
	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}
