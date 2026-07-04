package anomaly

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// Repository is the DB boundary for anomaly persistence: the scan cursor,
// finding upserts, and (in later tasks) the read queries behind the API.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// LoadScanState returns the single scan-state row, or a zero cursor when the
// scan has never run.
func (r *Repository) LoadScanState(ctx context.Context) (ScanState, error) {
	var row struct {
		LastAccessLogID int64      `db:"last_access_log_id"`
		LastBucketAt    *time.Time `db:"last_bucket_at"`
	}
	err := r.db.GetContext(ctx, &row,
		`SELECT last_access_log_id, last_bucket_at FROM anomaly_scan_state WHERE id = 1`)
	if errors.Is(err, sql.ErrNoRows) {
		return ScanState{}, nil
	}
	if err != nil {
		return ScanState{}, fmt.Errorf("load anomaly scan state: %w", err)
	}
	return ScanState{LastAccessLogID: row.LastAccessLogID, LastBucketAt: row.LastBucketAt}, nil
}

// SaveScanState upserts the single scan-state row (id pinned to 1 by the schema).
func (r *Repository) SaveScanState(ctx context.Context, s ScanState) error {
	const query = `
INSERT INTO anomaly_scan_state (id, last_access_log_id, last_bucket_at)
VALUES (1, ?, ?)
ON CONFLICT (id) DO UPDATE SET
    last_access_log_id = excluded.last_access_log_id,
    last_bucket_at     = excluded.last_bucket_at`
	if _, err := r.db.ExecContext(ctx, query, s.LastAccessLogID, s.LastBucketAt); err != nil {
		return fmt.Errorf("save anomaly scan state: %w", err)
	}
	return nil
}

// MaxAccessLogID snapshots the highest access_log id, bounding the raw scan
// window. Zero when the log is empty.
func (r *Repository) MaxAccessLogID(ctx context.Context) (int64, error) {
	var maxID int64
	if err := r.db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(id), 0) FROM access_log`); err != nil {
		return 0, fmt.Errorf("max access log id: %w", err)
	}
	return maxID, nil
}

// UpsertFinding writes a finding as an open anomaly, or — when an open row with
// the same fingerprint already exists — advances its last_seen_at and evidence.
// The partial unique index on (fingerprint) WHERE status = 'open' is the conflict
// target, so an acknowledged row with the same fingerprint does not block a new
// open row.
func (r *Repository) UpsertFinding(ctx context.Context, f Finding) error {
	evidence := []byte("{}")
	if f.Evidence != nil {
		b, err := json.Marshal(f.Evidence)
		if err != nil {
			return fmt.Errorf("marshal anomaly evidence: %w", err)
		}
		evidence = b
	}

	const query = `
INSERT INTO anomalies
    (kind, severity, status, fingerprint, first_seen_at, last_seen_at,
     device_id, device_name, user_id, user_name, client_ip, target_host, country_code, evidence_json)
VALUES (?, ?, 'open', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (fingerprint) WHERE status = 'open'
DO UPDATE SET
    last_seen_at  = excluded.last_seen_at,
    evidence_json = excluded.evidence_json`
	_, err := r.db.ExecContext(ctx, query,
		string(f.Kind), string(f.Severity), f.Fingerprint, f.ObservedAt, f.ObservedAt,
		nullableID(f.DeviceID), f.DeviceName, nullableID(f.UserID), f.UserName,
		nullable(f.ClientIP), nullable(f.TargetHost), nullable(f.CountryCode), string(evidence))
	if err != nil {
		return fmt.Errorf("upsert anomaly finding: %w", err)
	}
	return nil
}

// UpsertDeviceProfile records one profile sighting: it inserts a first-seen
// (device, dimension, fingerprint) row or, when the key already exists, advances
// last_seen_at and bumps seen_count. first_seen_at is never moved, so the
// learning gate keeps measuring from the genuine first sighting. last_seen_at
// takes the later of the two so an out-of-order sighting never rewinds it.
func (r *Repository) UpsertDeviceProfile(ctx context.Context, o ProfileObservation) error {
	const query = `
INSERT INTO device_profiles (device_id, dimension, fingerprint, first_seen_at, last_seen_at, seen_count)
VALUES (?, ?, ?, ?, ?, 1)
ON CONFLICT (device_id, dimension, fingerprint) DO UPDATE SET
    last_seen_at = MAX(device_profiles.last_seen_at, excluded.last_seen_at),
    seen_count   = device_profiles.seen_count + 1`
	if _, err := r.db.ExecContext(ctx, query,
		o.DeviceID, o.Dimension, o.Fingerprint, o.SeenAt, o.SeenAt); err != nil {
		return fmt.Errorf("upsert device profile: %w", err)
	}
	return nil
}

// WithinTx exposes the repository's transaction scope so the job can atomically
// upsert findings and advance the watermark together.
func (r *Repository) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithinTx(ctx, fn)
}

// nullableID renders a typed ID pointer as a driver-friendly int64 or NULL.
func nullableID[T interface{ Int64() int64 }](id *T) any {
	if id == nil {
		return nil
	}
	return (*id).Int64()
}

// nullable renders any pointer as its value or NULL.
func nullable[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
