package lease

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
)

// Repository provides SQL-backed persistence for address leases.
type Repository struct {
	db *database.DB
}

// NewRepository creates a new Repository backed by the given sqlx.DB.
func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db: db,
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

	created := new(AddressLease)
	if err := r.db.GetContext(ctx, created, query,
		addressLease.DeviceID,
		addressLease.AddressID,
		addressLease.ExpiresAt,
		addressLease.UpdatedAt,
		addressLease.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("upsert address lease: %w", err)
	}

	return created, nil
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
