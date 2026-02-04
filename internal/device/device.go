package device

import (
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
)

type Device struct {
	ID        DeviceID  `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (d Device) toResponse() api.Device {
	return api.Device{
		Id:        d.ID.Int64(),
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
	}
}
