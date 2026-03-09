package queries

import (
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
)

type DeviceView struct {
	ID           device.DeviceID `db:"id"`
	Name         string          `db:"name"`
	CreatedAt    time.Time       `db:"created_at"`
	KeyPrefix    string          `db:"key_prefix"`
	AddressCount int             `db:"address_count"`
}
