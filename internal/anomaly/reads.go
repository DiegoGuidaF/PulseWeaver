package anomaly

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// ExpiredAccessRow is one (device, client_ip) group of ip_not_registered denies
// whose address was disabled shortly before the deny.
type ExpiredAccessRow struct {
	DeviceID    int64           `db:"device_id"`
	DeviceName  string          `db:"device_name"`
	UserID      int64           `db:"user_id"`
	UserName    string          `db:"user_name"`
	ClientIP    string          `db:"client_ip"`
	DenyCount   int64           `db:"deny_count"`
	FirstSeen   database.DBTime `db:"first_seen"`
	LastSeen    database.DBTime `db:"last_seen"`
	DisabledAt  database.DBTime `db:"disabled_at"`
	LeaseSource string          `db:"lease_source"`
}

// ExpiredAccessDenials returns ip_not_registered denies in (fromID, toID] whose
// client_ip maps to a disabled address whose most recent disable event falls
// within grace before the deny. One row per matching device (a shared IP maps to
// several devices).
func (r *Repository) ExpiredAccessDenials(ctx context.Context, fromID, toID int64, grace time.Duration) ([]ExpiredAccessRow, error) {
	graceModifier := fmt.Sprintf("-%d minutes", int64(grace.Minutes()))
	const query = `
SELECT
    a.device_id                     AS device_id,
    d.name                          AS device_name,
    d.owner_id                      AS user_id,
    u.display_name                  AS user_name,
    al.client_ip                    AS client_ip,
    COUNT(*)                        AS deny_count,
    MIN(al.created_at)              AS first_seen,
    MAX(al.created_at)              AS last_seen,
    MAX(disable.disabled_at)        AS disabled_at,
    a.source                        AS lease_source
FROM access_log al
JOIN addresses a ON a.ip = al.client_ip AND a.is_enabled = 0
JOIN devices  d ON d.id = a.device_id
JOIN users    u ON u.id = d.owner_id
JOIN (
    SELECT address_id, MAX(created_at) AS disabled_at
    FROM address_events
    WHERE is_enabled = 0
    GROUP BY address_id
) disable ON disable.address_id = a.id
WHERE al.outcome = 0
  AND al.deny_reason = 'ip_not_registered'
  AND al.id > ? AND al.id <= ?
  AND disable.disabled_at <= al.created_at
  AND disable.disabled_at >= datetime(al.created_at, ?)
GROUP BY a.device_id, al.client_ip`
	var rows []ExpiredAccessRow
	if err := r.db.SelectContext(ctx, &rows, query, fromID, toID, graceModifier); err != nil {
		return nil, fmt.Errorf("read expired-access denials: %w", err)
	}
	return rows, nil
}

// InvalidTokenRow is one (client_ip, UTC day) group of invalid_token denies.
type InvalidTokenRow struct {
	ClientIP    string          `db:"client_ip"`
	UTCDay      string          `db:"utc_day"`
	DenyCount   int64           `db:"deny_count"`
	FirstSeen   database.DBTime `db:"first_seen"`
	LastSeen    database.DBTime `db:"last_seen"`
	TargetHosts *string         `db:"target_hosts"`
}

// InvalidTokenDenials returns invalid_token denies in (fromID, toID] grouped by
// source IP and UTC day.
func (r *Repository) InvalidTokenDenials(ctx context.Context, fromID, toID int64) ([]InvalidTokenRow, error) {
	const query = `
SELECT
    al.client_ip                       AS client_ip,
    date(al.created_at)                AS utc_day,
    COUNT(*)                           AS deny_count,
    MIN(al.created_at)                 AS first_seen,
    MAX(al.created_at)                 AS last_seen,
    GROUP_CONCAT(DISTINCT al.target_host) AS target_hosts
FROM access_log al
WHERE al.outcome = 0
  AND al.deny_reason = 'invalid_token'
  AND al.id > ? AND al.id <= ?
GROUP BY al.client_ip, date(al.created_at)`
	var rows []InvalidTokenRow
	if err := r.db.SelectContext(ctx, &rows, query, fromID, toID); err != nil {
		return nil, fmt.Errorf("read invalid-token denials: %w", err)
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
