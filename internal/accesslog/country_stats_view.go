package accesslog

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
)

// AccessLogCountryStat holds aggregated request counts for a single country.
type AccessLogCountryStat struct {
	CountryCode string
	CountryName string
	Total       int64
	Allowed     int64
	Denied      int64
}

// ListAccessLogStatsByCountry returns request counts grouped by country for all rows
// within the [from, to] time window. Only rows with GeoIP data are included.
//
// Dispatches on rollup.RawWindowThreshold like every other traffic widget,
// so the map/country tables answer from the same source as the stat cards and
// charts for a given window.
func (r *Repository) ListAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
	if to.Sub(from) <= rollup.RawWindowThreshold {
		return r.listRawAccessLogStatsByCountry(ctx, from, to)
	}
	return r.listAggregateAccessLogStatsByCountry(ctx, from, to)
}

func (r *Repository) listRawAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
	const query = `
		SELECT
			g.country_code,
			COALESCE(g.country_name, '')  AS country_name,
			COUNT(*) AS total,
			SUM(CASE WHEN ral.outcome = 1 THEN 1 ELSE 0 END) AS allowed,
			SUM(CASE WHEN ral.outcome = 0 THEN 1 ELSE 0 END) AS denied
		FROM access_log_geoip g
		JOIN access_log ral ON ral.id = g.access_log_id
		WHERE ral.created_at >= ? AND ral.created_at <= ?
		GROUP BY g.country_code, g.country_name
		ORDER BY total DESC
	`

	var rows []dbCountryStatsRow
	if err := r.db.SelectContext(ctx, &rows, query, from, to); err != nil {
		return nil, fmt.Errorf("list access log stats by country: %w", err)
	}

	return countryStatsFromRows(rows), nil
}

// listAggregateAccessLogStatsByCountry answers from hourly_traffic_aggregates.
// Buckets without country attribution (empty country_code: no GeoIP at rollup
// time, or rolled up before country columns existed) are excluded, matching
// the raw path's inner join on access_log_geoip.
func (r *Repository) listAggregateAccessLogStatsByCountry(ctx context.Context, from, to time.Time) ([]AccessLogCountryStat, error) {
	const query = `
		SELECT
			country_code,
			country_name,
			SUM(request_count) AS total,
			SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END) AS allowed,
			SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END) AS denied
		FROM hourly_traffic_aggregates
		WHERE country_code != ''
		  AND bucket_at >= ? AND bucket_at < ?
		GROUP BY country_code, country_name
		ORDER BY total DESC
	`

	var rows []dbCountryStatsRow
	if err := r.db.SelectContext(ctx, &rows, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("list aggregate access log stats by country: %w", err)
	}

	return countryStatsFromRows(rows), nil
}

func countryStatsFromRows(rows []dbCountryStatsRow) []AccessLogCountryStat {
	stats := make([]AccessLogCountryStat, len(rows))
	for i, row := range rows {
		stats[i] = AccessLogCountryStat(row)
	}
	return stats
}

type dbCountryStatsRow struct {
	CountryCode string `db:"country_code"`
	CountryName string `db:"country_name"`
	Total       int64  `db:"total"`
	Allowed     int64  `db:"allowed"`
	Denied      int64  `db:"denied"`
}
