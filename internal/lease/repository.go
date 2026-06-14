package lease

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
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

// DeleteAddressLease removes the lease row for an address. Disabling an address
// drops its lease entirely rather than nulling the expiry, so a lease row exists
// only while the address is enabled. This keeps the device-wide expiry re-arm in
// SetDeviceAddressLeasesExpiry from resurrecting leases on already-disabled
// addresses when the lease rule is saved.
func (r *Repository) DeleteAddressLease(ctx context.Context, addressID ids.AddressID) error {
	const query = `DELETE FROM address_leases WHERE address_id = ?`
	if _, err := r.db.ExecContext(ctx, query, addressID); err != nil {
		return fmt.Errorf("delete address lease: %w", err)
	}
	return nil
}

func (r *Repository) GetExpiredAddressIDs(ctx context.Context) ([]ids.AddressID, error) {
	var addressIDs []ids.AddressID
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
		return []ids.AddressID{}, nil
	}

	return addressIDs, nil
}

func (r *Repository) SetDeviceAddressLeasesExpiry(ctx context.Context, deviceID ids.DeviceID, expiresAt *time.Time, updatedAt time.Time) error {
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
