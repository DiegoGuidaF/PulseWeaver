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

// findingEvidence reads back one anomaly row's evidence and last_seen_at.
func findingEvidence(t *testing.T, db *database.DB, fingerprint string) (evidence string, lastSeen time.Time) {
	t.Helper()
	err := db.QueryRowxContext(t.Context(),
		`SELECT evidence_json, last_seen_at FROM anomalies WHERE fingerprint = ?`, fingerprint).
		Scan(&evidence, &lastSeen)
	if err != nil {
		t.Fatalf("read finding evidence: %v", err)
	}
	return evidence, lastSeen
}

// TestRepository_UpsertFinding_KeepsLargerObserved verifies the worst-hour
// guard: a same-day spike smaller than the one already recorded must not
// overwrite it, but a larger one must.
func TestRepository_UpsertFinding_KeepsLargerObserved(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindDenySpike, Severity: SeverityWarning, Fingerprint: "fp-spike",
		Evidence: map[string]any{"observed": 500}, ObservedAt: at,
	}))

	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindDenySpike, Severity: SeverityWarning, Fingerprint: "fp-spike",
		Evidence: map[string]any{"observed": 30}, ObservedAt: at.Add(time.Hour),
	}))
	evidence, _ := findingEvidence(t, db, "fp-spike")
	is.Equal(evidence, `{"observed":500}`) // smaller spike does not overwrite the worst hour

	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindDenySpike, Severity: SeverityWarning, Fingerprint: "fp-spike",
		Evidence: map[string]any{"observed": 700}, ObservedAt: at.Add(2 * time.Hour),
	}))
	evidence, _ = findingEvidence(t, db, "fp-spike")
	is.Equal(evidence, `{"observed":700}`) // a genuinely larger spike replaces it
}

// TestRepository_UpsertFinding_NoObservedKey_LatestWins verifies that kinds
// without a numeric $.observed (rules, novelty, travel, probing) keep
// latest-wins semantics — their evidence legitimately advances over time.
func TestRepository_UpsertFinding_NoObservedKey_LatestWins(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindGeoDenied, Severity: SeverityWarning, Fingerprint: "fp-geo",
		Evidence: map[string]any{"deny_count": 3}, ObservedAt: at,
	}))
	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindGeoDenied, Severity: SeverityWarning, Fingerprint: "fp-geo",
		Evidence: map[string]any{"deny_count": 9}, ObservedAt: at.Add(time.Hour),
	}))

	evidence, _ := findingEvidence(t, db, "fp-geo")
	is.Equal(evidence, `{"deny_count":9}`)
}

// TestRepository_UpsertFinding_LastSeenAt_DoesNotRewind mirrors the rewind
// guard in UpsertDeviceProfile: a rescan after a held watermark can revisit
// an older ObservedAt, and last_seen_at must never move backwards.
func TestRepository_UpsertFinding_LastSeenAt_DoesNotRewind(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	later := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	earlier := later.Add(-2 * time.Hour)

	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindInvalidToken, Severity: SeverityCritical, Fingerprint: "fp-rewind",
		Evidence: map[string]any{"n": 1}, ObservedAt: later,
	}))
	is.NoErr(repo.UpsertFinding(context.Background(), Finding{
		Kind: KindInvalidToken, Severity: SeverityCritical, Fingerprint: "fp-rewind",
		Evidence: map[string]any{"n": 2}, ObservedAt: earlier,
	}))

	_, lastSeen := findingEvidence(t, db, "fp-rewind")
	is.Equal(lastSeen.Unix(), later.Unix())
}
