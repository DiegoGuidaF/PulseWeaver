//go:build test

package dashboard_test

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/jmoiron/sqlx"
	"github.com/matryer/is"
)

func setupTestRepo(t *testing.T) (*dashboard.Repository, *sqlx.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	repo := dashboard.NewRepository(db.DB())
	return repo, db.DB()
}

// seedAccessLogRow inserts a single row into access_log for testing.
func seedAccessLogRow(t *testing.T, db *sqlx.DB, clientIP string, targetHost string, outcome bool, denyReason string, createdAt time.Time) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	_, err := db.Exec(`
		INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, ?, ?, ?, '{}')
	`, clientIP, targetHost, outcomeInt, denyReason, createdAt.UTC())
	if err != nil {
		t.Fatalf("seed access row: %v", err)
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

	is.NoErr(repo.RunRollup(ctx, from, to))

	buckets, err := repo.GetTrafficSeries(ctx, from, to, timebucket.GranularityHour)
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

	day1Hour1 := time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC)
	day1Hour2 := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	day2Hour1 := time.Date(2025, 3, 16, 8, 0, 0, 0, time.UTC)
	from := day1Hour1
	to := day2Hour1.Add(time.Hour)

	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", day1Hour1.Add(5*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", true, "", day1Hour2.Add(5*time.Minute))
	seedAccessLogRow(t, db, "10.0.0.1", "app.example.com", false, "ip_not_registered", day2Hour1.Add(5*time.Minute))

	is.NoErr(repo.RunRollup(ctx, from, to))

	buckets, err := repo.GetTrafficSeries(ctx, from, to, timebucket.GranularityDay)
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

	buckets, err := repo.GetTrafficSeries(ctx, from, to, timebucket.GranularityHour)
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

	is.NoErr(repo.RunRollup(ctx, from, to))

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

	is.NoErr(repo.RunRollup(ctx, from, to))

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

	is.NoErr(repo.RunRollup(ctx, from, to))

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

	// Total denied count should still reflect all 3
	stats, err := repo.GetSummaryStats(ctx, from, to)
	is.NoErr(err)
	is.Equal(stats.DeniedCount, int64(3))

	// Top denied IPs should aggregate across deny reasons
	ips, err := repo.GetTopDeniedIPs(ctx, from, to, 10)
	is.NoErr(err)
	is.Equal(len(ips), 1)
	is.Equal(ips[0].Count, int64(3)) // all deny reasons summed for same IP
}
