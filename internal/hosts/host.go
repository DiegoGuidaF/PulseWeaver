package hosts

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type Host struct {
	ID        ids.HostID `db:"id"`
	FQDN      string     `db:"fqdn"`
	UpdatedAt time.Time  `db:"updated_at"`
	CreatedAt time.Time  `db:"created_at"`
}
