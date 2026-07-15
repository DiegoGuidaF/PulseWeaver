//go:build test

package anomaly

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

func bucketScope(from, to time.Time) Scope {
	f := from
	return Scope{FromBucket: &f, ToBucket: to, Now: to, Sensitivity: "medium"}
}

type fakeGeo struct{ byIP map[string]geoip.Result }

func (f fakeGeo) Resolve(ip string) geoip.Result { return f.byIP[ip] }

func TestDenySpikeDetector_Spike_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	for i := 3; i <= 28; i++ { // 26 history buckets, all before observedFrom
		seedTrafficAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "", false, 2, "")
	}
	seedTrafficAgg(t, db, to.Add(-time.Hour), "", false, 100, "") // observed spike

	findings, err := denySpikeDetector{reader: repo}.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindDenySpike)
	is.Equal(findings[0].Severity, SeverityWarning)
	is.Equal(findings[0].Evidence["observed"], int64(100))
	is.Equal(findings[0].Evidence["baseline"], int64(2))
	is.Equal(findings[0].Evidence["threshold"], int64(20))
}

func TestDenySpikeDetector_QuietSeries_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	for i := 3; i <= 28; i++ {
		seedTrafficAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "", false, 2, "")
	}
	seedTrafficAgg(t, db, to.Add(-time.Hour), "", false, 3, "") // within threshold

	findings, err := denySpikeDetector{reader: repo}.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

// TestDenySpikeDetector_HostAllowSpike_InfoSeverity: a per-host allow spike is
// the same math but info severity — legit-load-or-compromise, not a scan.
func TestDenySpikeDetector_HostAllowSpike_InfoSeverity(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	for i := 3; i <= 28; i++ {
		seedTrafficAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "app.example.com", true, 5, "")
	}
	seedTrafficAgg(t, db, to.Add(-time.Hour), "app.example.com", true, 500, "") // allow spike

	findings, err := denySpikeDetector{reader: repo}.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Severity, SeverityInfo)
	is.Equal(findings[0].Evidence["outcome"], "allow")
	is.Equal(*findings[0].TargetHost, "app.example.com")
}

func TestEntityDriftDetector_Spike_FlagsAttributed(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	deviceID := int64(7)
	for i := 3; i <= 28; i++ {
		seedAttrAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "device", &deviceID, "laptop", 2)
	}
	seedAttrAgg(t, db, to.Add(-time.Hour), "device", &deviceID, "laptop", 100)

	findings, err := entityDriftDetector{reader: repo}.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindEntityDrift)
	is.Equal(*findings[0].DeviceID, ids.DeviceID(deviceID))
	is.Equal(findings[0].DeviceName, "laptop")
}

// TestEntityDriftDetector_NewEntity_Silenced: an entity with less than the
// minimum history never flags — its first day is not "drift".
func TestEntityDriftDetector_NewEntity_Silenced(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	userID := int64(3)
	for i := 3; i <= 7; i++ { // only 5 history buckets (< minHistory)
		seedAttrAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "user", &userID, "newbie", 2)
	}
	seedAttrAgg(t, db, to.Add(-time.Hour), "user", &userID, "newbie", 500)

	findings, err := entityDriftDetector{reader: repo}.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

func TestGeoDeniedDetector_OutsideExpectedSet_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "10.0.0.1", true, to.Add(-48*time.Hour)) // enabled → expected country
	seedTrafficAgg(t, db, to.Add(-time.Hour), "app.example.com", false, 30, "RU")

	geo := fakeGeo{byIP: map[string]geoip.Result{"10.0.0.1": {CountryCode: "US"}}}
	det := geoDeniedDetector{reader: repo, geo: geo}

	findings, err := det.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindGeoDenied)
	is.Equal(*findings[0].CountryCode, "RU")
	is.Equal(findings[0].Evidence["deny_count"], int64(30))
}

func TestGeoDeniedDetector_InsideExpectedSet_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "10.0.0.1", true, to.Add(-48*time.Hour))
	seedTrafficAgg(t, db, to.Add(-time.Hour), "app.example.com", false, 30, "US")

	geo := fakeGeo{byIP: map[string]geoip.Result{"10.0.0.1": {CountryCode: "US"}}}
	det := geoDeniedDetector{reader: repo, geo: geo}

	findings, err := det.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

func TestGeoDeniedDetector_NilResolver_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	seedTrafficAgg(t, db, to.Add(-time.Hour), "app.example.com", false, 30, "RU")

	det := geoDeniedDetector{reader: repo, geo: nil}

	findings, err := det.Detect(context.Background(), bucketScope(to.Add(-2*time.Hour), to))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

// TestScanJob_VolumeSpike_AdvancesCursorAndDedupes runs the full job over a
// seeded spike: the first pass records one anomaly and advances the bucket
// cursor; a second pass re-evaluates and dedupes into the same open row.
func TestScanJob_VolumeSpike_AdvancesCursorAndDedupes(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	for i := 25; i <= 50; i++ { // history before the first-run observedFrom (now-24h)
		seedTrafficAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "", false, 2, "")
	}
	seedTrafficAgg(t, db, to.Add(-time.Hour), "", false, 100, "") // spike inside last 24h

	opts := ScanOptions{Interval: 0, Sensitivity: "medium", DetectRules: true, DetectVolume: true, DetectNovelty: true}
	job := NewScanJob(repo, AllDetectors(repo, nil), opts, slog.New(slog.DiscardHandler))

	is.NoErr(job.Run(context.Background()))
	is.Equal(countAllAnomalies(t, db), 1)
	is.True(bucketCursorSet(t, db))

	is.NoErr(job.Run(context.Background()))
	is.Equal(countAllAnomalies(t, db), 1) // idempotent re-run
}

func countAllAnomalies(t *testing.T, db *database.DB) int {
	t.Helper()
	var n int
	if err := db.GetContext(t.Context(), &n, `SELECT COUNT(*) FROM anomalies`); err != nil {
		t.Fatalf("count anomalies: %v", err)
	}
	return n
}

func bucketCursorSet(t *testing.T, db *database.DB) bool {
	t.Helper()
	var n int
	if err := db.GetContext(t.Context(), &n,
		`SELECT COUNT(*) FROM anomaly_scan_state WHERE id = 1 AND last_bucket_at IS NOT NULL`); err != nil {
		t.Fatalf("read bucket cursor: %v", err)
	}
	return n == 1
}

func lastBucketAt(t *testing.T, db *database.DB) time.Time {
	t.Helper()
	var bucket time.Time
	if err := db.GetContext(t.Context(), &bucket,
		`SELECT last_bucket_at FROM anomaly_scan_state WHERE id = 1`); err != nil {
		t.Fatalf("read last bucket at: %v", err)
	}
	return bucket
}

// TestScanJob_BucketCursor_ClampsToRollupProgress: rollup runs before the
// anomaly scan in the same scheduler tick but on its own clock. If the hour
// boundary falls between the two, rollup may not have built the aggregate for
// the hour wall-clock now considers complete. Advancing the bucket cursor past
// that gap would mark the still-empty bucket "observed"; once rollup catches up
// and populates it, splitSeries would classify it as history instead, and any
// spike inside it is never evaluated. The cursor must clamp to rollup's actual
// progress (MAX(bucket_at)+1h), not outrun it.
func TestScanJob_BucketCursor_ClampsToRollupProgress(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	to := time.Now().Truncate(time.Hour)
	for i := 2; i <= 27; i++ { // 26 quiet history buckets ending at to-2h; to-1h intentionally not seeded (rollup lag)
		seedTrafficAgg(t, db, to.Add(-time.Duration(i)*time.Hour), "", false, 2, "")
	}

	opts := ScanOptions{Interval: 0, Sensitivity: "medium", DetectRules: true, DetectVolume: true, DetectNovelty: true}
	job := NewScanJob(repo, AllDetectors(repo, nil), opts, slog.New(slog.DiscardHandler))

	is.NoErr(job.Run(context.Background()))
	is.Equal(lastBucketAt(t, db).Unix(), to.Add(-time.Hour).Unix()) // clamped to MAX(bucket_at)+1h, not wall-clock `to`

	seedTrafficAgg(t, db, to.Add(-time.Hour), "", false, 100, "") // rollup caught up: spike bucket now populated
	is.NoErr(job.Run(context.Background()))
	is.Equal(countAllAnomalies(t, db), 1)
}
