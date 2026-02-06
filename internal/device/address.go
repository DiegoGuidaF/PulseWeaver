package device

import (
	"time"
)

type Address struct {
	ID         AddressId  `db:"id"`
	DeviceId   DeviceId   `db:"device_id"`
	IP         string     `db:"ip"`
	DisabledAt *time.Time `db:"disabled_at"`
	CreatedAt  time.Time  `db:"created_at"`
}
