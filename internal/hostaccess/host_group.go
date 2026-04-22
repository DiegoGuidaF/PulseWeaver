package hostaccess

import (
	"strconv"
	"time"
)

type HostGroupID int64

func (id HostGroupID) Int64() int64   { return int64(id) }
func (id HostGroupID) String() string { return strconv.FormatInt(int64(id), 10) }

type HostGroup struct {
	ID          HostGroupID `db:"id"`
	Name        string      `db:"name"`
	Description *string     `db:"description"`
	Icon        *string     `db:"icon"`
	UpdatedAt   time.Time   `db:"updated_at"`
	CreatedAt   time.Time   `db:"created_at"`
}

// KnownHostRef is a lightweight reference returned inside group member lists.
type KnownHostRef struct {
	ID   KnownHostID
	FQDN string
	Icon *string
}
