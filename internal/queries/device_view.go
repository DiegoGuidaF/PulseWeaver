package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
)

type DeviceView struct {
	ID           device.DeviceID `db:"id"`
	Name         string          `db:"name"`
	CreatedAt    time.Time       `db:"created_at"`
	KeyPrefix    string          `db:"key_prefix"`
	AddressCount int             `db:"address_count"`
	LastSeenAt   *time.Time      `db:"last_seen_at"`
}

func (r *Repository) GetDevices(ctx context.Context) ([]DeviceView, error) {
	var devices []DeviceView

	const query = `
		SELECT
			d.id,
			d.name,
			d.created_at,
			dk.key_prefix,
			COUNT(a.id) AS address_count,
			MAX(a.updated_at) AS last_seen_at
		FROM devices d
		JOIN device_api_keys dk ON dk.device_id = d.id
		LEFT JOIN addresses a ON a.device_id = d.id AND a.is_enabled = true
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
