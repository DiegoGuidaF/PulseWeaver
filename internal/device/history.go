package device

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
)

// AddressEventBucket holds aggregated activity data for one time period.
// Timestamp uses DBTime because SQLite's strftime returns TEXT even for
// DATETIME columns, and DBTime handles the multi-format scanning.
type AddressEventBucket struct {
	Timestamp   database.DBTime `db:"bucket"`
	ActiveCount int             `db:"active_count"`
	EventCount  int             `db:"event_count"`
}

// AddressStateChange represents a single recorded state change from the address_events audit table.
type AddressStateChange struct {
	ID         int64       `db:"id"`
	CreatedAt  time.Time   `db:"created_at"`
	IP         string      `db:"ip"`
	IsEnabled  bool        `db:"is_enabled"`
	Source     EventSource `db:"source"`
	DeviceID   DeviceID    `db:"device_id"`
	DeviceName string      `db:"device_name"`
}

// AddressHistory holds the complete history response.
type AddressHistory struct {
	Buckets     []AddressEventBucket
	Events      []AddressStateChange
	TotalEvents int
	QueryLimit  int // effective limit used for the query, needed for cursor logic
}

// AddressHistoryQuery encapsulates all filters and pagination for history queries.
type AddressHistoryQuery struct {
	From        time.Time
	To          time.Time
	Granularity timebucket.Granularity
	DeviceIDs   []DeviceID // empty = all devices
	Source      *string
	IsEnabled   *bool
	IP          *string
	BeforeID    *int64 // cursor for events pagination
	Limit       int    // events limit (default 50, max 200)
	IncludeAll  bool   // when false (default), only state-change events are returned
}

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 200
	defaultHistoryRange = 24 * time.Hour
)

// Validate normalizes defaults and validates business rules on the query.
// Must be called before passing the query to the repository.
func (q *AddressHistoryQuery) Validate() error {
	g, err := timebucket.ParseGranularity(string(q.Granularity))
	if err != nil {
		return err
	}
	q.Granularity = g

	now := time.Now().UTC()
	if q.From.IsZero() {
		q.From = now.Add(-defaultHistoryRange)
	}
	if q.To.IsZero() {
		q.To = now
	}

	if q.Limit <= 0 {
		q.Limit = defaultHistoryLimit
	}
	if q.Limit > maxHistoryLimit {
		q.Limit = maxHistoryLimit
	}

	return nil
}
