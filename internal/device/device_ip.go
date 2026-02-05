package device

import (
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
)

type DeviceIP struct {
	ID         DeviceIpID `db:"id"`
	DeviceID   DeviceID   `db:"device_id"`
	IPAddress  string     `db:"ip_address"`
	DisabledAt *time.Time `db:"disabled_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

func (d DeviceIP) toResponse() api.DeviceIP {
	return api.DeviceIP{
		Id:         d.ID.Int64(),
		DeviceId:   d.DeviceID.Int64(),
		IpAddress:  d.IPAddress,
		DisabledAt: d.DisabledAt,
		CreatedAt:  d.CreatedAt,
	}
}
