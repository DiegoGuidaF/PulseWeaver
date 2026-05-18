package hosts

import "time"

type IgnoredHostSuggestion struct {
	ID        int64     `db:"id"`
	FQDN      string    `db:"fqdn"`
	CreatedAt time.Time `db:"created_at"`
}
