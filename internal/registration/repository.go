package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
)

// Repository handles all database operations for the registration package.
type Repository struct {
	db                *database.DB
	deviceProvisioner deviceProvisioner
}

func NewRepository(db *database.DB, provisioner deviceProvisioner) *Repository {
	return &Repository{db: db, deviceProvisioner: provisioner}
}

// CreateInvite inserts a new pending registration record.
func (r *Repository) CreateInvite(ctx context.Context, req CreateInviteRequest) (*PendingRegistration, error) {
	pendingRegistration := new(PendingRegistration)

	query := `
		INSERT INTO pending_registrations (
			device_name, owner_id, registration_code,
			heartbeat_server_url, heartbeat_interval_seconds,
			app_biometric_enabled, app_settings_locked,
			expires_at, created_at
		) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP ) RETURNING *`
	err := r.db.GetContext(ctx, pendingRegistration, query,
		req.DeviceName,
		req.OwnerID,
		req.RegistrationCode,
		req.HeartbeatServerURL,
		req.IntervalSeconds,
		req.AppBiometricEnabled,
		req.AppSettingsLocked,
		req.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return pendingRegistration, nil
}

// GetInvite returns a pending registration by ID.
func (r *Repository) GetInvite(ctx context.Context, id PendingRegistrationID) (*PendingRegistration, error) {
	row := new(PendingRegistration)
	err := r.db.GetContext(ctx, row, `SELECT * FROM pending_registrations WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return row, nil
}

// GetInviteByCode returns a pending registration by ID.
func (r *Repository) GetInviteByCode(ctx context.Context, code string) (*PendingRegistration, error) {
	row := new(PendingRegistration)
	err := r.db.GetContext(ctx, row, `SELECT * FROM pending_registrations WHERE registration_code = ?`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInviteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invite: %w", err)
	}
	return row, nil
}

// ListInvites returns registration invites, filtered by the given options.
// When filter.IncludeAll is false, only pending (unclaimed, non-invalidated, non-expired) invites are returned.
func (r *Repository) ListInvites(ctx context.Context, filter InviteFilter) ([]PendingRegistration, error) {
	var rows []PendingRegistration

	query := `SELECT * FROM pending_registrations`
	if !filter.IncludeAll {
		query += ` WHERE used_at IS NULL AND invalidated_at IS NULL AND expires_at > CURRENT_TIMESTAMP`
	}
	query += ` ORDER BY created_at DESC`

	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	return rows, nil
}

// InvalidateInvite soft-deletes a pending invite by setting invalidated_at.
// Returns ErrInviteNotFound if no matching record exists.
// Returns ErrInviteNotPending if the invite has already been used or invalidated.
func (r *Repository) InvalidateInvite(ctx context.Context, id PendingRegistrationID) error {
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
		return ErrInviteNotFound
	}
	return nil
}

// ClaimInvite Sets an invitation as used and sets the deviceID that it created
func (r *Repository) ClaimInvite(ctx context.Context, id PendingRegistrationID, deviceID device.DeviceID) (*PendingRegistration, error) {
	claimedReg := new(PendingRegistration)

	now := time.Now().UTC()
	err := r.db.GetContext(ctx, claimedReg,
		`UPDATE pending_registrations
		 SET used_at = ?, created_device_id = ?, registration_code = NULL
		 WHERE id = ?
		 RETURNING *`,
		now, deviceID, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update pending registration: %w", err)
	}
	return claimedReg, nil
}
