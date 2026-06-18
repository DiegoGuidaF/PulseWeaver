//go:build test

package rollup_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/matryer/is"
)

func newTestJob(repo *rollup.Repository) *rollup.RollupJob {
	return repo.NewRollupJob(slog.New(slog.DiscardHandler))
}

func countBuckets(t *testing.T, db *database.DB) int {
	t.Helper()
	var n int
	if err := db.GetContext(t.Context(), &n,
		`SELECT COUNT(DISTINCT bucket_at) FROM hourly_traffic_aggregates`); err != nil {
		t.Fatalf("count buckets: %v", err)
	}
	return n
}

func countAttributionRows(t *testing.T, db *database.DB, kind rollup.AttributionKind) int {
	t.Helper()
	var n int
	if err := db.GetContext(t.Context(), &n,
		`SELECT COUNT(*) FROM hourly_attribution_aggregates WHERE entity_kind = ?`, string(kind)); err != nil {
		t.Fatalf("count attribution rows: %v", err)
	}
	return n
}

// TestRollupJob_CatchUp_PopulatesAttributionAggregates: the per-entity
// attribution aggregates ride the same catch-up pass, so attributed rows seeded
// across missed hours are all rolled up by one job run.
func TestRollupJob_CatchUp_PopulatesAttributionAggregates(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	currentHour := time.Now().UTC().Truncate(time.Hour)
	policyA := int64(1)
	seedNetworkPolicy(t, db, policyA, "policy-a", "10.0.0.0/8")
	for hoursAgo := 1; hoursAgo <= 3; hoursAgo++ {
		hour := currentHour.Add(-time.Duration(hoursAgo) * time.Hour)
		seedPolicyAccessLogRow(t, db, "10.0.0.1", &policyA, "policy-a", true, hour.Add(5*time.Minute))
	}

	is.NoErr(newTestJob(repo).Run(ctx))

	is.Equal(countAttributionRows(t, db, rollup.AttributionKindPolicy), 3) // one allow row per missed hour
}

// TestRollupJob_CatchUp_CoversAllMissedHours guards the core catch-up fix:
// hours that passed while no rollup ran (downtime, restarts, seeded data)
// must all be covered by a single job run, not just the previous hour.
func TestRollupJob_CatchUp_CoversAllMissedHours(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	currentHour := time.Now().UTC().Truncate(time.Hour)
	for hoursAgo := 1; hoursAgo <= 3; hoursAgo++ {
		hour := currentHour.Add(-time.Duration(hoursAgo) * time.Hour)
		seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(5*time.Minute))
	}

	is.NoErr(newTestJob(repo).Run(ctx))

	is.Equal(countBuckets(t, db), 3) // one bucket per missed hour, not just the previous one
}

// TestRollupJob_CatchUp_ResumesAfterLastRolledBucket verifies the job rolls
// forward from the last bucket present in the DB and does not re-roll hours
// already covered.
func TestRollupJob_CatchUp_ResumesAfterLastRolledBucket(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	currentHour := time.Now().UTC().Truncate(time.Hour)
	rolledHour := currentHour.Add(-3 * time.Hour)
	missedHour1 := currentHour.Add(-2 * time.Hour)
	missedHour2 := currentHour.Add(-time.Hour)

	// rolledHour is already aggregated with a sentinel count that disagrees with
	// its raw rows: a re-roll would overwrite it back to 1.
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", rolledHour.Add(5*time.Minute))
	seedAggregateRow(t, db, rolledHour, "10.0.0.1", "app.example.com", true, 99)
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", missedHour1.Add(5*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", missedHour2.Add(5*time.Minute))

	is.NoErr(newTestJob(repo).Run(ctx))

	is.Equal(countBuckets(t, db), 3)

	var sentinel int64
	is.NoErr(db.GetContext(ctx, &sentinel,
		`SELECT request_count FROM hourly_traffic_aggregates WHERE bucket_at = ?`,
		rolledHour.Format("2006-01-02 15:04:05+00:00")))
	is.Equal(sentinel, int64(99)) // already-rolled hour untouched
}

func TestRollupJob_EmptyAccessLog_NoOp(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	job := newTestJob(repo)
	is.NoErr(job.Run(ctx))
	is.NoErr(job.Run(ctx)) // second run exercises the in-memory guard

	is.Equal(countBuckets(t, db), 0)
}

// TestRollupJob_InFlightHourExcluded: rows in the current, incomplete hour must
// not be rolled up — the raw path serves them until the hour completes.
func TestRollupJob_InFlightHourExcluded(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	prevHour := now.Truncate(time.Hour).Add(-time.Hour)
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", prevHour.Add(5*time.Minute))
	// 30m ahead of now: always inside an hour that is still incomplete when the
	// job runs, even if the wall-clock hour flips between seeding and running.
	seedAccessLogRow(t, db, "10.0.0.2", "app.example.com", true, "", now.Add(30*time.Minute))

	is.NoErr(newTestJob(repo).Run(ctx))

	is.Equal(countBuckets(t, db), 1) // only the complete previous hour
}
