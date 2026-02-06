package device

import (
	"time"
)

type Device struct {
	ID        DeviceId  `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
