package lease

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/jmoiron/sqlx"
)

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Repository provides SQL-backed persistence for address leases.
type Repository struct {
	db     dBInterface
	rootDB *sqlx.DB
}

// Ensure Repository implements the lease repository interface.
var _ repository = (*Repository)(nil)

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db:     db,
		rootDB: db,
	}
}

// UpsertAddressLease creates or updates an address lease
func (r *Repository) UpsertAddressLease(ctx context.Context, addressLease *AddressLease) (*AddressLease, error) {
	const query = `
		INSERT INTO address_leases (device_id, address_id, expires_at, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(address_id) DO UPDATE SET
			device_id    = excluded.device_id,
			expires_at   = excluded.expires_at,
			updated_at   = excluded.updated_at
		RETURNING *
	`

	if err := r.db.GetContext(ctx, addressLease, query,
		addressLease.DeviceID,
		addressLease.AddressID,
		addressLease.ExpiresAt,
		addressLease.UpdatedAt,
		addressLease.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("upsert address lease: %w", err)
	}

	return addressLease, nil
}

func (r *Repository) GetExpiredAddressIDs(ctx context.Context) ([]device.AddressID, error) {
	var addressIDs []device.AddressID
	now := time.Now().UTC()
	const query = `
		SELECT address_id FROM address_leases
		WHERE expires_at IS NOT NULL AND expires_at <= ?
	`

	if err := r.db.SelectContext(ctx, &addressIDs, query,
		now,
	); err != nil {
		return nil, fmt.Errorf("get expired address leases: %w", err)
	}

	if len(addressIDs) == 0 {
		return []device.AddressID{}, nil
	}

	return addressIDs, nil
}

func (r *Repository) SetDeviceAddressLeasesExpiry(ctx context.Context, deviceID device.DeviceID, expiresAt *time.Time, updatedAt time.Time) error {
	const query = `
		UPDATE address_leases SET expires_at = ?, updated_at = ?
		WHERE device_id = ?
	`
	_, err := r.db.ExecContext(ctx, query, expiresAt, updatedAt, deviceID)
	if err != nil {
		return fmt.Errorf("set device address leases expiry: %w", err)
	}
	return nil
}
