package useraccess

import "github.com/DiegoGuidaF/PulseWeaver/internal/ids"

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
