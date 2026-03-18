//go:build test

package httpapi_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/matryer/is"
)

func TestUTCTime_MarshalJSON_NonUTCTimeConvertedToUTC(t *testing.T) {
	is := is.New(t)

	loc, err := time.LoadLocation("America/New_York")
	is.NoErr(err)
	local := time.Date(2024, 6, 15, 12, 0, 0, 0, loc) // noon New York = 16:00 UTC

	ut := httpapi.UTCTime(local)
	data, err := json.Marshal(ut)
	is.NoErr(err)

	var decoded time.Time
	err = json.Unmarshal(data, &decoded)
	is.NoErr(err)
	is.Equal(decoded.UTC(), local.UTC())
	is.Equal(decoded.Location().String(), "UTC")
}

func TestUTCTime_MarshalJSON_AlreadyUTCUnchanged(t *testing.T) {
	is := is.New(t)

	original := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ut := httpapi.UTCTime(original)
	data, err := json.Marshal(ut)
	is.NoErr(err)

	var decoded time.Time
	err = json.Unmarshal(data, &decoded)
	is.NoErr(err)
	is.Equal(decoded.UTC(), original)
}

func TestUTCTime_UnmarshalJSON_RoundTrip(t *testing.T) {
	is := is.New(t)

	original := time.Date(2025, 3, 18, 10, 30, 0, 0, time.UTC)
	data, err := json.Marshal(httpapi.UTCTime(original))
	is.NoErr(err)

	var ut httpapi.UTCTime
	err = json.Unmarshal(data, &ut)
	is.NoErr(err)
	is.Equal(time.Time(ut).UTC(), original)
}
