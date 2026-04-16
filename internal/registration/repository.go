package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// deviceProvisioner abstracts device creation during claim.
// Satisfied by *device.Service — the service is the public boundary of the device domain.
type deviceProvisioner interface {
	CreateDeviceWithAPIKey(ctx context.Context, name string, ownerID auth.UserID) (deviceID int64, rawAPIKey string, err error)
}

// Repository handles all database operations for the registration package.
type Repository struct {
	db                *sqlx.DB
	deviceProvisioner deviceProvisioner
}

func NewRepository(db *sqlx.DB, provisioner deviceProvisioner) *Repository {
	return &Repository{db: db, deviceProvisioner: provisioner}
}

// pendingRegistrationRow is the raw DB row for a pending_registrations record.
type pendingRegistrationRow struct {
	ID                    string         `db:"id"`
	DeviceName            string         `db:"device_name"`
	OwnerID               auth.UserID    `db:"owner_id"`
	RegistrationCode      sql.NullString `db:"registration_code"`
	HeartbeatServerURL    string         `db:"heartbeat_server_url"`
	HeartbeatIntervalSecs int            `db:"heartbeat_interval_seconds"`
	AppBiometricEnabled   bool           `db:"app_biometric_enabled"`
	AppSettingsLocked     bool           `db:"app_settings_locked"`
	ExpiresAt             time.Time      `db:"expires_at"`
	CreatedAt             time.Time      `db:"created_at"`
	UsedAt                sql.NullTime   `db:"used_at"`
	InvalidatedAt         sql.NullTime   `db:"invalidated_at"`
	CreatedDeviceID       sql.NullInt64  `db:"created_device_id"`
}

func rowToDomain(r pendingRegistrationRow) *PendingRegistration {
	p := &PendingRegistration{
		ID:                  r.ID,
		DeviceName:          r.DeviceName,
		OwnerID:             r.OwnerID,
		HeartbeatServerURL:  r.HeartbeatServerURL,
		IntervalSeconds:     r.HeartbeatIntervalSecs,
		AppBiometricEnabled: r.AppBiometricEnabled,
		AppSettingsLocked:   r.AppSettingsLocked,
		ExpiresAt:           r.ExpiresAt,
		CreatedAt:           r.CreatedAt,
	}
	if r.RegistrationCode.Valid {
		p.RegistrationCode = &r.RegistrationCode.String
	}
	if r.UsedAt.Valid {
		p.UsedAt = &r.UsedAt.Time
	}
	if r.InvalidatedAt.Valid {
		p.InvalidatedAt = &r.InvalidatedAt.Time
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
			id, device_name, owner_id, registration_code,
			heartbeat_server_url, heartbeat_interval_seconds,
			app_biometric_enabled, app_settings_locked,
			expires_at, created_at
		) VALUES (
			:id, :device_name, :owner_id, :registration_code,
			:heartbeat_server_url, :heartbeat_interval_seconds,
			:app_biometric_enabled, :app_settings_locked,
			:expires_at, :created_at
		)`
	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":                         p.ID,
		"device_name":                p.DeviceName,
		"owner_id":                   p.OwnerID,
		"registration_code":          p.RegistrationCode,
		"heartbeat_server_url":       p.HeartbeatServerURL,
		"heartbeat_interval_seconds": p.IntervalSeconds,
		"app_biometric_enabled":      p.AppBiometricEnabled,
		"app_settings_locked":        p.AppSettingsLocked,
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
// When filter.IncludeAll is false, only pending (unclaimed, non-invalidated, non-expired) invites are returned.
func (r *Repository) ListInvites(ctx context.Context, filter InviteFilter) ([]*PendingRegistration, error) {
	query := `SELECT * FROM pending_registrations`
	if !filter.IncludeAll {
		query += ` WHERE used_at IS NULL AND invalidated_at IS NULL AND expires_at > CURRENT_TIMESTAMP`
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

// InvalidateInvite soft-deletes a pending invite by setting invalidated_at.
// Returns ErrInviteNotFound if no matching record exists.
// Returns ErrInviteNotPending if the invite has already been used or invalidated.
func (r *Repository) InvalidateInvite(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE pending_registrations
		 SET invalidated_at = CURRENT_TIMESTAMP
		 WHERE id = ? AND used_at IS NULL AND invalidated_at IS NULL`,
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
		// Distinguish between "never existed" and "already used/invalidated".
		var exists bool
		err = r.db.QueryRowContext(ctx,
			`SELECT COUNT(*) > 0 FROM pending_registrations WHERE id = ?`, id,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("invalidate invite check: %w", err)
		}
		if !exists {
			return ErrInviteNotFound
		}
		return ErrInviteNotPending
	}
	return nil
}

// ClaimInvite validates the registration code and, in an atomic sequence:
//  1. Provisions a new device and API key via the device domain.
//  2. Marks the pending registration as used and links it to the created device.
//
// Returns ErrInviteNotFound if the code is unknown, already used, invalidated, or expired.
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

	// 1. Fetch the pending registration.
	var row pendingRegistrationRow
	err = tx.GetContext(ctx, &row,
		`SELECT * FROM pending_registrations
		 WHERE registration_code = ?
		   AND used_at IS NULL
		   AND invalidated_at IS NULL
		   AND expires_at > CURRENT_TIMESTAMP`,
		code,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("fetch pending registration: %w", err)
	}

	// 2. Provision device + API key via the device domain.
	// Note: device provisioning runs in its own internal transaction within the device service.
	// The pending_registrations update below completes the claim atomically within this tx.
	deviceID, rawAPIKey, err := r.deviceProvisioner.CreateDeviceWithAPIKey(ctx, row.DeviceName, row.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("provision device: %w", err)
	}

	// 3. Mark the pending registration as used and link it to the created device.
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx,
		`UPDATE pending_registrations
		 SET registration_code = NULL, used_at = ?, created_device_id = ?
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
		ServerURL:           row.HeartbeatServerURL,
		IntervalSeconds:     row.HeartbeatIntervalSecs,
		AppBiometricEnabled: row.AppBiometricEnabled,
		AppSettingsLocked:   row.AppSettingsLocked,
		RawAPIKey:           rawAPIKey,
	}, nil
}

// generateID returns a new random string ID suitable for pending_registrations.
func generateID() string {
	return uuid.NewString()
}
