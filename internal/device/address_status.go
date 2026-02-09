package device

import (
	"strconv"
	"time"
)

// AddressStatus represents a status change event for an address
type AddressStatus struct {
	ID        AddressStatusId `db:"id"`
	AddressID AddressID       `db:"address_id"`
	Status    bool            `db:"status"`
	CreatedAt time.Time       `db:"created_at"`
}

type AddressStatusId int64

func (id AddressStatusId) Int64() int64 {
	return int64(id)
}

func (id AddressStatusId) String() string {
	return strconv.FormatInt(int64(id), 10)
}
