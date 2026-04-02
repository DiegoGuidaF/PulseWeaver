package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
)

type DeviceView struct {
	ID           device.DeviceID   `db:"id"`
	Name         string            `db:"name"`
	DeviceType   device.DeviceType `db:"device_type"`
	Description  *string           `db:"description"`
	Icon         *string           `db:"icon"`
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
	KeyPrefix    string            `db:"key_prefix"`
	AddressCount int               `db:"address_count"`
	LastSeenAt   *database.DBTime  `db:"last_seen_at"`
	OwnerID      auth.UserID       `db:"owner_id"`
	OwnerName    string            `db:"owner_name"`
}

type DeviceDetail struct {
	ID          device.DeviceID   `db:"id"`
	Name        string            `db:"name"`
	DeviceType  device.DeviceType `db:"device_type"`
	Description *string           `db:"description"`
	Icon        *string           `db:"icon"`
	CreatedAt   time.Time         `db:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at"`
	DeletedAt   *time.Time        `db:"deleted_at"`
	KeyPrefix   string            `db:"key_prefix"`
	LastSeenAt  *database.DBTime  `db:"last_seen_at"`
	OwnerID     auth.UserID       `db:"owner_id"`
	OwnerName   string            `db:"owner_name"`
}

func (r *Repository) GetDevices(ctx context.Context, ownerID *auth.UserID) ([]DeviceView, error) {
	var devices []DeviceView

	query := `
		SELECT
			d.id,
			d.name,
			d.device_type,
			d.description,
			d.icon,
			d.created_at,
			d.updated_at,
			dk.key_prefix,
			COUNT(a.id) AS address_count,
			(SELECT MAX(a2.updated_at) FROM addresses a2 WHERE a2.device_id = d.id) AS last_seen_at,
			COALESCE(d.owner_id, 0) AS owner_id,
			COALESCE(u.display_name, '') AS owner_name
		FROM devices d
		JOIN device_api_keys dk ON dk.device_id = d.id
		LEFT JOIN addresses a ON a.device_id = d.id AND a.is_enabled = true
		LEFT JOIN users u ON d.owner_id = u.id
		WHERE d.deleted_at IS NULL`

	var err error
	if ownerID != nil {
		query += ` AND d.owner_id = ?`
		query += ` GROUP BY d.id, d.name, d.device_type, d.description, d.icon, d.created_at, d.updated_at, dk.key_prefix, d.owner_id ORDER BY d.updated_at DESC`
		err = r.db.SelectContext(ctx, &devices, query, *ownerID)
	} else {
		query += ` GROUP BY d.id, d.name, d.device_type, d.description, d.icon, d.created_at, d.updated_at, dk.key_prefix, d.owner_id ORDER BY d.updated_at DESC`
		err = r.db.SelectContext(ctx, &devices, query)
	}
	if err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}

	if devices == nil {
		return []DeviceView{}, nil
	}

	return devices, nil
}

func (r *Repository) GetDevicesByUser(ctx context.Context, targetUserID auth.UserID) ([]DeviceView, error) {
	return r.GetDevices(ctx, &targetUserID)
}

func (r *Repository) GetDeviceDetail(ctx context.Context, id device.DeviceID) (*DeviceDetail, error) {
	detail := new(DeviceDetail)

	query := `
		SELECT
		    d.id, d.name, d.device_type, d.description, d.icon, d.created_at, d.updated_at, d.deleted_at,
		    k.key_prefix,
		    (SELECT MAX(a.updated_at) FROM addresses a WHERE a.device_id = d.id) AS last_seen_at,
		    COALESCE(d.owner_id, 0) AS owner_id,
		    COALESCE(u.display_name, '') AS owner_name
		FROM devices d
		INNER JOIN device_api_keys k ON d.id = k.device_id
		LEFT JOIN users u ON d.owner_id = u.id
		WHERE d.id = ? AND d.deleted_at IS NULL
	`

	err := r.db.GetContext(ctx, detail, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, device.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("get device detail: %w", err)
	}

	return detail, nil
}
