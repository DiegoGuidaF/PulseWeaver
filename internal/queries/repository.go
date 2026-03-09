package queries

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DeviceExists(ctx context.Context, deviceID device.DeviceID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1
			FROM devices d
			WHERE d.id = ? AND d.deleted_at IS NULL
		)
	`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, deviceID); err != nil {
		return false, fmt.Errorf("check device existence: %w", err)
	}

	return exists, nil
}

func (r *Repository) GetDeviceAddresses(ctx context.Context, deviceID device.DeviceID) ([]AddressView, error) {
	var addresses []AddressView

	const query = `
		SELECT
			a.id,
			a.device_id,
			a.ip,
			acs.is_enabled,
			acs.source,
			acs.updated_at,
			a.created_at,
			al.expires_at
		FROM addresses a
		JOIN address_current_state acs ON acs.address_id = a.id
		LEFT JOIN address_leases al ON al.address_id = a.id
		WHERE a.device_id = ?
		ORDER BY a.created_at DESC
	`

	if err := r.db.SelectContext(ctx, &addresses, query, deviceID); err != nil {
		return nil, fmt.Errorf("get device addresses: %w", err)
	}

	if addresses == nil {
		return []AddressView{}, nil
	}

	return addresses, nil
}

func (r *Repository) GetDevices(ctx context.Context) ([]DeviceView, error) {
	var devices []DeviceView

	const query = `
		SELECT
			d.id,
			d.name,
			d.created_at,
			dk.key_prefix,
			COUNT(acs.address_id) AS address_count
		FROM devices d
		JOIN device_api_keys dk ON dk.device_id = d.id
		LEFT JOIN addresses a ON a.device_id = d.id
		LEFT JOIN address_current_state acs ON acs.address_id = a.id AND acs.is_enabled = true
		WHERE d.deleted_at IS NULL
		GROUP BY d.id, d.name, d.created_at, dk.key_prefix
		ORDER BY d.created_at DESC
	`

	if err := r.db.SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}

	if devices == nil {
		return []DeviceView{}, nil
	}

	return devices, nil
}
