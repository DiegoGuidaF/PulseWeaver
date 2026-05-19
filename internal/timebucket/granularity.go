package timebucket

import (
	"errors"
	"fmt"
)

// Granularity controls the time bucket size for aggregation queries.
type Granularity string

const (
	GranularityMinute Granularity = "minute"
	Granularity5Min   Granularity = "5min"
	GranularityHour   Granularity = "hour"
	GranularityDay    Granularity = "day"
)

// ErrInvalidGranularity is returned when a granularity value is not recognized.
var ErrInvalidGranularity = errors.New("invalid granularity: must be 'minute', '5min', 'hour', or 'day'")

// ParseGranularity validates and returns a Granularity value.
// Defaults to GranularityHour if the input is empty.
func ParseGranularity(s string) (Granularity, error) {
	switch Granularity(s) {
	case GranularityHour, "":
		return GranularityHour, nil
	case GranularityDay:
		return GranularityDay, nil
	case GranularityMinute:
		return GranularityMinute, nil
	case Granularity5Min:
		return Granularity5Min, nil
	default:
		return "", ErrInvalidGranularity
	}
}

// StrftimeISO returns the SQLite strftime format string that produces
// ISO 8601 timestamps (T separator, Z suffix).
// Only valid for hour and day granularities. For minute/5min use BucketExpr.
func (g Granularity) StrftimeISO() string {
	switch g {
	case GranularityDay:
		return "%Y-%m-%dT00:00:00Z"
	case GranularityMinute:
		return "%Y-%m-%dT%H:%M:00Z"
	default:
		return "%Y-%m-%dT%H:00:00Z"
	}
}

// BucketExpr returns a full SQL expression that truncates col to the granularity bucket.
// The returned string is safe to embed directly in a query via fmt.Sprintf — col must be
// a trusted column reference, never user input.
func (g Granularity) BucketExpr(col string) string {
	switch g {
	case GranularityDay:
		return fmt.Sprintf("strftime('%%Y-%%m-%%dT00:00:00Z', %s)", col)
	case GranularityMinute:
		return fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:%%M:00Z', %s)", col)
	case Granularity5Min:
		// SQLite has no native N-minute rounding: cast minute component to int,
		// floor-divide by 5, multiply back, then zero-pad with printf.
		return fmt.Sprintf(
			"strftime('%%Y-%%m-%%dT%%H:', %s) || printf('%%02d', (CAST(strftime('%%M', %s) AS INTEGER)/5)*5) || ':00Z'",
			col, col,
		)
	default: // GranularityHour
		return fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:00:00Z', %s)", col)
	}
}
