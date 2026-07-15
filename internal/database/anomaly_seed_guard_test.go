//go:build test

package database_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestSampleWorld_MaterializesShowcaseAnomalies guards the presentable seed: the
// sample world, run through the real detection pipeline, must produce a finding
// from every anomaly family so a local deploy or screenshot is never an empty
// list. It also locks the crafted scenarios (Priya's lapsed lease, Liam's foreign
// presence) against regressions in the detectors or the sample fixtures.
//
// Unlike make back-seed-db-sample (which writes a file), this runs in-memory under
// the normal test build so it executes in CI.
func TestSampleWorld_MaterializesShowcaseAnomalies(t *testing.T) {
	is := is.New(t)

	srv := testutils.SetupIntegrationServer(t)
	result := testutils.SeedSampleWorld(t).Build(srv)
	testutils.MaterializeSampleAnomalies(t, srv, result)

	counts := map[string]int{}
	rows, err := srv.Database.DB().QueryxContext(t.Context(),
		`SELECT kind, COUNT(*) FROM anomalies GROUP BY kind`)
	is.NoErr(err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kind string
		var n int
		is.NoErr(rows.Scan(&kind, &n))
		counts[kind] = n
	}
	is.NoErr(rows.Err())

	// Rules family.
	is.True(counts["expired_access"] >= 1) // Priya's lapsed lease + denied request
	is.True(counts["invalid_token"] >= 1)
	// Volume family.
	is.True(counts["host_probing"] >= 1) // Noah and the partial-access team devices
	is.True(counts["geo_denied"] >= 1)   // scanner traffic from unexpected countries
	// Novelty / geo-velocity family.
	is.True(counts["impossible_travel"] >= 1) // Liam's ThinkPad live in two countries
	is.True(counts["new_country"] >= 1)       // Liam's ThinkPad's novel German presence
}
