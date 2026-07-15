package anomaly

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// UARow is one allowed access_log row attributed to a device via a contributor,
// carrying the raw header set the novelty family parses the User-Agent out of. A
// shared IP yields one row per contributing device on the same log row.
type UARow struct {
	DeviceID    int64           `db:"device_id"`
	DeviceName  string          `db:"device_name"`
	UserID      int64           `db:"user_id"`
	UserName    string          `db:"user_name"`
	ClientIP    string          `db:"client_ip"`
	HeadersJSON string          `db:"headers_json"`
	CreatedAt   database.DBTime `db:"created_at"`
}

// NewUserAgentRows returns allowed rows in (fromID, toID] that have a device
// contributor. Only allowed traffic is scanned — the allowed-but-unfamiliar
// request is the threat; header parsing happens in the detector, bounded by the
// watermark delta rather than the table size.
func (r *Repository) NewUserAgentRows(ctx context.Context, fromID, toID int64) ([]UARow, error) {
	const query = `
SELECT
    c.device_id      AS device_id,
    d.name           AS device_name,
    c.user_id        AS user_id,
    u.display_name   AS user_name,
    al.client_ip     AS client_ip,
    al.headers_json  AS headers_json,
    al.created_at    AS created_at
FROM access_log al
JOIN access_log_contributors c ON c.access_log_id = al.id
JOIN devices d ON d.id = c.device_id
JOIN users   u ON u.id = c.user_id
WHERE al.outcome = 1
  AND al.id > ? AND al.id <= ?`
	var rows []UARow
	if err := r.db.SelectContext(ctx, &rows, query, fromID, toID); err != nil {
		return nil, fmt.Errorf("read new-user-agent rows: %w", err)
	}
	return rows, nil
}

// CountryTrafficRow is one allowed row's persisted GeoIP country attributed to a
// device — the traffic-derived half of new_country.
type CountryTrafficRow struct {
	DeviceID    int64           `db:"device_id"`
	DeviceName  string          `db:"device_name"`
	UserID      int64           `db:"user_id"`
	UserName    string          `db:"user_name"`
	CountryCode string          `db:"country_code"`
	CreatedAt   database.DBTime `db:"created_at"`
}

// AllowedTrafficCountries returns allowed rows in (fromID, toID] whose access_log
// row carries a resolved country (from the log-time GeoIP child table), attributed
// via contributors. Empty country codes are excluded at the source.
func (r *Repository) AllowedTrafficCountries(ctx context.Context, fromID, toID int64) ([]CountryTrafficRow, error) {
	const query = `
SELECT
    c.device_id      AS device_id,
    d.name           AS device_name,
    c.user_id        AS user_id,
    u.display_name   AS user_name,
    g.country_code   AS country_code,
    al.created_at    AS created_at
FROM access_log al
JOIN access_log_contributors c ON c.access_log_id = al.id
JOIN access_log_geoip g ON g.access_log_id = al.id
JOIN devices d ON d.id = c.device_id
JOIN users   u ON u.id = c.user_id
WHERE al.outcome = 1
  AND al.id > ? AND al.id <= ?
  AND g.country_code != ''`
	var rows []CountryTrafficRow
	if err := r.db.SelectContext(ctx, &rows, query, fromID, toID); err != nil {
		return nil, fmt.Errorf("read allowed-traffic countries: %w", err)
	}
	return rows, nil
}

// AddressSightingRow is one (device, IP) the device had enabled — either a
// currently-enabled address, or one enabled at least once within a trailing
// window. CreatedAt is the most recent enable in the window (zero for the
// currently-enabled read, which does not window).
type AddressSightingRow struct {
	DeviceID   int64           `db:"device_id"`
	DeviceName string          `db:"device_name"`
	UserID     int64           `db:"user_id"`
	UserName   string          `db:"user_name"`
	IP         string          `db:"ip"`
	CreatedAt  database.DBTime `db:"created_at"`
}

// EnabledAddressSightings returns one row per (device, IP) enabled at least once
// since `since`, ordered by device then most-recent enable. Collapsing per
// (device, IP) folds away heartbeat refreshes — which re-emit an enable event
// each beat — so the result is bounded by distinct addresses, not traffic. Feeds
// new_country's stolen-key guard and impossible_travel's country-hop signal.
func (r *Repository) EnabledAddressSightings(ctx context.Context, since time.Time) ([]AddressSightingRow, error) {
	const query = `
SELECT
    a.device_id        AS device_id,
    d.name             AS device_name,
    d.owner_id         AS user_id,
    u.display_name     AS user_name,
    a.ip               AS ip,
    MAX(e.created_at)  AS created_at
FROM address_events e
JOIN addresses a ON a.id = e.address_id
JOIN devices   d ON d.id = a.device_id
JOIN users     u ON u.id = d.owner_id
WHERE e.is_enabled = 1
  AND e.created_at >= ?
GROUP BY a.device_id, a.ip
ORDER BY a.device_id, created_at`
	var rows []AddressSightingRow
	if err := r.db.SelectContext(ctx, &rows, query, since.UTC()); err != nil {
		return nil, fmt.Errorf("read enabled address sightings: %w", err)
	}
	return rows, nil
}

// EnabledAddresses returns one row per currently-enabled (device, IP). It seeds
// impossible_travel's concurrent-presence check — no time window, the live
// picture of where each device is present right now.
func (r *Repository) EnabledAddresses(ctx context.Context) ([]AddressSightingRow, error) {
	const query = `
SELECT
    a.device_id    AS device_id,
    d.name         AS device_name,
    d.owner_id     AS user_id,
    u.display_name AS user_name,
    a.ip           AS ip
FROM addresses a
JOIN devices d ON d.id = a.device_id
JOIN users   u ON u.id = d.owner_id
WHERE a.is_enabled = 1
ORDER BY a.device_id`
	var rows []AddressSightingRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("read enabled addresses: %w", err)
	}
	return rows, nil
}

// DeviceProfileRow is one learned (device, dimension, fingerprint) baseline row.
// FirstSeenAt drives the per-device learning gate.
type DeviceProfileRow struct {
	DeviceID    int64           `db:"device_id"`
	Dimension   string          `db:"dimension"`
	Fingerprint string          `db:"fingerprint"`
	FirstSeenAt database.DBTime `db:"first_seen_at"`
}

// DeviceProfiles loads every profile row for the given devices — the pass's whole
// novelty baseline in one read. Scoped to the devices actually seen this pass, so
// the set stays proportional to new traffic, not to the fleet size.
func (r *Repository) DeviceProfiles(ctx context.Context, deviceIDs []int64) ([]DeviceProfileRow, error) {
	if len(deviceIDs) == 0 {
		return nil, nil
	}
	query, args, err := sq.
		Select("device_id", "dimension", "fingerprint", "first_seen_at").
		From("device_profiles").
		Where(sq.Eq{"device_id": deviceIDs}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device-profiles query: %w", err)
	}
	var rows []DeviceProfileRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("read device profiles: %w", err)
	}
	return rows, nil
}
