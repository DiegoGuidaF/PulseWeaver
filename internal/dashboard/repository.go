package dashboard

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/jmoiron/sqlx"
)

// Repository provides both read and write access to traffic aggregates.
type Repository struct {
	db     dbInterface
	rootDB *sqlx.DB
}

type dbInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		rootDB: db,
		db:     db,
	}
}

// RunRollup aggregates request_audit_log rows in [from, to) into hourly_traffic_aggregates.
// Idempotent via INSERT OR REPLACE on the unique index.
//
// The strftime output is concatenated with '+00:00' so that bucket_at stores
// values in the same format the driver produces for time.Time parameters
// ("2006-01-02 15:04:05+00:00"), keeping WHERE comparisons consistent.
func (r *Repository) RunRollup(ctx context.Context, from, to time.Time) error {
	const query = `
		INSERT OR REPLACE INTO hourly_traffic_aggregates
			(bucket_at, client_ip, target_host, outcome, deny_reason, request_count)
		SELECT
			strftime('%Y-%m-%d %H:00:00', created_at) || '+00:00' AS bucket_at,
			client_ip,
			COALESCE(target_host, '')                              AS target_host,
			outcome,
			COALESCE(deny_reason, '')                              AS deny_reason,
			COUNT(*)                                               AS request_count
		FROM request_audit_log
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
func (r *Repository) GetSummaryStats(ctx context.Context, from, to time.Time) (SummaryStats, error) {
	const query = `
	SELECT
		COALESCE(SUM(request_count), 0)                                       AS total_requests,
		COALESCE(SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END), 0) AS allowed_count,
		COALESCE(SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END), 0) AS denied_count,
		COUNT(DISTINCT client_ip)                                              AS unique_ips
	FROM hourly_traffic_aggregates
	WHERE bucket_at >= ? AND bucket_at < ?
	`
	var stats SummaryStats
	if err := r.db.GetContext(ctx, &stats, query, from.UTC(), to.UTC()); err != nil {
		return SummaryStats{}, fmt.Errorf("get summary stats: %w", err)
	}
	return stats, nil
}

// GetTrafficSeries returns time-bucketed allow/deny counts.
// granularity must be "hour" or "day".
func (r *Repository) GetTrafficSeries(ctx context.Context, from, to time.Time, granularity timebucket.Granularity) ([]TrafficBucket, error) {
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

// GetTopDeniedIPs returns the top denied IPs by total denied request count.
func (r *Repository) GetTopDeniedIPs(ctx context.Context, from, to time.Time, limit int) ([]IPCount, error) {
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

// GetServiceSplit returns per-host allow/deny counts.
func (r *Repository) GetServiceSplit(ctx context.Context, from, to time.Time) ([]ServiceCount, error) {
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
