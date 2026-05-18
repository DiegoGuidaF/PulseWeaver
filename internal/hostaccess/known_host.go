package hostaccess

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type KnownHost struct {
	ID        ids.KnownHostID `db:"id"`
	FQDN      string          `db:"fqdn"`
	Icon      *string         `db:"icon"`
	UpdatedAt time.Time       `db:"updated_at"`
	CreatedAt time.Time       `db:"created_at"`
}

// UserHostSetting is the per-user bypass flag from user_host_settings.
type UserHostSetting struct {
	UserID          ids.UserID `db:"user_id"`
	BypassAllowlist bool       `db:"bypass_host_allowlist"`
}

// UserHostGrant is a resolved (user, fqdn) pair from either direct or group grants.
type UserHostGrant struct {
	UserID ids.UserID `db:"user_id"`
	FQDN   string     `db:"fqdn"`
}
