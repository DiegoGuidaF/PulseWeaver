package devicepairing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// Repository handles all database operations for the devicepairing package.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// CreatePairing inserts a new device pairing record with status=pending.
func (r *Repository) CreatePairing(ctx context.Context, req CreatePairingRequest) (*DevicePairing, error) {
	row := new(pairingRow)

	query := `
		INSERT INTO device_pairings (
			device_id, pairing_code,
			heartbeat_server_url, heartbeat_interval_seconds,
			app_biometric_enabled, app_settings_locked,
			expires_at, created_at, status
		) VALUES ( ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ? ) RETURNING *`
	err := r.db.GetContext(ctx, row, query,
		req.DeviceID,
		req.PairingCode,
		req.HeartbeatServerURL,
		req.IntervalSeconds,
		req.AppBiometricEnabled,
		req.AppSettingsLocked,
		req.ExpiresAt,
		storedPending,
	)
	if err != nil {
		return nil, fmt.Errorf("create pairing: %w", err)
	}
	return new(fromRow(*row)), nil
}

// GetPairing returns a device pairing by ID.
func (r *Repository) GetPairing(ctx context.Context, id ids.DevicePairingID) (*DevicePairing, error) {
	row := new(pairingRow)
	err := r.db.GetContext(ctx, row, `SELECT * FROM device_pairings WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPairingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get pairing: %w", err)
	}
	return new(fromRow(*row)), nil
}

// GetPairingByCode returns a device pairing by its pairing code.
func (r *Repository) GetPairingByCode(ctx context.Context, code string) (*DevicePairing, error) {
	row := new(pairingRow)
	err := r.db.GetContext(ctx, row, `SELECT * FROM device_pairings WHERE pairing_code = ?`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPairingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get pairing by code: %w", err)
	}
	return new(fromRow(*row)), nil
}

// ListPairings returns device pairings for a device, filtered by the given options.
// When filter.IncludeAll is false, only claimable (pending, not yet expired) pairings are returned.
func (r *Repository) ListPairings(ctx context.Context, filter PairingFilter) ([]DevicePairing, error) {
	// Conditions vary at runtime, so build with squirrel rather than concatenating
	// SQL by hand — see docs/patterns/backend/dynamic-query-filtering.md.
	q := sq.
		Select("*").
		From("device_pairings").
		Where(sq.Eq{"device_id": filter.DeviceID})
	if !filter.IncludeAll {
		// Only claimable pairings: still pending and not yet expired.
		q = q.
			Where(sq.Eq{"status": storedPending}).
			Where(sq.Expr("expires_at > CURRENT_TIMESTAMP"))
	}

	query, args, err := q.OrderBy("created_at DESC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list pairings query: %w", err)
	}

	var rows []pairingRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("list pairings: %w", err)
	}

	pairings := make([]DevicePairing, len(rows))
	for i, row := range rows {
		pairings[i] = fromRow(row)
	}
	return pairings, nil
}

// ReplacePendingPairings marks all pending pairings for the given device as replaced.
// Called inside a transaction before creating a new pairing.
func (r *Repository) ReplacePendingPairings(ctx context.Context, deviceID ids.DeviceID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE device_pairings
		 SET status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE device_id = ? AND status = ?`,
		storedReplaced, deviceID, storedPending,
	)
	if err != nil {
		return fmt.Errorf("replace pending pairings: %w", err)
	}
	return nil
}

// InvalidatePairing soft-cancels a pending pairing by setting status=invalidated.
// Returns ErrPairingNotFound if no matching pending record exists.
func (r *Repository) InvalidatePairing(ctx context.Context, deviceID ids.DeviceID, id ids.DevicePairingID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE device_pairings
		 SET status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND device_id = ? AND status = ?`,
		storedInvalidated, id, deviceID, storedPending,
	)
	if err != nil {
		return fmt.Errorf("invalidate pairing: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("invalidate pairing rows affected: %w", err)
	}
	if rows == 0 {
		return ErrPairingNotFound
	}
	return nil
}

// ClaimPairing marks a pairing as used.
func (r *Repository) ClaimPairing(ctx context.Context, id ids.DevicePairingID) (*DevicePairing, error) {
	row := new(pairingRow)

	err := r.db.GetContext(ctx, row,
		`UPDATE device_pairings
		 SET status = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?
		 RETURNING *`,
		storedUsed, id,
	)
	if err != nil {
		return nil, fmt.Errorf("claim pairing: %w", err)
	}
	return new(fromRow(*row)), nil
}
