package timebucket

import "errors"

// Granularity controls the time bucket size for aggregation queries.
type Granularity string

const (
	GranularityHour Granularity = "hour"
	GranularityDay  Granularity = "day"
)

// ErrInvalidGranularity is returned when a granularity value is not recognized.
var ErrInvalidGranularity = errors.New("invalid granularity: must be 'hour' or 'day'")

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

// StrftimeISO returns the SQLite strftime format string that produces
// ISO 8601 timestamps (T separator, Z suffix).
func (g Granularity) StrftimeISO() string {
	if g == GranularityDay {
		return "%Y-%m-%dT00:00:00Z"
	}
	return "%Y-%m-%dT%H:00:00Z"
}
