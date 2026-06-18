package rollup

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

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
//
// Country attribution rides along without changing row cardinality: country is
// a function of client_ip, so the existing GROUP BY already isolates it. MAX
// picks the geo-enriched value over the empty string when only some of a group's rows carry
// a geoip child row (e.g. the resolver came up mid-hour).
func (r *Repository) RunRollup(ctx context.Context, from, to time.Time) error {
	const query = `
		INSERT OR REPLACE INTO hourly_traffic_aggregates
			(bucket_at, client_ip, target_host, outcome, deny_reason, request_count, sum_duration_us, max_duration_us,
			 country_code, country_name, continent_code)
		SELECT
			strftime('%Y-%m-%d %H:00:00', created_at) || '+00:00' AS bucket_at,
			client_ip,
			COALESCE(target_host, '')                              AS target_host,
			outcome,
			COALESCE(deny_reason, '')                              AS deny_reason,
			COUNT(*)                                               AS request_count,
			SUM(COALESCE(duration_us, 0))                          AS sum_duration_us,
			MAX(COALESCE(duration_us, 0))                          AS max_duration_us,
			MAX(COALESCE(g.country_code, ''))                      AS country_code,
			MAX(COALESCE(g.country_name, ''))                      AS country_name,
			MAX(COALESCE(g.continent_code, ''))                    AS continent_code
		FROM access_log
		LEFT JOIN access_log_geoip g ON g.access_log_id = access_log.id
		WHERE created_at >= ?
		  AND created_at <  ?
		  AND strftime('%Y-%m-%d %H:00:00', created_at) IS NOT NULL
		GROUP BY bucket_at, client_ip, target_host, outcome, deny_reason
		`
	if _, err := r.db.ExecContext(ctx, query, from.UTC(), to.UTC()); err != nil {
		return fmt.Errorf("run rollup: %w", err)
	}
	return nil
}

// EarliestAccessLogAt returns the oldest created_at in access_log, or a zero
// time.Time if the table is empty. Used to bound the first rollup catch-up.
func (r *Repository) EarliestAccessLogAt(ctx context.Context) (time.Time, error) {
	var t database.DBTime
	const query = `SELECT MIN(created_at) FROM access_log`
	if err := r.db.GetContext(ctx, &t, query); err != nil {
		return time.Time{}, fmt.Errorf("earliest access log at: %w", err)
	}
	return t.UTC(), nil
}

// DeleteAggregatesOlderThan prunes hourly_traffic_aggregates buckets that start
// before the given cutoff and returns the number of deleted rows.
func (r *Repository) DeleteAggregatesOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM hourly_traffic_aggregates WHERE bucket_at < ?`, before.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete aggregates older than: %w", err)
	}
	return res.RowsAffected()
}
