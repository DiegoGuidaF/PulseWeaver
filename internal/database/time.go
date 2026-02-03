package database

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Time wraps time.Time to handle SQLite TEXT format
type Time struct {
	time.Time
}

// Scan implements sql.Scanner for reading from DB
func (t *Time) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}

	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return fmt.Errorf("parse time: %w", err)
	}

	t.Time = parsed
	return nil
}

// Value implements driver.Valuer for writing to DB
func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(time.RFC3339), nil
}
