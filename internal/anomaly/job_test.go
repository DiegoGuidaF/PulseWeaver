//go:build test

package anomaly_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/anomaly"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func noopLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func setupTestRepo(t *testing.T) (*anomaly.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return anomaly.NewRepository(db.DB()), db.DB()
}

// allFamilies enables every family; interval 0 makes the job run on every call
// so tests can drive multiple passes back to back.
func allFamilies() anomaly.ScanOptions {
	return anomaly.ScanOptions{Interval: 0, Sensitivity: "medium", DetectRules: true, DetectVolume: true, DetectNovelty: true}
}

type fakeDetector struct {
	family   anomaly.Family
	findings []anomaly.Finding
	err      error
	calls    int
}

func (d *fakeDetector) Family() anomaly.Family { return d.family }

func (d *fakeDetector) Detect(_ context.Context, _ anomaly.Scope) ([]anomaly.Finding, error) {
	d.calls++
	return d.findings, d.err
}

var _ anomaly.Detector = (*fakeDetector)(nil)

func seedAccessLogRow(t *testing.T, db *database.DB, createdAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES ('10.0.0.1', 'app.example.com', 0, 'ip_not_registered', ?, '{}')
	`, createdAt.UTC())
	if err != nil {
		t.Fatalf("seed access row: %v", err)
	}
}

func countAnomalies(t *testing.T, db *database.DB, where string) int {
	t.Helper()
	var n int
	q := "SELECT COUNT(*) FROM anomalies"
	if where != "" {
		q += " WHERE " + where
	}
	if err := db.GetContext(t.Context(), &n, q); err != nil {
		t.Fatalf("count anomalies: %v", err)
	}
	return n
}

func scanWatermark(t *testing.T, db *database.DB) int64 {
	t.Helper()
	var id int64
	if err := db.GetContext(t.Context(), &id,
		`SELECT COALESCE((SELECT last_access_log_id FROM anomaly_scan_state WHERE id = 1), -1)`); err != nil {
		t.Fatalf("read watermark: %v", err)
	}
	return id
}

func finding(fingerprint string, evidence map[string]any, at time.Time) anomaly.Finding {
	return anomaly.Finding{
		Kind:        anomaly.KindInvalidToken,
		Severity:    anomaly.SeverityCritical,
		Fingerprint: fingerprint,
		Evidence:    evidence,
		ObservedAt:  at,
	}
}

func TestScanJob_UpsertsFindings(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	det := &fakeDetector{family: anomaly.FamilyRules, findings: []anomaly.Finding{
		finding("fp-a", map[string]any{"n": 1}, time.Now()),
		finding("fp-b", nil, time.Now()),
	}}
	job := anomaly.NewScanJob(repo, []anomaly.Detector{det}, allFamilies(), noopLogger())

	is.NoErr(job.Run(context.Background()))

	is.Equal(countAnomalies(t, db, ""), 2)
	is.Equal(countAnomalies(t, db, "status = 'open'"), 2)
}

// TestScanJob_ReEmitSameFingerprint_UpdatesInPlace: re-detecting the same open
// condition advances last_seen_at and evidence rather than inserting a new row.
func TestScanJob_ReEmitSameFingerprint_UpdatesInPlace(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)

	det := &fakeDetector{family: anomaly.FamilyRules, findings: []anomaly.Finding{finding("fp-x", map[string]any{"n": 1}, t1)}}
	job := anomaly.NewScanJob(repo, []anomaly.Detector{det}, allFamilies(), noopLogger())
	is.NoErr(job.Run(context.Background()))

	det.findings = []anomaly.Finding{finding("fp-x", map[string]any{"n": 2}, t2)}
	is.NoErr(job.Run(context.Background()))

	is.Equal(countAnomalies(t, db, ""), 1) // updated, not inserted

	var evidence string
	var lastSeen time.Time
	is.NoErr(db.QueryRowxContext(t.Context(),
		`SELECT evidence_json, last_seen_at FROM anomalies WHERE fingerprint = 'fp-x'`).Scan(&evidence, &lastSeen))
	is.Equal(evidence, `{"n":2}`)
	is.Equal(lastSeen.Unix(), t2.Unix())
}

// TestScanJob_AcknowledgedRow_ReEmitOpensNewRow: the open-row uniqueness is
// partial, so an acknowledged finding does not block a fresh recurrence.
func TestScanJob_AcknowledgedRow_ReEmitOpensNewRow(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	det := &fakeDetector{family: anomaly.FamilyRules, findings: []anomaly.Finding{finding("fp-dup", nil, time.Now())}}
	job := anomaly.NewScanJob(repo, []anomaly.Detector{det}, allFamilies(), noopLogger())

	is.NoErr(job.Run(context.Background()))
	_, err := db.ExecContext(t.Context(), `UPDATE anomalies SET status = 'acknowledged' WHERE fingerprint = 'fp-dup'`)
	is.NoErr(err)

	is.NoErr(job.Run(context.Background()))

	is.Equal(countAnomalies(t, db, ""), 2)                // acknowledged history + new open row
	is.Equal(countAnomalies(t, db, "status = 'open'"), 1) // exactly one open
	is.Equal(countAnomalies(t, db, "status = 'acknowledged'"), 1)
}

// TestScanJob_CleanRun_AdvancesWatermark: with no detector error the raw cursor
// advances to the snapshotted MAX(access_log.id).
func TestScanJob_CleanRun_AdvancesWatermark(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	for range 3 {
		seedAccessLogRow(t, db, time.Now())
	}
	det := &fakeDetector{family: anomaly.FamilyRules}
	job := anomaly.NewScanJob(repo, []anomaly.Detector{det}, allFamilies(), noopLogger())

	is.NoErr(job.Run(context.Background()))

	is.Equal(scanWatermark(t, db), int64(3))
}

// TestScanJob_DetectorError_HoldsWatermark_OthersStillRun: one failing detector
// must not starve the others, and the raw watermark is held so its window is
// rescanned next pass (findings dedupe by fingerprint, so rescan is safe).
func TestScanJob_DetectorError_HoldsWatermark_OthersStillRun(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	for range 2 {
		seedAccessLogRow(t, db, time.Now())
	}
	failing := &fakeDetector{family: anomaly.FamilyRules, err: errors.New("boom")}
	clean := &fakeDetector{family: anomaly.FamilyVolume, findings: []anomaly.Finding{finding("fp-clean", nil, time.Now())}}
	job := anomaly.NewScanJob(repo, []anomaly.Detector{failing, clean}, allFamilies(), noopLogger())

	is.NoErr(job.Run(context.Background())) // a detector error is isolated, not returned

	is.Equal(clean.calls, 1)                 // clean detector still ran
	is.Equal(countAnomalies(t, db, ""), 1)   // its finding still persisted
	is.Equal(scanWatermark(t, db), int64(0)) // watermark held for the failed window
}

// TestScanJob_FamilyToggle_SkipsDisabledDetectors: a disabled family's detectors
// are never invoked.
func TestScanJob_FamilyToggle_SkipsDisabledDetectors(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	rules := &fakeDetector{family: anomaly.FamilyRules}
	novelty := &fakeDetector{family: anomaly.FamilyNovelty}
	opts := anomaly.ScanOptions{Interval: 0, Sensitivity: "medium", DetectRules: true, DetectVolume: true, DetectNovelty: false}
	job := anomaly.NewScanJob(repo, []anomaly.Detector{rules, novelty}, opts, noopLogger())

	is.NoErr(job.Run(context.Background()))

	is.Equal(rules.calls, 1)
	is.Equal(novelty.calls, 0)
}

// failingScanRepo wraps the real repository and injects an error from the
// Nth UpsertFinding call, so tests can drive a mid-scan failure through the
// same WithinTx path the job uses in production.
type failingScanRepo struct {
	*anomaly.Repository
	failOnUpsert int // 1-indexed; 0 disables injection
	upsertCalls  int
	err          error
}

func (r *failingScanRepo) UpsertFinding(ctx context.Context, f anomaly.Finding) error {
	r.upsertCalls++
	if r.failOnUpsert != 0 && r.upsertCalls == r.failOnUpsert {
		return r.err
	}
	return r.Repository.UpsertFinding(ctx, f)
}

var _ anomaly.ScanRepository = (*failingScanRepo)(nil)

// TestScanJob_UpsertFails_RollsBackFindingsAndHoldsWatermark: an error from a
// later UpsertFinding call inside the scan transaction must roll back the
// earlier upserts in the same pass and leave the watermark unmoved — proving
// SaveScanState really shares the tx rather than running after it.
func TestScanJob_UpsertFails_RollsBackFindingsAndHoldsWatermark(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	failing := &failingScanRepo{Repository: repo, failOnUpsert: 2, err: errors.New("boom")}
	for range 2 {
		seedAccessLogRow(t, db, time.Now())
	}
	det := &fakeDetector{family: anomaly.FamilyRules, findings: []anomaly.Finding{
		finding("fp-a", nil, time.Now()),
		finding("fp-b", nil, time.Now()),
	}}
	job := anomaly.NewScanJob(failing, []anomaly.Detector{det}, allFamilies(), noopLogger())

	err := job.Run(context.Background())

	is.True(errors.Is(err, failing.err))
	is.Equal(countAnomalies(t, db, ""), 0)    // both upserts rolled back, including the first
	is.Equal(scanWatermark(t, db), int64(-1)) // watermark never advanced
}

// TestScanJob_SelfGate_SkipsWithinInterval: the job runs the first tick, then
// skips subsequent ticks that fall inside the scan interval.
func TestScanJob_SelfGate_SkipsWithinInterval(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	det := &fakeDetector{family: anomaly.FamilyRules}
	opts := allFamilies()
	opts.Interval = time.Hour
	job := anomaly.NewScanJob(repo, []anomaly.Detector{det}, opts, noopLogger())

	is.NoErr(job.Run(context.Background()))
	is.NoErr(job.Run(context.Background())) // within the interval → skipped

	is.Equal(det.calls, 1)
}
