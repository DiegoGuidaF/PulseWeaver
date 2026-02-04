package device

import (
	"time"
)

type DeviceIP struct {
	ID         DeviceIpID `db:"id"`
	DeviceID   DeviceID   `db:"device_id"`
	IPAddress  string     `db:"ip_address"`
	CreatedAt  time.Time  `db:"created_at"`
	DisabledAt *time.Time `db:"disabled_at"`
}
