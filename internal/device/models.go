package device

import (
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

type Device struct {
	ID        string        `db:"id" json:"id"`
	Name      string        `db:"name" json:"name"`
	CreatedAt database.Time `db:"created_at" json:"created_at"`
}

type DeviceIP struct {
	ID         int64          `db:"id"`
	DeviceID   string         `db:"device_id"`
	IPAddress  string         `db:"ip_address"`
	CreatedAt  database.Time  `db:"created_at"`
	DisabledAt *database.Time `db:"disabled_at"`
}
