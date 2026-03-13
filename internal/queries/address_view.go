package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
)

type AddressView struct {
	ID        device.AddressID `db:"id"`
	DeviceID  device.DeviceID  `db:"device_id"`
	IP        string           `db:"ip"`
	IsEnabled bool             `db:"is_enabled"`
	Source    string           `db:"source"`
	UpdatedAt time.Time        `db:"updated_at"`
	CreatedAt time.Time        `db:"created_at"`
	ExpiresAt *time.Time       `db:"expires_at"`
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
