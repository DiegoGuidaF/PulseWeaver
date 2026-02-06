package device

import (
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
)

type Address struct {
	ID         AddressId  `db:"id"`
	DeviceId   DeviceId   `db:"device_id"`
	IP         string     `db:"ip"`
	DisabledAt *time.Time `db:"disabled_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

func (d Address) toResponse() api.Address {
	return api.Address{
		ID:         d.ID.Int64(),
		DeviceId:   d.DeviceId.Int64(),
		IP:         d.IP,
		DisabledAt: d.DisabledAt,
		CreatedAt:  d.CreatedAt,
	}
}
