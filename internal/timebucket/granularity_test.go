package timebucket_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/timebucket"
	"github.com/matryer/is"
)

func TestGranularity_TruncateToBucket(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		name string
		g    timebucket.Granularity
		in   time.Time
		want time.Time
	}{
		{"minute", timebucket.GranularityMinute, time.Date(2026, 8, 11, 14, 37, 42, 0, time.UTC), time.Date(2026, 8, 11, 14, 37, 0, 0, time.UTC)},
		{"5min floor", timebucket.Granularity5Min, time.Date(2026, 8, 11, 14, 37, 42, 0, time.UTC), time.Date(2026, 8, 11, 14, 35, 0, 0, time.UTC)},
		{"5min exact", timebucket.Granularity5Min, time.Date(2026, 8, 11, 14, 35, 0, 0, time.UTC), time.Date(2026, 8, 11, 14, 35, 0, 0, time.UTC)},
		{"hour", timebucket.GranularityHour, time.Date(2026, 8, 11, 14, 37, 42, 0, time.UTC), time.Date(2026, 8, 11, 14, 0, 0, 0, time.UTC)},
		{"day", timebucket.GranularityDay, time.Date(2026, 8, 11, 14, 37, 42, 0, time.UTC), time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)},
		{"non-UTC input normalized", timebucket.GranularityHour, time.Date(2026, 8, 11, 14, 37, 0, 0, time.FixedZone("X", 3600)), time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			got := tc.g.TruncateToBucket(tc.in)
			is.True(got.Equal(tc.want))
			is.Equal(got.Location(), time.UTC)
		})
	}
}

// TestGranularityForWindow_LadderBoundaries pins the ladder at each cutoff,
// since Sequence materialises one entry per bucket: a ladder that picked a
// finer granularity than intended would blow the bucket count up for the same
// window.
func TestGranularityForWindow_LadderBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		window time.Duration
		want   timebucket.Granularity
	}{
		{"below the minute cutoff", time.Minute, timebucket.GranularityMinute},
		{"at the minute cutoff", 5 * time.Minute, timebucket.GranularityMinute},
		{"just past the minute cutoff", 5*time.Minute + time.Nanosecond, timebucket.Granularity5Min},
		{"at the 5min cutoff", time.Hour, timebucket.Granularity5Min},
		{"just past the 5min cutoff", time.Hour + time.Nanosecond, timebucket.GranularityHour},
		{"at the hour cutoff", 7 * 24 * time.Hour, timebucket.GranularityHour},
		{"just past the hour cutoff", 7*24*time.Hour + time.Nanosecond, timebucket.GranularityDay},
		{"a year", 365 * 24 * time.Hour, timebucket.GranularityDay},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(timebucket.GranularityForWindow(tc.window), tc.want)
		})
	}
}

// TestGranularity_Sequence_BoundedByLadder is the guard behind Sequence's
// "materialising the full range is safe" claim: pairing it with
// GranularityForWindow keeps the bucket count small at every cutoff, so no
// window can produce an unbounded response.
func TestGranularity_Sequence_BoundedByLadder(t *testing.T) {
	from := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		window   time.Duration
		maxCount int
	}{
		{"5 minutes", 5 * time.Minute, 6},         // minute buckets
		{"1 hour", time.Hour, 13},                 // 5-minute buckets
		{"7 days", 7 * 24 * time.Hour, 169},       // hourly buckets
		{"1 year", 365 * 24 * time.Hour, 366 + 1}, // daily buckets
		{"10 years", 3650 * 24 * time.Hour, 3653}, // still daily — the ladder tops out
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			g := timebucket.GranularityForWindow(tc.window)
			seq := g.Sequence(from, from.Add(tc.window))
			is.True(len(seq) <= tc.maxCount)
			is.True(len(seq) > 0)
			is.True(seq[0].Equal(g.TruncateToBucket(from)))
			is.True(seq[len(seq)-1].Equal(g.TruncateToBucket(from.Add(tc.window))))
		})
	}
}

// TestGranularity_Sequence_InvertedRangeIsEmpty pins the degenerate case: a
// to before from yields no buckets rather than looping.
func TestGranularity_Sequence_InvertedRangeIsEmpty(t *testing.T) {
	is := is.New(t)

	from := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	seq := timebucket.GranularityHour.Sequence(from, from.Add(-3*time.Hour))

	is.Equal(len(seq), 0)
	is.True(seq != nil) // an empty list, never a nil the caller would marshal as null
}

func TestGranularity_Sequence(t *testing.T) {
	is := is.New(t)

	from := time.Date(2026, 8, 11, 10, 15, 0, 0, time.UTC)
	to := time.Date(2026, 8, 11, 12, 45, 0, 0, time.UTC)
	seq := timebucket.GranularityHour.Sequence(from, to)

	is.Equal(len(seq), 3) // 10:00, 11:00, 12:00
	is.True(seq[0].Equal(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)))
	is.True(seq[1].Equal(time.Date(2026, 8, 11, 11, 0, 0, 0, time.UTC)))
	is.True(seq[2].Equal(time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)))
}

func TestGranularity_Sequence_SingleBucket(t *testing.T) {
	is := is.New(t)

	from := time.Date(2026, 8, 11, 10, 15, 0, 0, time.UTC)
	to := time.Date(2026, 8, 11, 10, 45, 0, 0, time.UTC)
	seq := timebucket.GranularityHour.Sequence(from, to)

	is.Equal(len(seq), 1)
	is.True(seq[0].Equal(time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)))
}

func TestGranularity_Sequence_DaySpansMonthBoundary(t *testing.T) {
	is := is.New(t)

	from := time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	seq := timebucket.GranularityDay.Sequence(from, to)

	is.Equal(len(seq), 3) // Jan 30, Jan 31, Feb 1
	is.True(seq[2].Equal(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
}
