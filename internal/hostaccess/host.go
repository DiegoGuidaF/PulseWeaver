package hostaccess

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

// UserHostSetting is the per-user bypass flag from user_host_settings.
type UserHostSetting struct {
	UserID          ids.UserID `db:"user_id"`
	BypassHostCheck bool       `db:"bypass_host_check"`
}

// UserHostGrant is a resolved (user, fqdn) access grant.
type UserHostGrant struct {
	UserID ids.UserID `db:"user_id"`
	FQDN   string     `db:"fqdn"`
}
