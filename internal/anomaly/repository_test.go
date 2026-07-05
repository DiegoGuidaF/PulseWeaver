//go:build test

package anomaly

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/matryer/is"
)

// seedAnomaly inserts one row into the anomalies table and returns its id.
func seedAnomaly(t *testing.T, db *database.DB, kind, status string, lastSeen time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowxContext(t.Context(),
		`INSERT INTO anomalies (kind, severity, status, fingerprint, first_seen_at, last_seen_at, evidence_json)
		 VALUES (?, 'warning', ?, ?, ?, ?, '{}') RETURNING id`,
		kind, status, kind+":"+lastSeen.Format(time.RFC3339Nano), lastSeen.UTC(), lastSeen.UTC(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("seed anomaly: %v", err)
	}
	return id
}

func anomalyStatus(t *testing.T, db *database.DB, id int64) string {
	t.Helper()
	var status string
	if err := db.GetContext(t.Context(), &status, `SELECT status FROM anomalies WHERE id = ?`, id); err != nil {
		t.Fatalf("read anomaly status: %v", err)
	}
	return status
}

func TestRepository_Acknowledge_FlipsAndIsIdempotent(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	id := seedAnomaly(t, db, "deny_spike", "open", time.Now())

	is.NoErr(repo.Acknowledge(context.Background(), id))
	is.Equal(anomalyStatus(t, db, id), "acknowledged")

	// Re-acknowledging an already-acknowledged row still succeeds.
	is.NoErr(repo.Acknowledge(context.Background(), id))
	is.Equal(anomalyStatus(t, db, id), "acknowledged")
}

func TestRepository_Acknowledge_UnknownID_NotFound(t *testing.T) {
	is := is.New(t)
	repo, _ := newRepo(t)

	err := repo.Acknowledge(context.Background(), 424242)
	is.True(errors.Is(err, ErrNotFound))
}

// TestRepository_DeleteAnomaliesOlderThan prunes stale rows regardless of status
// while leaving recent ones and device_profiles untouched.
func TestRepository_DeleteAnomaliesOlderThan(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()

	stale := seedAnomaly(t, db, "deny_spike", "open", now.Add(-40*24*time.Hour))
	staleAck := seedAnomaly(t, db, "invalid_token", "acknowledged", now.Add(-40*24*time.Hour))
	recent := seedAnomaly(t, db, "geo_denied", "open", now.Add(-1*24*time.Hour))

	// A learned profile must survive pruning — deleting it would re-flag the value.
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimUserAgent, "fp-1", now.Add(-90*24*time.Hour))

	cutoff := now.Add(-30 * 24 * time.Hour)
	deleted, err := repo.DeleteAnomaliesOlderThan(context.Background(), cutoff)
	is.NoErr(err)
	is.Equal(deleted, int64(2)) // both stale rows, open and acknowledged

	is.Equal(countRows(t, db, "anomalies", "id = ?", stale), 0)
	is.Equal(countRows(t, db, "anomalies", "id = ?", staleAck), 0)
	is.Equal(countRows(t, db, "anomalies", "id = ?", recent), 1)
	is.Equal(countRows(t, db, "device_profiles", "device_id = ?", 1), 1)
}

func countRows(t *testing.T, db *database.DB, table, where string, arg any) int {
	t.Helper()
	var n int
	if err := db.GetContext(t.Context(), &n,
		"SELECT COUNT(*) FROM "+table+" WHERE "+where, arg); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
