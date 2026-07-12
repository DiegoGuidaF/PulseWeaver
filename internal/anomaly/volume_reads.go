package anomaly

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// BucketCount is one hourly bucket's summed request count for a single series.
type BucketCount struct {
	BucketAt database.DBTime `db:"bucket_at"`
	Count    int64           `db:"count"`
}

// HostBucketCount is one (host, outcome) series' hourly count.
type HostBucketCount struct {
	TargetHost string          `db:"target_host"`
	Outcome    int             `db:"outcome"`
	BucketAt   database.DBTime `db:"bucket_at"`
	Count      int64           `db:"count"`
}

// EntityBucketCount is one attributed (kind, name, outcome) series' hourly count.
// EntityID is nil once the entity is hard-deleted; the name survives.
type EntityBucketCount struct {
	EntityKind string          `db:"entity_kind"`
	EntityName string          `db:"entity_name"`
	EntityID   *int64          `db:"entity_id"`
	Outcome    int             `db:"outcome"`
	BucketAt   database.DBTime `db:"bucket_at"`
	Count      int64           `db:"count"`
}

// CountryBucketCount is one country's denied hourly count with a host sample and
// a representative IP for ASN enrichment.
type CountryBucketCount struct {
	CountryCode   string          `db:"country_code"`
	CountryName   string          `db:"country_name"`
	ContinentCode string          `db:"continent_code"`
	BucketAt      database.DBTime `db:"bucket_at"`
	Count         int64           `db:"count"`
	Hosts         *string         `db:"hosts"`
	SampleIP      string          `db:"sample_ip"`
}

// LastAggregateBucketAt returns the most recent bucket_at stored in
// hourly_traffic_aggregates, or nil when the table has no rows.
func (r *Repository) LastAggregateBucketAt(ctx context.Context) (*time.Time, error) {
	var t database.DBTime
	const query = `SELECT MAX(bucket_at) FROM hourly_traffic_aggregates`
	if err := r.db.GetContext(ctx, &t, query); err != nil {
		return nil, fmt.Errorf("last aggregate bucket at: %w", err)
	}
	if t.IsZero() {
		return nil, nil
	}
	last := t.UTC()
	return &last, nil
}

// GlobalDenyBuckets returns the global denied-per-hour series over [from, to).
func (r *Repository) GlobalDenyBuckets(ctx context.Context, from, to time.Time) ([]BucketCount, error) {
	const query = `
SELECT bucket_at, SUM(request_count) AS count
FROM hourly_traffic_aggregates
WHERE outcome = 0 AND bucket_at >= ? AND bucket_at < ?
GROUP BY bucket_at`
	var rows []BucketCount
	if err := r.db.SelectContext(ctx, &rows, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("read global deny buckets: %w", err)
	}
	return rows, nil
}

// HostTrafficBuckets returns per-(host, outcome) hourly counts over [from, to).
func (r *Repository) HostTrafficBuckets(ctx context.Context, from, to time.Time) ([]HostBucketCount, error) {
	const query = `
SELECT target_host, outcome, bucket_at, SUM(request_count) AS count
FROM hourly_traffic_aggregates
WHERE target_host != '' AND bucket_at >= ? AND bucket_at < ?
GROUP BY target_host, outcome, bucket_at`
	var rows []HostBucketCount
	if err := r.db.SelectContext(ctx, &rows, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("read host traffic buckets: %w", err)
	}
	return rows, nil
}

// AttributionBuckets returns per-(entity_kind, entity_name, outcome) hourly
// counts over [from, to).
func (r *Repository) AttributionBuckets(ctx context.Context, from, to time.Time) ([]EntityBucketCount, error) {
	const query = `
SELECT entity_kind, entity_name, MAX(entity_id) AS entity_id, outcome, bucket_at, SUM(request_count) AS count
FROM hourly_attribution_aggregates
WHERE bucket_at >= ? AND bucket_at < ?
GROUP BY entity_kind, entity_name, outcome, bucket_at`
	var rows []EntityBucketCount
	if err := r.db.SelectContext(ctx, &rows, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("read attribution buckets: %w", err)
	}
	return rows, nil
}

// EnabledAddressIPs returns the IPs of all currently-enabled addresses, the
// seed for the geo expected-country set.
func (r *Repository) EnabledAddressIPs(ctx context.Context) ([]string, error) {
	var ips []string
	if err := r.db.SelectContext(ctx, &ips, `SELECT ip FROM addresses WHERE is_enabled = 1`); err != nil {
		return nil, fmt.Errorf("read enabled address ips: %w", err)
	}
	return ips, nil
}

// AllowedCountries returns the distinct countries seen in allowed aggregate rows
// since `since` — the traffic-derived half of the geo expected set.
func (r *Repository) AllowedCountries(ctx context.Context, since time.Time) ([]string, error) {
	const query = `
SELECT DISTINCT country_code
FROM hourly_traffic_aggregates
WHERE outcome = 1 AND country_code != '' AND bucket_at >= ?`
	var countries []string
	if err := r.db.SelectContext(ctx, &countries, query, since.UTC()); err != nil {
		return nil, fmt.Errorf("read allowed countries: %w", err)
	}
	return countries, nil
}

// DeniedCountryBuckets returns per-(country, hour) denied counts over [from, to),
// with a bounded host sample and a representative IP per group.
func (r *Repository) DeniedCountryBuckets(ctx context.Context, from, to time.Time) ([]CountryBucketCount, error) {
	const query = `
SELECT
    country_code,
    MAX(country_name)                     AS country_name,
    MAX(continent_code)                   AS continent_code,
    bucket_at,
    SUM(request_count)                    AS count,
    GROUP_CONCAT(DISTINCT target_host)    AS hosts,
    MAX(client_ip)                        AS sample_ip
FROM hourly_traffic_aggregates
WHERE outcome = 0 AND country_code != '' AND bucket_at >= ? AND bucket_at < ?
GROUP BY country_code, bucket_at`
	var rows []CountryBucketCount
	if err := r.db.SelectContext(ctx, &rows, query, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("read denied country buckets: %w", err)
	}
	return rows, nil
}
