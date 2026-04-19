package hostaccess

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
)

type UserAllowedHost struct {
	ID          int64       `db:"id"`
	UserID      auth.UserID `db:"user_id"`
	KnownHostID KnownHostID `db:"known_host_id"`
	CreatedAt   time.Time   `db:"created_at"`
}

type UserAllowedHostGroup struct {
	ID          int64       `db:"id"`
	UserID      auth.UserID `db:"user_id"`
	HostGroupID HostGroupID `db:"host_group_id"`
	CreatedAt   time.Time   `db:"created_at"`
}
