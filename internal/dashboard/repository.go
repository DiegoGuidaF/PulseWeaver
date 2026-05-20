package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

// rawThreshold is the maximum window size for which queries run directly against
// access_log. Windows wider than this use hourly_traffic_aggregates instead.
// The current in-flight hour is always absent from aggregates (rollup covers only
// complete hours), so any window ≤ 24h benefits from the raw path.
const rawThreshold = 24 * time.Hour

// Repository provides both read and write access to traffic aggregates.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db: db,
	}
}

// LastRollupAt returns the most recent bucket_at stored in hourly_traffic_aggregates,
// or a zero time.Time if the table is empty.
func (r *Repository) LastRollupAt(ctx context.Context) (time.Time, error) {
	var t database.DBTime
	const query = `SELECT MAX(bucket_at) FROM hourly_traffic_aggregates`
	if err := r.db.GetContext(ctx, &t, query); err != nil {
		return time.Time{}, fmt.Errorf("last rollup at: %w", err)
	}
	return t.UTC(), nil
}

// RunRollup aggregates access_log rows in [from, to) into hourly_traffic_aggregates.
// Idempotent via INSERT OR REPLACE on the unique index.
//
// The strftime output is concatenated with '+00:00' so that bucket_at stores
// values in the same format the driver produces for time.Time parameters
// ("2006-01-02 15:04:05+00:00"), keeping WHERE comparisons consistent.
func (r *Repository) RunRollup(ctx context.Context, from, to time.Time) error {
	const query = `
		INSERT OR REPLACE INTO hourly_traffic_aggregates
			(bucket_at, client_ip, target_host, outcome, deny_reason, request_count, sum_duration_us, max_duration_us)
		SELECT
			strftime('%Y-%m-%d %H:00:00', created_at) || '+00:00' AS bucket_at,
			client_ip,
			COALESCE(target_host, '')                              AS target_host,
			outcome,
			COALESCE(deny_reason, '')                              AS deny_reason,
			COUNT(*)                                               AS request_count,
			SUM(COALESCE(duration_us, 0))                          AS sum_duration_us,
			MAX(COALESCE(duration_us, 0))                          AS max_duration_us
		FROM access_log
		WHERE created_at >= ?
		  AND created_at <  ?
		GROUP BY bucket_at, client_ip, target_host, outcome, deny_reason
		`
	if _, err := r.db.ExecContext(ctx, query, from.UTC(), to.UTC()); err != nil {
		return fmt.Errorf("run rollup: %w", err)
	}
	return nil
}

// GetSummaryStats returns aggregate counts over the given time window.
// Uses access_log directly for windows ≤ 24h; hourly_traffic_aggregates for longer windows.
func (r *Repository) GetSummaryStats(ctx context.Context, from, to time.Time) (SummaryStats, error) {
	if to.Sub(from) <= rawThreshold {
		return r.getRawSummaryStats(ctx, from, to)
	}
	return r.getAggregateSummaryStats(ctx, from, to)
}

// GetTrafficSeries returns time-bucketed allow/deny counts.
// Granularity is chosen automatically from the window size (see granularityForWindow).
// Uses access_log directly for windows ≤ 24h; hourly_traffic_aggregates for longer windows.
func (r *Repository) GetTrafficSeries(ctx context.Context, from, to time.Time) ([]TrafficBucket, error) {
	window := to.Sub(from)
	granularity := granularityForWindow(window)
	if window <= rawThreshold {
		return r.getRawTrafficSeries(ctx, from, to, granularity)
	}
	return r.getAggregateTrafficSeries(ctx, from, to, granularity)
}

// granularityForWindow maps a query window to the appropriate bucket size.
// ≤5m → minute, ≤1h → 5min, ≤7d → hour, >7d → day.
func granularityForWindow(d time.Duration) timebucket.Granularity {
	switch {
	case d <= 5*time.Minute:
		return timebucket.GranularityMinute
	case d <= time.Hour:
		return timebucket.Granularity5Min
	case d <= 7*24*time.Hour:
		return timebucket.GranularityHour
	default:
		return timebucket.GranularityDay
	}
}

// GetTopDeniedIPs returns the top denied IPs by total denied request count.
// Uses access_log directly for windows ≤ 24h; hourly_traffic_aggregates for longer windows.
func (r *Repository) GetTopDeniedIPs(ctx context.Context, from, to time.Time, limit int) ([]IPCount, error) {
	if to.Sub(from) <= rawThreshold {
		return r.getRawTopDeniedIPs(ctx, from, to, limit)
	}
	return r.getAggregateTopDeniedIPs(ctx, from, to, limit)
}

// GetServiceSplit returns per-host allow/deny counts.
// Uses access_log directly for windows ≤ 24h; hourly_traffic_aggregates for longer windows.
func (r *Repository) GetServiceSplit(ctx context.Context, from, to time.Time) ([]ServiceCount, error) {
	if to.Sub(from) <= rawThreshold {
		return r.getRawServiceSplit(ctx, from, to)
	}
	return r.getAggregateServiceSplit(ctx, from, to)
}

// ── Raw path (access_log) ─────────────────────────────────────────────────────

func (r *Repository) getRawSummaryStats(ctx context.Context, from, to time.Time) (SummaryStats, error) {
	const query = `
	SELECT
		COALESCE(COUNT(*), 0)                                                            AS total_requests,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN 1 ELSE 0 END), 0)                        AS allowed_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN 1 ELSE 0 END), 0)                        AS denied_count,
		COUNT(DISTINCT client_ip)                                                         AS unique_ips,
		CASE WHEN COUNT(*) > 0
			THEN SUM(COALESCE(duration_us, 0)) / COUNT(*)
			ELSE 0
		END                                                                               AS avg_duration_us
	FROM access_log
	WHERE created_at >= ? AND created_at < ?
	`
	var stats SummaryStats
	if err := r.db.GetContext(ctx, &stats, query, from.UTC(), to.UTC()); err != nil {
		return SummaryStats{}, fmt.Errorf("get raw summary stats: %w", err)
	}
	return stats, nil
}

func (r *Repository) getRawTrafficSeries(ctx context.Context, from, to time.Time, granularity timebucket.Granularity) ([]TrafficBucket, error) {
	bucketExpr := granularity.BucketExpr("created_at")

	query := fmt.Sprintf(`
	SELECT
		%s                                                                     AS timestamp,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN 1 ELSE 0 END), 0)             AS allow_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN 1 ELSE 0 END), 0)             AS deny_count
	FROM access_log
	WHERE created_at >= ? AND created_at < ?
	GROUP BY timestamp
	ORDER BY timestamp
	`, bucketExpr)

	var buckets []TrafficBucket
	if err := r.db.SelectContext(ctx, &buckets, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("get raw traffic series: %w", err)
	}
	if buckets == nil {
		buckets = []TrafficBucket{}
	}
	return buckets, nil
}

func (r *Repository) getRawTopDeniedIPs(ctx context.Context, from, to time.Time, limit int) ([]IPCount, error) {
	const query = `
	SELECT
		client_ip              AS ip,
		COUNT(*)               AS count
	FROM access_log
	WHERE outcome = 0
	  AND created_at >= ? AND created_at < ?
	GROUP BY client_ip
	ORDER BY count DESC
	LIMIT ?
	`
	var ips []IPCount
	if err := r.db.SelectContext(ctx, &ips, query, from.UTC(), to.UTC(), limit); err != nil {
		return nil, fmt.Errorf("get raw top denied ips: %w", err)
	}
	if ips == nil {
		ips = []IPCount{}
	}
	return ips, nil
}

func (r *Repository) getRawServiceSplit(ctx context.Context, from, to time.Time) ([]ServiceCount, error) {
	const query = `
	SELECT
		COALESCE(target_host, '')                                              AS host,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN 1 ELSE 0 END), 0)             AS allow_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN 1 ELSE 0 END), 0)             AS deny_count
	FROM access_log
	WHERE created_at >= ? AND created_at < ?
	GROUP BY host
	ORDER BY (allow_count + deny_count) DESC
	`
	var services []ServiceCount
	if err := r.db.SelectContext(ctx, &services, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("get raw service split: %w", err)
	}
	if services == nil {
		services = []ServiceCount{}
	}
	return services, nil
}

// ── Aggregate path (hourly_traffic_aggregates) ────────────────────────────────

func (r *Repository) getAggregateSummaryStats(ctx context.Context, from, to time.Time) (SummaryStats, error) {
	const query = `
	SELECT
		COALESCE(SUM(request_count), 0)                                                     AS total_requests,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END), 0)               AS allowed_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END), 0)               AS denied_count,
		COUNT(DISTINCT client_ip)                                                            AS unique_ips,
		CASE WHEN SUM(request_count) > 0
			THEN SUM(sum_duration_us) / SUM(request_count)
			ELSE 0
		END                                                                                  AS avg_duration_us
	FROM hourly_traffic_aggregates
	WHERE bucket_at >= ? AND bucket_at < ?
	`
	var stats SummaryStats
	if err := r.db.GetContext(ctx, &stats, query, from.UTC(), to.UTC()); err != nil {
		return SummaryStats{}, fmt.Errorf("get summary stats: %w", err)
	}
	return stats, nil
}

func (r *Repository) getAggregateTrafficSeries(ctx context.Context, from, to time.Time, granularity timebucket.Granularity) ([]TrafficBucket, error) {
	bucketExpr := "bucket_at" // hour — already truncated
	if granularity == timebucket.GranularityDay {
		bucketExpr = "strftime('%Y-%m-%d', bucket_at) || ' 00:00:00+00:00'"
	}

	query := fmt.Sprintf(`
	SELECT
		%s                                                                     AS timestamp,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END), 0) AS allow_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END), 0) AS deny_count
	FROM hourly_traffic_aggregates
	WHERE bucket_at >= ? AND bucket_at < ?
	GROUP BY timestamp
	ORDER BY timestamp
	`, bucketExpr)

	var buckets []TrafficBucket
	if err := r.db.SelectContext(ctx, &buckets, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("get traffic series: %w", err)
	}
	if buckets == nil {
		buckets = []TrafficBucket{}
	}
	return buckets, nil
}

func (r *Repository) getAggregateTopDeniedIPs(ctx context.Context, from, to time.Time, limit int) ([]IPCount, error) {
	const query = `
	SELECT
		client_ip                      AS ip,
		SUM(request_count)             AS count
	FROM hourly_traffic_aggregates
	WHERE outcome = 0
	  AND bucket_at >= ? AND bucket_at < ?
	GROUP BY client_ip
	ORDER BY count DESC
	LIMIT ?
	`
	var ips []IPCount
	if err := r.db.SelectContext(ctx, &ips, query, from.UTC(), to.UTC(), limit); err != nil {
		return nil, fmt.Errorf("get top denied ips: %w", err)
	}
	if ips == nil {
		ips = []IPCount{}
	}
	return ips, nil
}

func (r *Repository) getAggregateServiceSplit(ctx context.Context, from, to time.Time) ([]ServiceCount, error) {
	const query = `
	SELECT
		target_host                                                            AS host,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END), 0) AS allow_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END), 0) AS deny_count
	FROM hourly_traffic_aggregates
	WHERE bucket_at >= ? AND bucket_at < ?
	GROUP BY target_host
	ORDER BY (allow_count + deny_count) DESC
	`
	var services []ServiceCount
	if err := r.db.SelectContext(ctx, &services, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("get service split: %w", err)
	}
	if services == nil {
		services = []ServiceCount{}
	}
	return services, nil
}
