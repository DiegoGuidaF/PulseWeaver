package device

import (
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

// AddressWithStatus combines address with its latest status via a DB View
type AddressWithStatus struct {
	Id        AddressID       `db:"id"`
	DeviceId  DeviceID        `db:"device_id"`
	IP        string          `db:"ip"`
	Status    bool            `db:"status"`
	CreatedAt time.Time       `db:"created_at"`
	UpdatedAt database.DBTime `db:"updated_at"` // Uses custom date format to fix sqlite issues with type fn result in a view
}
