package hostaccess

import (
	"strconv"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
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

// UserHostSetting is the per-user bypass flag from user_host_settings.
type UserHostSetting struct {
	UserID          auth.UserID `db:"user_id"`
	BypassAllowlist bool        `db:"bypass_host_allowlist"`
}

// UserHostGrant is a resolved (user, fqdn) pair from either direct or group grants.
type UserHostGrant struct {
	UserID auth.UserID `db:"user_id"`
	FQDN   string      `db:"fqdn"`
}
