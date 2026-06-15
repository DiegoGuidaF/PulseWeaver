//go:build test

package database_test

import (
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestSeededAccessLogSurvivesRollupOnRestart is the PW-68 regression guard.
//
// It boots an app against a seed-generator DSN (seedDBDSN), writes access_log rows
// in the previous complete hour — the window the init rollup covers on restart —
// then runs the rollup the way app init does. Before the fix, those rows were
// stored in Go's time.Time.String() format ("… +0000 UTC"), strftime returned
// NULL, and the rollup hit a bucket_at NOT NULL violation that crash-looped the
// app. The test asserts the rollup succeeds and populates a bucket.
//
// It guards both fixes: the DSN time-format params (seedDBDSN) keep seeded
// timestamps parseable, and RunRollup's strftime-IS-NOT-NULL filter is the
// defence-in-depth backstop.
func TestSeededAccessLogSurvivesRollupOnRestart(t *testing.T) {
	is := is.New(t)

	dbPath := filepath.Join(t.TempDir(), "seed.db")
	conf := &config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: testutils.TestAdminPassword,
			TrustedProxy:  netip.MustParseAddr("127.0.0.1"),
		},
		DB:     config.ConfDB{Dsn: seedDBDSN(dbPath)},
		Rules:  config.ConfRules{CheckInterval: time.Minute},
		Policy: config.ConfPolicy{APISecret: testutils.TestPolicySecret},
		// GeoIP left zero-valued (Enabled=false) → no disk access.
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx := t.Context()

	application, err := app.NewWithConfigAndLogger(ctx, conf, logger)
	is.NoErr(err) // boot runs the init rollup once (window empty at this point)
	t.Cleanup(func() { _ = application.Close() })

	// Write access_log rows in the previous complete hour, through the configured
	// driver — exactly the path the seeder's BatchInsert takes.
	db := application.Database.DB()
	prevHour := time.Now().Truncate(time.Hour).Add(-time.Hour)
	for i := range 3 {
		_, err := db.ExecContext(ctx, `
			INSERT INTO access_log (client_ip, outcome, deny_reason, contributor_count, created_at, headers_json)
			VALUES (?, ?, ?, ?, ?, '{}')
		`, "100.64.0.1", 1, nil, 0, prevHour.Add(time.Duration(i)*time.Minute))
		is.NoErr(err)
	}

	// Re-run the rollup the way init does on restart: a fresh, uninitialised job
	// whose window is the previous complete hour.
	job := dashboard.NewRepository(db, nil).NewRollupJob(logger)
	is.NoErr(job.Run(ctx)) // before the fix: bucket_at NOT NULL constraint violation

	// The seeded rows must have produced a non-NULL bucket.
	var bucketCount int
	is.NoErr(db.GetContext(ctx, &bucketCount,
		`SELECT COUNT(*) FROM hourly_traffic_aggregates WHERE bucket_at IS NOT NULL`))
	is.Equal(bucketCount, 1)
}
