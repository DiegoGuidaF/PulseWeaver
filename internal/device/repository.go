package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db     dBInterface
	rootDB *sqlx.DB
}

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		rootDB: db,
		db:     db,
	}
}

func (r *Repository) GetDeviceByID(ctx context.Context, id DeviceID) (*Device, error) {
	device := &Device{}

	query := `SELECT * FROM devices WHERE id = ?`

	err := r.db.GetContext(ctx, device, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return device, nil
}

func (r *Repository) CreateDevice(ctx context.Context, device *Device) (*Device, error) {
	query := `
		INSERT INTO devices (name, created_at)
		VALUES (?, ?) returning *
	`

	err := r.db.GetContext(ctx, device, query, device.Name, device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return device, nil
}

func (r *Repository) GetDevices(ctx context.Context) ([]DeviceWithApiKeyPrefix, error) {
	var devices []DeviceWithApiKeyPrefix

	query := `
		SELECT d.id, d.name, d.created_at, k.key_prefix
		FROM devices d
		INNER JOIN device_api_keys k ON d.id = k.device_id
		ORDER BY d.created_at DESC
	`

	if err := r.db.SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	if devices == nil {
		return []DeviceWithApiKeyPrefix{}, nil
	}

	return devices, nil
}

func (r *Repository) CreateAddress(ctx context.Context, address *Address) (*Address, error) {
	query := `
		INSERT INTO addresses (device_id, ip, created_at)
		VALUES (?, ?, ?) returning *
	`

	err := r.db.GetContext(ctx, address, query, address.DeviceId, address.IP, address.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert address: %w", err)
	}

	return address, nil
}

func (r *Repository) GetAddressByID(ctx context.Context, id AddressID) (*Address, error) {
	address := &Address{}

	query := `SELECT * FROM addresses WHERE id = ?`

	err := r.db.GetContext(ctx, address, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}

	return address, nil
}

func (r *Repository) GetAddressForDeviceByIp(ctx context.Context, deviceId DeviceID, ip string) (*AddressWithStatus, error) {
	address := &Address{}

	query := `SELECT * FROM addresses WHERE device_id = ? and ip = ?`

	err := r.db.GetContext(ctx, address, query, deviceId, ip)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}

	return r.GetAddressWithStatus(ctx, address.ID)
}

func (r *Repository) ListAddresses(ctx context.Context, deviceId DeviceID) ([]AddressWithStatus, error) {
	var addresses []AddressWithStatus

	query := `SELECT * FROM address_with_status WHERE device_id = ? ORDER BY updated_at DESC`

	err := r.db.SelectContext(ctx, &addresses, query, deviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to list device addresses: %w", err)
	}

	if addresses == nil {
		return []AddressWithStatus{}, nil
	}

	return addresses, nil
}

func (r *Repository) DisableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error) {
	// Validate that the address belongs to the device
	return r.setAddressStatus(ctx, addressId, false)
}

func (r *Repository) EnableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error) {
	return r.setAddressStatus(ctx, addressId, true)
}

func (r *Repository) setAddressStatus(ctx context.Context, addressId AddressID, isEnabled bool) (*AddressWithStatus, error) {
	query := `
		INSERT INTO address_status (address_id, status, created_at)
		VALUES (?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, addressId, isEnabled, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to record status: %w", err)
	}
	return r.GetAddressWithStatus(ctx, addressId)
}

func (r *Repository) CheckAddressOwnership(ctx context.Context, deviceId DeviceID, addressId AddressID) error {
	var dummy int

	query := `SELECT 1 FROM addresses WHERE id = ? AND device_id = ?`

	err := r.db.GetContext(ctx, &dummy, query, addressId, deviceId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAddressNotOwnedByDevice
		}
		return fmt.Errorf("failed to check address ownership: %w", err)
	}
	return nil
}

func (r *Repository) GetAddressWithStatus(ctx context.Context, id AddressID) (*AddressWithStatus, error) {
	addresswStatus := &AddressWithStatus{}

	query := `SELECT * FROM address_with_status WHERE id = ?`

	err := r.db.GetContext(ctx, addresswStatus, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address with status: %w", err)
	}

	return addresswStatus, nil

}

func (r *Repository) CreateDeviceApiKey(ctx context.Context, apiKey *ApiKey) (*ApiKey, error) {
	query := `
		INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at)
		VALUES (?, ?, ?, ?) returning *
	`

	err := r.db.GetContext(ctx, apiKey, query, apiKey.DeviceID, apiKey.KeyPrefix, apiKey.KeyHash, apiKey.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}

	return apiKey, nil
}

func (r *Repository) GetDeviceByApiKeyHash(ctx context.Context, keyHash string) (*Device, error) {
	device := &Device{}

	query := `
		SELECT d.* FROM devices d
		INNER JOIN device_api_keys k ON d.id = k.device_id
		WHERE k.key_hash = ?
	`

	err := r.db.GetContext(ctx, device, query, keyHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device by api key hash: %w", err)
	}

	return device, nil
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) RunInTx(ctx context.Context, fn func(repository) error) error {
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
	defer tx.Rollback()

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
