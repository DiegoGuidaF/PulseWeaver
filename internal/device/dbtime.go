package device

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// DBTime is a small helper type to scan SQLite DATETIME values.
//
// In SQLite, expressions like MAX(created_at) frequently come back as TEXT,
// which database/sql cannot scan into time.Time directly. DBTime accepts
// time.Time, string, or []byte and parses common layouts.
type DBTime struct {
	time.Time
}

func (t *DBTime) Scan(value any) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		tt, err := parseDBTime(v)
		if err != nil {
			return err
		}
		t.Time = tt
		return nil
	case []byte:
		tt, err := parseDBTime(string(v))
		if err != nil {
			return err
		}
		t.Time = tt
		return nil
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *device.DBTime", value)
	}
}

func (t DBTime) Value() (driver.Value, error) {
	// Only needed if you ever write this back to DB.
	// Use RFC3339Nano for stability.
	if t.IsZero() {
		return nil, nil
	}
	return t.Time.UTC().Format(time.RFC3339Nano), nil
}

func parseDBTime(s string) (time.Time, error) {
	// Trim common SQLite trailing/leading spaces.
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}

	// Try the most common layouts we might have in this project:
	// - SQLite CURRENT_TIMESTAMP: "2006-01-02 15:04:05"
	// - RFC3339 from Go time.Time serialization
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.999999999",
	}

	var lastErr error
	for _, layout := range layouts {
		tt, err := time.Parse(layout, s)
		if err == nil {
			return tt, nil
		}
		lastErr = err
	}
	return time.Time{}, fmt.Errorf("failed to parse time %q: %w", s, lastErr)
}
