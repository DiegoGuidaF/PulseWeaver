package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/logging"
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

func (r *Repository) GetDevice(ctx context.Context, id DeviceID) (*Device, error) {
	device := &Device{}

	query := `
		SELECT 
		    d.id,
			d.name,
			d.created_at,
			d.deleted_at,
			k.key_prefix
		FROM devices d
        INNER JOIN device_api_keys k ON d.id = k.device_id
		WHERE d.id = ? AND d.deleted_at IS NULL`

	err := r.db.GetContext(ctx, device, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	return device, nil
}

func (r *Repository) CreateDevice(ctx context.Context, params *CreateDeviceParams) (*Device, error) {
	now := time.Now().UTC()
	var createdDevice *Device

	err := r.runInTx(ctx, func(tx *Repository) error {
		// Create device
		deviceQuery := `
		INSERT INTO devices (name, created_at)
		VALUES (?, ?) RETURNING id
		`
		var createdDeviceID DeviceID
		err := tx.db.GetContext(ctx, &createdDeviceID, deviceQuery, params.Name, now)
		if err != nil {
			if domainErr, ok := mapDeviceNameUniqueConstraintError(err); ok {
				return domainErr
			}
			return fmt.Errorf("insert device: %w", err)
		}

		// Add API KEY to device
		apiQuery := `
		INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at)
		VALUES (?, ?, ?, ?)
	`

		_, err = tx.db.ExecContext(ctx, apiQuery, createdDeviceID, params.KeyPrefix, params.KeyHash, now)
		if err != nil {
			return err
		}

		createdDevice, err = tx.GetDevice(ctx, createdDeviceID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdDevice, nil
}

func (r *Repository) GetDevices(ctx context.Context) ([]Device, error) {
	var devices []Device

	query := `
		SELECT 
			d.id,
			d.name,
			d.created_at,
			d.deleted_at,
			k.key_prefix
		FROM devices d
		INNER JOIN device_api_keys k ON d.id = k.device_id
		WHERE d.deleted_at IS NULL
		ORDER BY d.created_at DESC
	`

	if err := r.db.SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	if devices == nil {
		return []Device{}, nil
	}

	return devices, nil
}

func (r *Repository) CreateAddress(ctx context.Context, params *CreateAddressParams) (*Address, error) {
	var address *Address

	err := r.runInTx(ctx, func(tx *Repository) error {
		query := `
		INSERT INTO addresses (device_id, ip, created_at)
		VALUES (?, ?, ?) RETURNING id
	`
		var addressID AddressID
		err := tx.db.GetContext(ctx, &addressID, query, params.DeviceID, params.IP, time.Now().UTC())
		if err != nil {
			return err
		}

		address, err = tx.recordStatusChange(ctx, addressID, true, StatusSourceManual)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}
	return address, nil
}

func (r *Repository) GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip string) (*Address, error) {
	address := &Address{}

	query := `
		SELECT a.id,
		       a.device_id,
		       a.ip,
		       ac.is_enabled,
		       ac.source,
		       a.created_at,
		       ac.updated_at
		FROM addresses a
		INNER JOIN address_current_state ac ON a.id = ac.address_id
		WHERE a.device_id = ?
		and a.ip = ?
		ORDER BY ac.updated_at DESC
	`

	err := r.db.GetContext(ctx, address, query, deviceID, ip)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}

	return address, nil
}

func (r *Repository) ListAddresses(ctx context.Context, deviceID DeviceID) ([]Address, error) {
	var addresses []Address

	query := `
		SELECT a.id,
		       a.device_id,
		       a.ip,
		       ac.is_enabled,
		       ac.source,
		       a.created_at,
		       ac.updated_at
		FROM addresses a
		INNER JOIN address_current_state ac ON a.id = ac.address_id
		WHERE a.device_id = ?
		ORDER BY ac.updated_at DESC
	`

	err := r.db.SelectContext(ctx, &addresses, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list device addresses: %w", err)
	}

	if addresses == nil {
		return []Address{}, nil
	}

	return addresses, nil
}

func (r *Repository) CheckAddressOwnership(ctx context.Context, deviceID DeviceID, addressID AddressID) error {
	var dummy int

	query := `SELECT 1 FROM addresses WHERE id = ? AND device_id = ?`

	err := r.db.GetContext(ctx, &dummy, query, addressID, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAddressNotOwnedByDevice
		}
		return fmt.Errorf("failed to check address ownership: %w", err)
	}
	return nil
}

func (r *Repository) GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error) {
	device := &Device{}

	query := `
		SELECT d.* FROM devices d
		INNER JOIN device_api_keys k ON d.id = k.device_id
		WHERE k.key_hash = ? AND d.deleted_at IS NULL
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

func mapDeviceNameUniqueConstraintError(err error) (error, bool) {
	message := strings.ToLower(err.Error())
	if !strings.Contains(message, "unique constraint failed") {
		return nil, false
	}
	if strings.Contains(message, "name") || strings.Contains(message, "idx_devices_name_active") {
		return ErrDuplicateDeviceName, true
	}
	return nil, false
}

func (r *Repository) DeleteDevice(ctx context.Context, deviceID DeviceID) error {
	query := `UPDATE devices SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, time.Now().UTC(), deviceID)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete device check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *Repository) GetEnabledUniqueIPs(ctx context.Context) ([]string, error) {
	var ips []string

	query := `
		SELECT DISTINCT a.ip
		FROM addresses a
		INNER JOIN address_current_state ac ON a.id = ac.address_id
		WHERE ac.is_enabled = 1
	`

	err := r.db.SelectContext(ctx, &ips, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled unique IPs: %w", err)
	}

	if ips == nil {
		return []string{}, nil
	}

	return ips, nil
}

// GetAddress returns the current state for a single address ID.
func (r *Repository) GetAddress(ctx context.Context, addressID AddressID) (*Address, error) {
	state := &Address{}

	query := `
		SELECT a.id,
		       a.device_id,
		       a.ip,
		       ac.is_enabled,
		       ac.source,
		       a.created_at,
		       ac.updated_at
		FROM addresses a
		INNER JOIN address_current_state ac ON a.id = ac.address_id
		WHERE a.id = ?
	`

	err := r.db.GetContext(ctx, state, query, addressID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get address current state: %w", err)
	}

	return state, nil
}

func (r *Repository) DisableAddress(ctx context.Context, addressID AddressID) (*Address, error) {
	return r.recordStatusChange(ctx, addressID, false, StatusSourceManual)
}

func (r *Repository) DisableAddresses(ctx context.Context, addressIDs []AddressID, source StatusSource) ([]Address, error) {
	if len(addressIDs) == 0 {
		return []Address{}, nil
	}

	disabledAddresses := make([]Address, len(addressIDs))

	err := r.runInTx(ctx, func(tx *Repository) error {
		for i, addressID := range addressIDs {
			disabledAddress, err := tx.recordStatusChange(ctx, addressID, false, source)
			if err != nil {
				return fmt.Errorf("failed to disable address %d: %w", addressID, err)
			}
			disabledAddresses[i] = *disabledAddress
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return disabledAddresses, nil
}

func (r *Repository) EnableAddress(ctx context.Context, addressID AddressID, source StatusSource) (*Address, error) {
	return r.recordStatusChange(ctx, addressID, true, source)
}
func (r *Repository) recordStatusChange(ctx context.Context, addressID AddressID, isEnabled bool, source StatusSource) (*Address, error) {
	var finalAddress *Address
	err := r.runInTx(ctx, func(tx *Repository) error {
		now := time.Now().UTC()

		insertStatus := `
		INSERT INTO address_status (address_id, status, source, created_at)
		VALUES (?, ?, ?, ?)
	`

		if _, err := tx.db.ExecContext(ctx, insertStatus, addressID, isEnabled, source, now); err != nil {
			return fmt.Errorf("failed to record status: %w", err)
		}

		upsertState := `
		INSERT INTO address_current_state (address_id, is_enabled, source, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(address_id) DO UPDATE SET
			is_enabled = excluded.is_enabled,
			source     = excluded.source,
			updated_at = excluded.updated_at
	`

		if _, err := tx.db.ExecContext(ctx, upsertState, addressID, isEnabled, source, now); err != nil {
			return fmt.Errorf("failed to upsert address current state: %w", err)
		}

		var err error
		finalAddress, err = tx.GetAddress(ctx, addressID)
		if err != nil {
			return fmt.Errorf("failed to get address current state: %w", err)

		}

		return nil

	})
	if err != nil {
		return nil, err
	}

	return finalAddress, nil
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
