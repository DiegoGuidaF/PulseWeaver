package device

import (
	"strconv"
	"time"
)

type Address struct {
	ID        AddressID `db:"id"`
	DeviceId  DeviceID  `db:"device_id"`
	IP        string    `db:"ip"`
	CreatedAt time.Time `db:"created_at"`
}

type AddressID int64

func (id AddressID) Int64() int64 {
	return int64(id)
}

func (id AddressID) String() string {
	return strconv.FormatInt(int64(id), 10)
}
