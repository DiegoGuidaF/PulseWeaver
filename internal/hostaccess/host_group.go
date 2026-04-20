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
	CreatedAt   time.Time   `db:"created_at"`
}
