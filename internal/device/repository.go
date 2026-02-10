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
	db     DBInterface
	rootDB *sqlx.DB
}

type DBInterface interface {
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
	var device Device
	query := `
		SELECT id, name, created_at
		FROM devices
		WHERE id = ?
	`
	err := r.db.GetContext(ctx, &device, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	return &device, nil
}

func (r *Repository) CreateDevice(ctx context.Context, name string) (*Device, error) {
	device := Device{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	query := `
		INSERT INTO devices (name, created_at)
		VALUES (?, ?) returning id, name, created_at
	`

	err := r.db.GetContext(ctx, &device, query, device.Name, device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return &device, nil
}

func (r *Repository) GetDevices(ctx context.Context) ([]Device, error) {
	var devices []Device

	query := `
		SELECT id, name, created_at
		FROM devices
		ORDER BY created_at DESC
	`

	if err := r.db.SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	if devices == nil {
		return []Device{}, nil
	}

	return devices, nil
}

func (r *Repository) CreateAddress(ctx context.Context, deviceId DeviceID, ipAddress string) (*Address, error) {
	address := Address{
		DeviceId:  deviceId,
		IP:        ipAddress,
		CreatedAt: time.Now().UTC(),
	}

	query := `
		INSERT INTO addresses (device_id, ip, created_at)
		VALUES (?, ?, ?) returning id, device_id, ip, created_at
	`
	var createdAddress Address
	err := r.db.GetContext(ctx, &createdAddress, query, address.DeviceId, address.IP, address.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert address: %w", err)
	}

	return &createdAddress, nil
}

func (r *Repository) GetAddressByID(ctx context.Context, id AddressID) (*Address, error) {
	var address Address
	query := `SELECT id, device_id, ip, created_at FROM addresses WHERE id = ?`
	err := r.db.GetContext(ctx, &address, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}
	return &address, nil
}

func (r *Repository) GetAddressForDeviceByIp(ctx context.Context, deviceId DeviceID, ip string) (*AddressWithStatus, error) {
	var address Address
	query := `SELECT id, device_id, ip, created_at FROM addresses WHERE device_id = ? and ip = ?`
	err := r.db.GetContext(ctx, &address, query, deviceId, ip)
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
	query := `SELECT address_id, device_id, ip, created_at, updated_at, status
			FROM address_with_status WHERE device_id = ? ORDER BY updated_at DESC`

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

func (r *Repository) GetAddressWithStatus(ctx context.Context, addressId AddressID) (*AddressWithStatus, error) {
	var addresswStatus AddressWithStatus
	query := `SELECT address_id, device_id, ip, created_at, updated_at, status
				FROM address_with_status WHERE address_id = ?`
	err := r.db.GetContext(ctx, &addresswStatus, query, addressId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address with status: %w", err)
	}
	return &addresswStatus, nil

}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) RunInTx(ctx context.Context, fn func(DeviceRepository) error) error {
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
