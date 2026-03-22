package device

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// HistoryBucket holds aggregated activity data for one time period.
// Timestamp uses DBTime because SQLite's strftime returns TEXT even for
// DATETIME columns, and DBTime handles the multi-format scanning.
type HistoryBucket struct {
	Timestamp   database.DBTime `db:"bucket"`
	ActiveCount int             `db:"active_count"`
	EventCount  int             `db:"event_count"`
}

// HistoryEvent represents a single address state change.
type HistoryEvent struct {
	Timestamp time.Time   `db:"created_at"`
	IP        string      `db:"ip"`
	IsEnabled bool        `db:"is_enabled"`
	Source    EventSource `db:"source"`
}

// AddressHistory holds the complete history response for a device.
type AddressHistory struct {
	Buckets []HistoryBucket
	Events  []HistoryEvent
}

// Granularity controls the time bucket size for history aggregation.
type Granularity string

const (
	GranularityHour Granularity = "hour"
	GranularityDay  Granularity = "day"
)

// ParseGranularity validates and returns a Granularity value.
// Defaults to GranularityHour if the input is empty.
func ParseGranularity(s string) (Granularity, error) {
	switch Granularity(s) {
	case GranularityHour, "":
		return GranularityHour, nil
	case GranularityDay:
		return GranularityDay, nil
	default:
		return "", ErrInvalidGranularity
	}
}
