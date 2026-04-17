package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/jmoiron/sqlx"
)

// deviceProvisioner abstracts device creation during claim.
// Satisfied by *device.Service — the service is the public boundary of the device domain.
// TODO: This should go to the service ideally
type deviceProvisioner interface {
	CreateDeviceWithAPIKey(ctx context.Context, name string, ownerID auth.UserID) (deviceID device.DeviceID, rawAPIKey string, err error)
}

// Repository handles all database operations for the registration package.
type Repository struct {
	db                dBInterface
	rootDB            *sqlx.DB
	deviceProvisioner deviceProvisioner
}

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB, provisioner deviceProvisioner) *Repository {
	return &Repository{db: db, deviceProvisioner: provisioner, rootDB: db}
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
		// Distinguish between "never existed" and "already used/invalidated".
		var exists bool
		err = r.rootDB.QueryRowContext(ctx,
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
// TODO: CONTINUE HERE!!!
// It is not clear how the transaction accross domains should be managed. Ideally it should be something like the
// registration service starts a transaction that is received as a parameter on the device domain for the device
// creation and then the registration domain continues to use it to confirm the pendingRegistration.
// Unsure how interfaces are defined and how much it should change for this.
// Another solution could be event driven. When device is created we look for a pendingRegistration for it and if present
// we confirm it and fill in the data. Eventual consistency
func (r *Repository) ClaimInvite(ctx context.Context, code string) (*ClaimResult, error) {
	claimedReg := new(PendingRegistration)
	var rawAPIKey string

	err := r.runInTx(ctx, func(tx *Repository) error {
		// 1. Fetch the pending registration.
		err := tx.db.GetContext(ctx, claimedReg,
			`SELECT * FROM pending_registrations
		 WHERE registration_code = ?
		   AND used_at IS NULL
		   AND invalidated_at IS NULL
		   AND expires_at > CURRENT_TIMESTAMP`,
			code,
		)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInviteNotFound
		}
		if err != nil {
			return fmt.Errorf("fetch pending registration: %w", err)
		}

		// 2. Provision device + API key via the device domain.
		// Note: device provisioning runs in its own internal transaction within the device service.
		// The pending_registrations update below completes the claim atomically within this tx.
		var deviceID device.DeviceID
		deviceID, rawAPIKey, err = r.deviceProvisioner.CreateDeviceWithAPIKey(ctx, claimedReg.DeviceName, claimedReg.OwnerID)
		if err != nil {
			return fmt.Errorf("provision device: %w", err)
		}

		// 3. Mark the pending registration as used and link it to the created device.
		now := time.Now().UTC()
		_, err = r.db.ExecContext(ctx,
			`UPDATE pending_registrations
		 SET registration_code = NULL, used_at = ?, created_device_id = ?
		 WHERE id = ?`,
			now, deviceID, claimedReg.ID,
		)
		if err != nil {
			return fmt.Errorf("update pending registration: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("claim registration: %w", err)
	}

	//TODO: Unsure if the DB should be building this. Maybe the service?
	return &ClaimResult{
		ServerURL:           claimedReg.HeartbeatServerURL,
		IntervalSeconds:     claimedReg.HeartbeatIntervalSeconds,
		AppBiometricEnabled: claimedReg.AppBiometricEnabled,
		AppSettingsLocked:   claimedReg.AppSettingsLocked,
		RawAPIKey:           rawAPIKey,
	}, nil
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) runInTx(ctx context.Context, fn func(*Repository) error) error {
	logger := slog.Default()
	if r.rootDB == nil {
		// We are already in a transaction. Do not nest it.
		return fn(r)
	}

	// Start the transaction
	tx, err := r.rootDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// Defer rollback (standard practice)
	defer func() {
		//nolint:staticcheck // Empty branch is intentional - ErrTxDone is expected after commit
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			// Rollback error is only significant if transaction wasn't already committed/rolled back
			logger.Error("failed to rollback transaction", slog.Any(logging.AttrKeyError, err))
		}
	}()

	// Create a COPY of the repository
	// We replace 'db' with the transaction 'tx' and set the rootDB to nil so that it is not reused
	txRepo := &Repository{
		rootDB: nil, // Prevent nested transactions
		db:     tx,  // All queries using txRepo.dbtmp will now use this transaction
	}

	// Run the business logic with the transactional repo
	if err := fn(txRepo); err != nil {
		return err
	}

	// Commit if successful
	return tx.Commit()
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) RunInTx(ctx context.Context, fn func(repository) error) error {
	return r.runInTx(ctx, func(repo *Repository) error {
		return fn(repo)
	})
}
