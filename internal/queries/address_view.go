package queries

import (
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
