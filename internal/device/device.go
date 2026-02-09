package device

import (
	"strconv"
	"time"
)

type Device struct {
	ID        DeviceID  `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type DeviceID int64

func (id DeviceID) Int64() int64 {
	return int64(id)
}

func (id DeviceID) String() string {
	return strconv.FormatInt(int64(id), 10)
}
