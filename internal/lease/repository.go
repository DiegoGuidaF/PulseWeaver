package lease

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
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
	now := time.Now().UTC()
	const query = `
		INSERT INTO address_leases (address_id, expires_at, updated_at, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(address_id, expires_at) DO UPDATE SET
			expires_at = excluded.expires_at,
			updated_at   = excluded.updated_at
		RETURNING * 
	`

	if err := r.db.GetContext(ctx, addressLease, query,
		addressLease.AddressID,
		addressLease.ExpiresAt,
		now,
		now,
	); err != nil {
		return nil, fmt.Errorf("upsert address lease: %w", err)
	}

	return addressLease, nil
}

func (r *Repository) DeleteAddressLeaseByAddressID(ctx context.Context, addressID device.AddressID) error {
	const query = `
		DELETE FROM address_leases WHERE address_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, addressID)
	if err != nil {
		return fmt.Errorf("delete address lease: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAddressLeaseNotFound
	}

	return nil
}
func (r *Repository) GetExpiredAddressIDs(ctx context.Context) ([]device.AddressID, error) {
	var addressIDs []device.AddressID
	now := time.Now().UTC()
	const query = `
		SELECT address_id FROM address_leases WHERE expires_at <= ?
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
