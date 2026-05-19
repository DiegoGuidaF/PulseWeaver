//go:build test

package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupTestRepo(t *testing.T) (*dashboard.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	repo := dashboard.NewRepository(db.DB())
	return repo, db.DB()
}

// seedAccessLogRow inserts a single row into access_log for testing.
func seedAccessLogRow(t *testing.T, db *database.DB, clientIP string, targetHost string, outcome bool, denyReason string, createdAt time.Time) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, ?, ?, ?, '{}')
	`, clientIP, targetHost, outcomeInt, denyReason, createdAt.UTC())
	if err != nil {
		t.Fatalf("seed access row: %v", err)
	}
}

// seedAggregateRow inserts a pre-computed aggregate row directly into hourly_traffic_aggregates.
// Used to set up the long-range (> 24h) query path without needing RunRollup.
func seedAggregateRow(t *testing.T, db *database.DB, bucketAt time.Time, clientIP string, targetHost string, outcome bool, count int64) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	bucketStr := bucketAt.UTC().Truncate(time.Hour).Format("2006-01-02 15:04:05+00:00")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO hourly_traffic_aggregates
			(bucket_at, client_ip, target_host, outcome, deny_reason, request_count, sum_duration_us, max_duration_us)
		VALUES (?, ?, ?, ?, '', ?, 0, 0)
	`, bucketStr, clientIP, targetHost, outcomeInt, count)
	if err != nil {
		t.Fatalf("seed aggregate row: %v", err)
	}
}

// --- RunRollup ---

func TestRunRollup_EmptyAccessLog_NoAggregates(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)

	err := repo.RunRollup(ctx, from, to)
	is.NoErr(err)

	// 1h window → raw path; access_log is empty so total must be 0.
	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.TotalRequests, int64(0))
}

func TestRunRollup_SingleHour_AggregatesCorrectly(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	// 3 allow + 2 deny from same IP, 1 deny from different IP
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(5*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(10*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(15*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(20*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(25*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.2", "api.example.com", false, "no_device_match", hour.Add(30*time.Minute))

	err := repo.RunRollup(ctx, from, to)
	is.NoErr(err)

	// 1h window → raw path reads directly from access_log.
	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.TotalRequests, int64(6))
	is.Equal(stats.AllowedCount, int64(3))
	is.Equal(stats.DeniedCount, int64(3))
	is.Equal(stats.UniqueIPs, int64(2))
}

func TestRunRollup_Idempotent(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(5*time.Minute))

	is.NoErr(repo.RunRollup(ctx, from, to))
	is.NoErr(repo.RunRollup(ctx, from, to)) // second run should not duplicate

	// 1h window → raw path; access_log has exactly 1 row.
	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.TotalRequests, int64(1))
}

// --- GetTrafficSeries ---

func TestGetTrafficSeries_MultiHour(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour1 := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	hour2 := time.Date(2025, 3, 15, 15, 0, 0, 0, time.UTC)
	from := hour1
	to := hour2.Add(time.Hour)

	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour1.Add(5*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour1.Add(10*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.2", "app.example.com", true, "", hour2.Add(5*time.Minute))

	// 2h window → raw path, auto-selects hour granularity.
	buckets, err := repo.GetTrafficSeries(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(buckets), 2)
	is.Equal(buckets[0].AllowCount, int64(1))
	is.Equal(buckets[0].DenyCount, int64(1))
	is.Equal(buckets[1].AllowCount, int64(1))
	is.Equal(buckets[1].DenyCount, int64(0))
}

func TestGetTrafficSeries_DayGranularity(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	day1 := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2025, 3, 16, 0, 0, 0, 0, time.UTC)
	from := day1
	to := day1.Add(15 * 24 * time.Hour) // 15d window → aggregate path, auto-selects day granularity

	seedAggregateRow(t, db, day1.Add(10*time.Hour), "10.0.0.1", "app.example.com", true, 2)
	seedAggregateRow(t, db, day2.Add(8*time.Hour), "10.0.0.1", "app.example.com", false, 1)

	buckets, err := repo.GetTrafficSeries(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(buckets), 2) // two days collapsed
	is.Equal(buckets[0].AllowCount, int64(2))
	is.Equal(buckets[0].DenyCount, int64(0))
	is.Equal(buckets[1].AllowCount, int64(0))
	is.Equal(buckets[1].DenyCount, int64(1))
}

func TestGetTrafficSeries_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	buckets, err := repo.GetTrafficSeries(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(buckets), 0)
}

// --- GetTopDeniedIPs ---

func TestGetTopDeniedIPs(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	// 10.0.0.2 has 3 denies, 10.0.0.1 has 1 deny, 10.0.0.3 has 0 denies (allow only)
	seedAccessLogRow(t, db, "10.0.0.2", "app.example.com", false, "ip_not_registered", hour.Add(1*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.2", "app.example.com", false, "ip_not_registered", hour.Add(2*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.2", "app.example.com", false, "no_device_match", hour.Add(3*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(4*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.3", "app.example.com", true, "", hour.Add(5*time.Minute))

	// 1h window → raw path.
	ips, err := repo.GetTopDeniedIPs(ctx, from, to, 10)
	is.NoErr(err)
	is.Equal(len(ips), 2)           // only denied IPs
	is.Equal(ips[0].IP, "10.0.0.2") // highest count first
	is.Equal(ips[0].Count, int64(3))
	is.Equal(ips[1].IP, "10.0.0.1")
	is.Equal(ips[1].Count, int64(1))
}

func TestGetTopDeniedIPs_RespectsLimit(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(1*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.2", "app.example.com", false, "ip_not_registered", hour.Add(2*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.3", "app.example.com", false, "ip_not_registered", hour.Add(3*time.Minute))

	ips, err := repo.GetTopDeniedIPs(ctx, from, to, 2)
	is.NoErr(err)
	is.Equal(len(ips), 2)
}

func TestGetTopDeniedIPs_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	ips, err := repo.GetTopDeniedIPs(ctx, from, to, 10)
	is.NoErr(err)
	is.Equal(len(ips), 0)
}

// --- GetServiceSplit ---

func TestGetServiceSplit(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(1*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", hour.Add(2*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(3*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "api.example.com", true, "", hour.Add(4*time.Minute))

	// 1h window → raw path.
	services, err := repo.GetServiceSplit(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(services), 2)
	// app.example.com has 3 total (2 allow + 1 deny), should be first
	is.Equal(services[0].Host, "app.example.com")
	is.Equal(services[0].AllowCount, int64(2))
	is.Equal(services[0].DenyCount, int64(1))
	is.Equal(services[1].Host, "api.example.com")
	is.Equal(services[1].AllowCount, int64(1))
	is.Equal(services[1].DenyCount, int64(0))
}

func TestGetServiceSplit_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	services, err := repo.GetServiceSplit(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(services), 0)
}

// --- DenyReason grouping ---

func TestRunRollup_DenyReasonGrouping(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	// Same IP, same host, same outcome=deny, different deny_reasons → separate aggregate rows
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(1*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", hour.Add(2*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "no_device_match", hour.Add(3*time.Minute))

	is.NoErr(repo.RunRollup(ctx, from, to))

	// 1h window → raw path; total denied count should reflect all 3 rows.
	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.DeniedCount, int64(3))

	// Top denied IPs should aggregate across deny reasons.
	ips, err := repo.GetTopDeniedIPs(ctx, from, to, 10)
	is.NoErr(err)
	is.Equal(len(ips), 1)
	is.Equal(ips[0].Count, int64(3)) // all deny reasons summed for same IP
}

// --- Dispatch: raw vs aggregate path ---

// TestGetSummaryStats_ShortRange_UsesRaw verifies that a short window reads directly
// from access_log without requiring RunRollup.
func TestGetSummaryStats_ShortRange_UsesRaw(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(6 * time.Hour) // 6h window → raw path

	seedAccessLogRow(t, db, "10.0.0.1", "svc.example.com", true, "", base.Add(10*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.2", "svc.example.com", false, "ip_not_registered", base.Add(20*time.Minute))

	// No RunRollup call — raw path should return correct results.
	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.TotalRequests, int64(2))
	is.Equal(stats.AllowedCount, int64(1))
	is.Equal(stats.DeniedCount, int64(1))
	is.Equal(stats.UniqueIPs, int64(2))
}

// TestGetSummaryStats_LongRange_UsesAggregates verifies that a long window reads from
// hourly_traffic_aggregates and ignores access_log rows outside the rolled-up data.
func TestGetSummaryStats_LongRange_UsesAggregates(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	// Seed access_log with sentinel IP "10.1.1.1" — should NOT appear in results.
	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	seedAccessLogRow(t, db, "10.1.1.1", "raw.example.com", true, "", base.Add(5*time.Minute))

	// Seed aggregates with sentinel IP "10.2.2.2" — SHOULD appear in results.
	seedAggregateRow(t, db, base, "10.2.2.2", "agg.example.com", true, 5)

	from := base
	to := base.Add(48 * time.Hour) // 48h window → aggregate path

	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.TotalRequests, int64(5)) // aggregate count, not access_log count
	is.Equal(stats.UniqueIPs, int64(1))     // only 10.2.2.2, not 10.1.1.1
}

// TestDispatch_Boundary verifies the 24h threshold: exactly 24h uses raw, just over uses aggregates.
func TestDispatch_Boundary(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 4, 10, 0, 0, 0, 0, time.UTC)

	// Sentinel IP in access_log (raw path should return this).
	seedAccessLogRow(t, db, "10.1.0.1", "raw.example.com", true, "", base.Add(30*time.Minute))

	// Sentinel IP in aggregates (aggregate path should return this).
	seedAggregateRow(t, db, base, "10.2.0.1", "agg.example.com", true, 7)

	exactThreshold := base.Add(24 * time.Hour)
	overThreshold := base.Add(24*time.Hour + time.Second)

	// Exactly 24h → raw path → returns access_log sentinel.
	stats24h, err := repo.GetSummaryStats(ctx, base, exactThreshold)
	is.NoErr(err)
	is.Equal(stats24h.UniqueIPs, int64(1)) // only 10.1.0.1

	// 24h + 1s → aggregate path → returns aggregate sentinel.
	statsOver, err := repo.GetSummaryStats(ctx, base, overThreshold)
	is.NoErr(err)
	is.Equal(statsOver.TotalRequests, int64(7)) // aggregate count
	is.Equal(statsOver.UniqueIPs, int64(1))     // only 10.2.0.1
}

// --- Fine-grained granularities (raw path) ---

// TestGetTrafficSeries_MinuteGranularity verifies that minute-level buckets are produced
// from access_log for short windows.
func TestGetTrafficSeries_MinuteGranularity(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(5 * time.Minute)

	// Two rows in the same minute, one in the next minute.
	seedAccessLogRow(t, db, "10.0.0.1", "svc", true, "", base.Add(30*time.Second))
	seedAccessLogRow(t, db, "10.0.0.2", "svc", true, "", base.Add(45*time.Second))
	seedAccessLogRow(t, db, "10.0.0.3", "svc", false, "denied", base.Add(90*time.Second))

	// 5min window → auto-selects GranularityMinute.
	buckets, err := repo.GetTrafficSeries(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(buckets), 2) // minute 0 and minute 1
	is.Equal(buckets[0].AllowCount, int64(2))
	is.Equal(buckets[0].DenyCount, int64(0))
	is.Equal(buckets[1].AllowCount, int64(0))
	is.Equal(buckets[1].DenyCount, int64(1))
}

// TestGetTrafficSeries_5minGranularity verifies 5-minute-snapped buckets from access_log.
func TestGetTrafficSeries_5minGranularity(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC) // 10:00
	from := base
	to := base.Add(15 * time.Minute)

	// 10:00 window (bucket 10:00): one allow
	seedAccessLogRow(t, db, "10.0.0.1", "svc", true, "", base.Add(2*time.Minute))
	// 10:05 window (bucket 10:05): one allow, one deny
	seedAccessLogRow(t, db, "10.0.0.2", "svc", true, "", base.Add(6*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.3", "svc", false, "denied", base.Add(8*time.Minute))
	// 10:10 window (bucket 10:10): one deny
	seedAccessLogRow(t, db, "10.0.0.4", "svc", false, "denied", base.Add(11*time.Minute))

	// 15min window → auto-selects Granularity5Min.
	buckets, err := repo.GetTrafficSeries(ctx, from, to)
	is.NoErr(err)
	is.Equal(len(buckets), 3) // 10:00, 10:05, 10:10
	is.Equal(buckets[0].AllowCount, int64(1))
	is.Equal(buckets[0].DenyCount, int64(0))
	is.Equal(buckets[1].AllowCount, int64(1))
	is.Equal(buckets[1].DenyCount, int64(1))
	is.Equal(buckets[2].AllowCount, int64(0))
	is.Equal(buckets[2].DenyCount, int64(1))
}
