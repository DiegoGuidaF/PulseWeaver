package httpapi

import (
	"encoding/json"
	"time"
)

// UTCTime wraps time.Time and always marshals to UTC RFC3339.
// It is used for all date-time fields in API response types so that UTC
// serialization is enforced at the type level — the compiler requires an
// explicit UTCTime(t) conversion in every handler, making it impossible to
// accidentally return a non-UTC timestamp.
type UTCTime time.Time

func (t UTCTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).UTC())
}

func (t *UTCTime) UnmarshalJSON(data []byte) error {
	var tt time.Time
	if err := json.Unmarshal(data, &tt); err != nil {
		return err
	}
	*t = UTCTime(tt)
	return nil
}
