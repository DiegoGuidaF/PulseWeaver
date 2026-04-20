package hostaccess

import (
	"strconv"
	"time"
)

type KnownHostID int64

func (id KnownHostID) Int64() int64   { return int64(id) }
func (id KnownHostID) String() string { return strconv.FormatInt(int64(id), 10) }

type KnownHost struct {
	ID        KnownHostID `db:"id"`
	FQDN      string      `db:"fqdn"`
	Icon      *string     `db:"icon"`
	UpdatedAt time.Time   `db:"updated_at"`
	CreatedAt time.Time   `db:"created_at"`
}
