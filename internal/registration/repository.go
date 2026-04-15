package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository handles all database operations for the registration package.
type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// pendingRegistrationRow is the raw DB row for a pending_registrations record.
type pendingRegistrationRow struct {
	ID                     string         `db:"id"`
	DeviceName             string         `db:"device_name"`
	RegistrationCode       sql.NullString `db:"registration_code"`
	DeviceAPIKey           sql.NullString `db:"device_api_key"`
	DeviceAPIKeyPrefix     string         `db:"device_api_key_prefix"`
	HeartbeatServerURL     string         `db:"heartbeat_server_url"`
	HeartbeatIntervalSecs  int            `db:"heartbeat_interval_seconds"`
	BiometricEnabled       bool           `db:"biometric_enabled"`
	BiometricUserCanToggle bool           `db:"biometric_user_can_toggle"`
	ExpiresAt              time.Time      `db:"expires_at"`
	CreatedAt              time.Time      `db:"created_at"`
	UsedAt                 sql.NullTime   `db:"used_at"`
	CreatedDeviceID        sql.NullInt64  `db:"created_device_id"`
}

func rowToDomain(r pendingRegistrationRow) *PendingRegistration {
	p := &PendingRegistration{
		ID:                     r.ID,
		DeviceName:             r.DeviceName,
		DeviceAPIKeyPrefix:     r.DeviceAPIKeyPrefix,
		HeartbeatServerURL:     r.HeartbeatServerURL,
		IntervalSeconds:        r.HeartbeatIntervalSecs,
		BiometricEnabled:       r.BiometricEnabled,
		BiometricUserCanToggle: r.BiometricUserCanToggle,
		ExpiresAt:              r.ExpiresAt,
		CreatedAt:              r.CreatedAt,
	}
	if r.RegistrationCode.Valid {
		p.RegistrationCode = &r.RegistrationCode.String
	}
	if r.DeviceAPIKey.Valid {
		p.DeviceAPIKey = &r.DeviceAPIKey.String
	}
	if r.UsedAt.Valid {
		p.UsedAt = &r.UsedAt.Time
	}
	if r.CreatedDeviceID.Valid {
		p.CreatedDeviceID = &r.CreatedDeviceID.Int64
	}
	return p
}

// CreateInvite inserts a new pending registration record.
func (r *Repository) CreateInvite(ctx context.Context, p *PendingRegistration) error {
	query := `
		INSERT INTO pending_registrations (
			id, device_name, registration_code, device_api_key, device_api_key_prefix,
			heartbeat_server_url, heartbeat_interval_seconds,
			biometric_enabled, biometric_user_can_toggle,
			expires_at, created_at
		) VALUES (
			:id, :device_name, :registration_code, :device_api_key, :device_api_key_prefix,
			:heartbeat_server_url, :heartbeat_interval_seconds,
			:biometric_enabled, :biometric_user_can_toggle,
			:expires_at, :created_at
		)`
	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":                         p.ID,
		"device_name":                p.DeviceName,
		"registration_code":          p.RegistrationCode,
		"device_api_key":             p.DeviceAPIKey,
		"device_api_key_prefix":      p.DeviceAPIKeyPrefix,
		"heartbeat_server_url":       p.HeartbeatServerURL,
		"heartbeat_interval_seconds": p.IntervalSeconds,
		"biometric_enabled":          p.BiometricEnabled,
		"biometric_user_can_toggle":  p.BiometricUserCanToggle,
		"expires_at":                 p.ExpiresAt,
		"created_at":                 p.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	return nil
}

// GetInvite returns a pending registration by ID.
func (r *Repository) GetInvite(ctx context.Context, id string) (*PendingRegistration, error) {
	var row pendingRegistrationRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM pending_registrations WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return rowToDomain(row), nil
}

// ListInvites returns registration invites, filtered by the given options.
// When filter.IncludeAll is false, only pending (unclaimed and non-expired) invites are returned.
func (r *Repository) ListInvites(ctx context.Context, filter InviteFilter) ([]*PendingRegistration, error) {
	query := `SELECT * FROM pending_registrations`
	if !filter.IncludeAll {
		query += ` WHERE used_at IS NULL AND expires_at > CURRENT_TIMESTAMP`
	}
	query += ` ORDER BY created_at DESC`

	var rows []pendingRegistrationRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	result := make([]*PendingRegistration, 0, len(rows))
	for _, row := range rows {
		result = append(result, rowToDomain(row))
	}
	return result, nil
}

// InvalidateInvite hard-deletes an unclaimed (pending) invite.
// Returns ErrInviteNotFound if no matching record exists.
// Returns ErrInviteNotPending if the invite has already been used.
// TODO: This shouldn't hard-delete but soft-delete
func (r *Repository) InvalidateInvite(ctx context.Context, id string) error {
	// Only delete unclaimed, non-expired records.
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM pending_registrations WHERE id = ? AND used_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("invalidate invite: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("invalidate invite rows affected: %w", err)
	}
	if rows == 0 {
		// Distinguish between "never existed / already claimed" and "already used".
		var usedAt sql.NullTime
		err = r.db.QueryRowContext(ctx,
			`SELECT used_at FROM pending_registrations WHERE id = ?`, id,
		).Scan(&usedAt)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInviteNotFound
		}
		if err != nil {
			return fmt.Errorf("invalidate invite check: %w", err)
		}
		return ErrInviteNotPending
	}
	return nil
}

// ClaimInvite validates the registration code and, in a single atomic transaction:
//  1. Creates the device row.
//  2. Inserts the pre-staged API key into device_api_keys.
//  3. Marks the pending registration as used and links it to the created device.
//
// Returns ErrInviteNotFound if the code is unknown, already used, or expired.
// TODO: Add tx manager from other repositories and use it here
func (r *Repository) ClaimInvite(ctx context.Context, code string) (*ClaimResult, error) {
	logger := slog.Default()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			logger.ErrorContext(ctx, "claim invite rollback failed", slog.Any(logging.AttrKeyError, err))
		}
	}()

	// 1. Fetch and lock the pending registration.
	var row pendingRegistrationRow
	err = tx.GetContext(ctx, &row,
		`SELECT * FROM pending_registrations
		 WHERE registration_code = ? AND used_at IS NULL AND expires_at > CURRENT_TIMESTAMP`,
		code,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch pending registration: %w", err)
	}

	// device_api_key must be present (it was set at invite creation time).
	if !row.DeviceAPIKey.Valid || row.DeviceAPIKey.String == "" {
		return nil, fmt.Errorf("pending registration has no device api key")
	}
	rawAPIKey := row.DeviceAPIKey.String

	// 2. Create the device row — let SQLite assign the integer ID.
	now := time.Now().UTC()
	//TODO: Remove hardcoded owner_id. It must be added to the pending_registration UI and DB so it can be retrieved here
	result, err := tx.ExecContext(ctx,
		`INSERT INTO devices (name, device_type, created_at, updated_at, owner_id)
		 VALUES (?, 'mobile', ?, ?, ?)`,
		row.DeviceName, now, now, 1,
	)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}
	deviceID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get device id: %w", err)
	}

	// 3. Insert the pre-staged API key (hash the plaintext stored in pending_registrations).
	keyHash := device.HashAPIKey(rawAPIKey)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at)
		 VALUES (?, ?, ?, ?)`,
		deviceID, row.DeviceAPIKeyPrefix, keyHash, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert device api key: %w", err)
	}

	// 4. Mark the pending registration as used and link it to the created device.
	_, err = tx.ExecContext(ctx,
		`UPDATE pending_registrations
		 SET registration_code = NULL, device_api_key = NULL, used_at = ?, created_device_id = ?
		 WHERE id = ?`,
		now, deviceID, row.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update pending registration: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}

	return &ClaimResult{
		ServerURL:              row.HeartbeatServerURL,
		IntervalSeconds:        row.HeartbeatIntervalSecs,
		BiometricEnabled:       row.BiometricEnabled,
		BiometricUserCanToggle: row.BiometricUserCanToggle,
		RawAPIKey:              rawAPIKey,
	}, nil
}

// generateID returns a new random string ID suitable for pending_registrations.
func generateID() string {
	return uuid.NewString()
}
