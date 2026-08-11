package timebucket

import (
	"errors"
	"fmt"
	"time"
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
	case GranularityHour, Granularity5Min:
		return "%Y-%m-%dT%H:00:00Z"
	default:
		return "%Y-%m-%dT%H:00:00Z"
	}
}

// GranularityForWindow maps a query window to the appropriate bucket size:
// ≤5m → minute, ≤1h → 5min, ≤7d → hour, >7d → day.
func GranularityForWindow(d time.Duration) Granularity {
	switch {
	case d <= 5*time.Minute:
		return GranularityMinute
	case d <= time.Hour:
		return Granularity5Min
	case d <= 7*24*time.Hour:
		return GranularityHour
	default:
		return GranularityDay
	}
}

// TruncateToBucket returns the start of t's Granularity bucket, in UTC. It
// mirrors the truncation BucketExpr performs in SQL, so a caller filling in
// missing buckets on the Go side lands on the same bucket boundaries the
// query does.
func (g Granularity) TruncateToBucket(t time.Time) time.Time {
	t = t.UTC()
	switch g {
	case GranularityDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	case GranularityMinute:
		return t.Truncate(time.Minute)
	case Granularity5Min:
		return t.Truncate(5 * time.Minute)
	case GranularityHour:
		return t.Truncate(time.Hour)
	default:
		return t.Truncate(time.Hour)
	}
}

// Next returns the start of the bucket immediately following t's bucket. t
// must already be bucket-aligned (the result of TruncateToBucket).
func (g Granularity) Next(t time.Time) time.Time {
	switch g {
	case GranularityDay:
		return t.AddDate(0, 0, 1)
	case GranularityMinute:
		return t.Add(time.Minute)
	case Granularity5Min:
		return t.Add(5 * time.Minute)
	case GranularityHour:
		return t.Add(time.Hour)
	default:
		return t.Add(time.Hour)
	}
}

// Sequence returns every bucket start from the bucket containing from
// through the bucket containing to, inclusive, oldest first, in UTC. Pair it
// with GranularityForWindow, whose ladder bounds the result (at most 168
// hourly buckets at the 7-day cutoff, daily beyond) so materializing the
// full range is safe.
func (g Granularity) Sequence(from, to time.Time) []time.Time {
	start := g.TruncateToBucket(from)
	end := g.TruncateToBucket(to)
	seq := []time.Time{}
	for b := start; !b.After(end); b = g.Next(b) {
		seq = append(seq, b)
	}
	return seq
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
	case GranularityHour:
		return fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:00:00Z', %s)", col)
	default:
		return fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:00:00Z', %s)", col)
	}
}
