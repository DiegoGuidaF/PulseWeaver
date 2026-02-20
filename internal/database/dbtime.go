package database

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// DBTime is only needed for aggregate functions (MAX, MIN etc.)
// that return TEXT even for DATETIME columns.
// Regular columns declared as DATETIME scan directly into time.Time
// when using modernc.org/sqlite with _texttotime=1 in the DSN.
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
		return t.parse(v)
	case []byte:
		return t.parse(string(v))
	default:
		return fmt.Errorf("DBTime: cannot scan type %T", value)
	}
}

func (t *DBTime) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	// modernc with _time_format=sqlite writes this format.
	// Keep RFC3339 as fallback for data written by mattn/go-sqlite3 previously.
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST", // SQLite CURRENT_TIMESTAMP with microseconds and timezone
		"2006-01-02 15:04:05.999999999-07:00",     // modernc with microseconds
		"2006-01-02 15:04:05-07:00",               // _time_format=sqlite output
		"2006-01-02 15:04:05",                     // SQLite CURRENT_TIMESTAMP
		time.RFC3339,                              // legacy mattn data
	} {
		if tt, err := time.Parse(layout, s); err == nil {
			t.Time = tt
			return nil
		}
	}
	return fmt.Errorf("DBTime: cannot parse %q", s)
}

func (t DBTime) Value() (driver.Value, error) {
	return t.Time, nil
}
