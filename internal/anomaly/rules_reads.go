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
