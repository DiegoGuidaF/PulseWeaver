package device

import (
	"time"
)

type Device struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type DeviceIP struct {
	ID         int64      `db:"id"`
	DeviceID   string     `db:"device_id"`
	IPAddress  string     `db:"ip_address"`
	CreatedAt  time.Time  `db:"created_at"`
	DisabledAt *time.Time `db:"disabled_at"`
}
