package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

type Repository struct {
	db *database.SQLite
}

func NewRepository(db *database.SQLite) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDeviceByID(ctx context.Context, id DeviceId) (*Device, error) {
	var device Device
	query := `
		SELECT id, name, created_at
		FROM devices
		WHERE id = ?
	`
	err := r.db.DB().GetContext(ctx, &device, query, id)
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
		VALUES (?, ?)
	`

	result, err := r.db.DB().ExecContext(ctx, query, device.Name, device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return r.GetDeviceByID(ctx, DeviceId(id))
}

func (r *Repository) GetDevices(ctx context.Context) ([]Device, error) {
	var devices []Device

	query := `
		SELECT id, name, created_at
		FROM devices
		ORDER BY created_at DESC
	`

	if err := r.db.DB().SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	if devices == nil {
		return []Device{}, nil
	}

	return devices, nil
}

func (r *Repository) GetAddressByDeviceAndIP(ctx context.Context, deviceId DeviceId, ipAddress string) (*Address, error) {
	var address Address
	query := `SELECT id, device_id, ip, created_at, disabled_at FROM addresses WHERE device_id = ? AND ip = ?`
	err := r.db.DB().GetContext(ctx, &address, query, deviceId, ipAddress)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}
	return &address, nil
}

func (r *Repository) CreateAddress(ctx context.Context, deviceId DeviceId, ipAddress string) (*Address, error) {
	address, _, err := r.CreateAddressWithNew(ctx, deviceId, ipAddress)
	return address, err
}

func (r *Repository) CreateAddressWithNew(ctx context.Context, deviceId DeviceId, ipAddress string) (*Address, bool, error) {
	// Try to insert first
	deviceIP := Address{
		DeviceId:  deviceId,
		IP:        ipAddress,
		CreatedAt: time.Now().UTC(),
	}

	query := `
		INSERT INTO addresses (device_id, ip, created_at)
		VALUES (?, ?, ?)
	`
	result, err := r.db.DB().ExecContext(ctx, query, deviceIP.DeviceId, deviceIP.IP, deviceIP.CreatedAt)
	if err != nil {
		// Check if it's a unique constraint violation
		errStr := err.Error()
		if strings.Contains(errStr, "UNIQUE constraint failed") || strings.Contains(errStr, "UNIQUE constraint") {
			// Address already exists, handle upsert logic
			existing, findErr := r.GetAddressByDeviceAndIP(ctx, deviceId, ipAddress)
			if findErr != nil {
				return nil, false, fmt.Errorf("failed to find existing address: %w", findErr)
			}

			// If address exists and is enabled, return it (not new)
			if existing.DisabledAt == nil {
				return existing, false, nil
			}

			// If address exists and is disabled, re-enable it (not new)
			updateQuery := `UPDATE addresses SET disabled_at = NULL WHERE id = ?`
			_, updateErr := r.db.DB().ExecContext(ctx, updateQuery, existing.ID)
			if updateErr != nil {
				return nil, false, fmt.Errorf("failed to re-enable address: %w", updateErr)
			}

			// Return the updated address (not new)
			updated, err := r.GetAddressByID(ctx, existing.ID)
			return updated, false, err
		}
		return nil, false, fmt.Errorf("failed to create device address: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get last insert id: %w", err)
	}

	address, err := r.GetAddressByID(ctx, AddressId(id))
	return address, true, err
}

func (r *Repository) GetAddressByID(ctx context.Context, id AddressId) (*Address, error) {
	var address Address
	query := `SELECT id, device_id, ip, created_at, disabled_at FROM addresses WHERE id = ?`
	err := r.db.DB().GetContext(ctx, &address, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAddressNotFound
		}
		return nil, fmt.Errorf("failed to get device address: %w", err)
	}
	return &address, nil
}

func (r *Repository) ListActiveAddresses(ctx context.Context, deviceId DeviceId) ([]Address, error) {
	var addresses []Address
	query := `
		SELECT id, device_id, ip, created_at, disabled_at 
		FROM addresses 
		WHERE device_id = ? AND disabled_at IS NULL
		ORDER BY created_at DESC
	`
	err := r.db.DB().SelectContext(ctx, &addresses, query, deviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to list device addresses: %w", err)
	}

	if addresses == nil {
		return []Address{}, nil
	}

	return addresses, nil
}

func (r *Repository) DisableAddress(ctx context.Context, deviceId DeviceId, addressId AddressId) (*Address, error) {
	query := `UPDATE addresses SET disabled_at = CURRENT_TIMESTAMP 
        		WHERE id = ? AND device_id = ? AND disabled_at IS NULL 
        		RETURNING id, device_id, ip, created_at, disabled_at`
	var address Address
	err := r.db.DB().GetContext(ctx, &address, query, addressId, deviceId)
	if err == nil {
		return &address, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAddressNotFound
	}

	return nil, fmt.Errorf("unexpected state during disable operation: %w", err)
}
