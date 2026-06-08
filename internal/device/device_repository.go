package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

func (r *Repository) GetDevice(ctx context.Context, id ids.DeviceID) (*Device, error) {
	device := new(Device)

	query := `
			SELECT
				d.id,
				d.name,
				d.description,
				d.icon,
				d.created_at,
				d.updated_at,
				d.deleted_at,
				d.disabled_at,
				k.key_prefix,
				d.owner_id AS owner_id
			FROM devices d
			LEFT JOIN device_api_keys k ON d.id = k.device_id
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

func (r *Repository) CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error) {
	now := time.Now().UTC()
	createdDevice := new(Device)

	deviceQuery := `INSERT INTO devices (name, owner_id, description, icon, created_at) VALUES (?, ?, ?, ?, ?) RETURNING *`

	err := r.db.GetContext(ctx, createdDevice, deviceQuery, params.Name, params.OwnerID, params.Description, params.Icon, now)
	if err != nil {
		if domainErr, ok := mapDeviceNameUniqueConstraintError(err); ok {
			return nil, domainErr
		}
		if domainErr, ok := mapOwnerFKConstraintError(err); ok {
			return nil, domainErr
		}
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return createdDevice, nil
}

func (r *Repository) UpsertAPIKey(ctx context.Context, deviceID ids.DeviceID, keyHash string, keyPrefix string) error {
	query := `
		INSERT INTO device_api_keys (device_id, key_prefix, key_hash, created_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(device_id) DO UPDATE SET
			key_prefix = excluded.key_prefix,
			key_hash   = excluded.key_hash,
			created_at = CURRENT_TIMESTAMP`
	_, err := r.db.ExecContext(ctx, query, deviceID, keyPrefix, keyHash)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
			return ErrDeviceNotFound
		}
		return fmt.Errorf("upsert api key: %w", err)
	}
	return nil
}

func (r *Repository) DeleteAPIKey(ctx context.Context, deviceID ids.DeviceID) error {
	query := `DELETE FROM device_api_keys WHERE device_id = ?`
	result, err := r.db.ExecContext(ctx, query, deviceID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete api key rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNoAPIKey
	}
	return nil
}

func (r *Repository) GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error) {
	device := new(Device)

	query := `
			SELECT d.id, d.name, d.description, d.icon, d.created_at, d.updated_at, d.deleted_at,
				   d.disabled_at,
				   k.key_prefix,
				   d.owner_id AS owner_id
			FROM devices d
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

func (r *Repository) GetDeviceIDsByOwner(ctx context.Context, ownerID ids.UserID) ([]ids.DeviceID, error) {
	var deviceIDs []ids.DeviceID

	const query = `SELECT id FROM devices WHERE owner_id = ? AND deleted_at IS NULL`
	if err := r.db.SelectContext(ctx, &deviceIDs, query, ownerID); err != nil {
		return nil, fmt.Errorf("get device IDs by owner: %w", err)
	}
	return deviceIDs, nil
}

func (r *Repository) DeleteDevice(ctx context.Context, deviceID ids.DeviceID) error {
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

// SetDeviceDisabled flips a device's disabled state. When disabled is true the
// disabled_at timestamp is stamped; when false it is cleared (re-enabled).
func (r *Repository) SetDeviceDisabled(ctx context.Context, deviceID ids.DeviceID, disabled bool) error {
	var disabledAt *time.Time
	if disabled {
		now := time.Now().UTC()
		disabledAt = &now
	}
	query := `UPDATE devices SET disabled_at = ? WHERE id = ? AND deleted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, disabledAt, deviceID)
	if err != nil {
		return fmt.Errorf("set device disabled: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set device disabled rows affected: %w", err)
	}
	if rows == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (r *Repository) UpdateDevice(ctx context.Context, device *Device) (*Device, error) {
	query := `
		UPDATE devices
		SET name = ?, description = ?, icon = ?, owner_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query,
		device.Name, device.Description, device.Icon, device.OwnerID, device.ID,
	)
	if err != nil {
		if domainErr, ok := mapDeviceNameUniqueConstraintError(err); ok {
			return nil, domainErr
		}
		if domainErr, ok := mapOwnerFKConstraintError(err); ok {
			return nil, domainErr
		}
		return nil, fmt.Errorf("update device: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update device rows affected: %w", err)
	}
	if rows == 0 {
		return nil, ErrDeviceNotFound
	}
	return r.GetDevice(ctx, device.ID)
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

func mapOwnerFKConstraintError(err error) (error, bool) {
	if strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed") {
		return ErrOwnerNotFound, true
	}
	return nil, false
}
