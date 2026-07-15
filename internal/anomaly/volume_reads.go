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

// DeviceDenyRow is one device's aggregate of host_not_allowed denies over a
// trailing window.
type DeviceDenyRow struct {
	DeviceID      int64           `db:"device_id"`
	DeviceName    string          `db:"device_name"`
	UserID        int64           `db:"user_id"`
	UserName      string          `db:"user_name"`
	DistinctHosts int64           `db:"distinct_hosts"`
	DenyCount     int64           `db:"deny_count"`
	Hosts         *string         `db:"hosts"`
	FirstSeen     database.DBTime `db:"first_seen"`
	LastSeen      database.DBTime `db:"last_seen"`
}

// HostProbingDenials returns devices denied host_not_allowed on at least
// threshold distinct hosts since the window start. Device attribution comes from
// access_log_contributors (written whenever the IP matched a device, including
// host_not_allowed denies).
func (r *Repository) HostProbingDenials(ctx context.Context, since time.Time, threshold int) ([]DeviceDenyRow, error) {
	const query = `
SELECT
    c.device_id                        AS device_id,
    d.name                             AS device_name,
    d.owner_id                         AS user_id,
    u.display_name                     AS user_name,
    COUNT(DISTINCT al.target_host)     AS distinct_hosts,
    COUNT(*)                           AS deny_count,
    GROUP_CONCAT(DISTINCT al.target_host) AS hosts,
    MIN(al.created_at)                 AS first_seen,
    MAX(al.created_at)                 AS last_seen
FROM access_log al
JOIN access_log_contributors c ON c.access_log_id = al.id
JOIN devices d ON d.id = c.device_id
JOIN users   u ON u.id = d.owner_id
WHERE al.outcome = 0
  AND al.deny_reason = 'host_not_allowed'
  AND al.created_at >= ?
GROUP BY c.device_id
HAVING COUNT(DISTINCT al.target_host) >= ?`
	var rows []DeviceDenyRow
	if err := r.db.SelectContext(ctx, &rows, query, since.UTC(), threshold); err != nil {
		return nil, fmt.Errorf("read host-probing denials: %w", err)
	}
	return rows, nil
}

// AddressChurnRow is one device's count of addresses created over a trailing
// window.
type AddressChurnRow struct {
	DeviceID     int64           `db:"device_id"`
	DeviceName   string          `db:"device_name"`
	UserID       int64           `db:"user_id"`
	UserName     string          `db:"user_name"`
	NewAddresses int64           `db:"new_addresses"`
	FirstSeen    database.DBTime `db:"first_seen"`
	LastSeen     database.DBTime `db:"last_seen"`
}

// AddressChurn returns devices that created at least threshold addresses since
// the window start. Counting new address rows (not events) distinguishes a fresh
// IP from a heartbeat refresh of an existing one.
func (r *Repository) AddressChurn(ctx context.Context, since time.Time, threshold int) ([]AddressChurnRow, error) {
	const query = `
SELECT
    a.device_id           AS device_id,
    d.name                AS device_name,
    d.owner_id            AS user_id,
    u.display_name        AS user_name,
    COUNT(*)              AS new_addresses,
    MIN(a.created_at)     AS first_seen,
    MAX(a.created_at)     AS last_seen
FROM addresses a
JOIN devices d ON d.id = a.device_id
JOIN users   u ON u.id = d.owner_id
WHERE a.created_at >= ?
GROUP BY a.device_id
HAVING COUNT(*) >= ?`
	var rows []AddressChurnRow
	if err := r.db.SelectContext(ctx, &rows, query, since.UTC(), threshold); err != nil {
		return nil, fmt.Errorf("read address churn: %w", err)
	}
	return rows, nil
}
