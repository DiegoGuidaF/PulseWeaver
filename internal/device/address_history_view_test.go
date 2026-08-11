//go:build test

package device

import (
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/matryer/is"
)

func bucketRow(ts time.Time, risk TTLRisk, count int) addressHistoryRiskBucketRow {
	return addressHistoryRiskBucketRow{
		Timestamp:   database.DBTime{Time: ts},
		RiskRank:    slices.Index(ttlRiskOrder, risk),
		DeviceCount: count,
	}
}

func hourly(n int) []time.Time {
	base := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	seq := make([]time.Time, n)
	for i := range seq {
		seq[i] = base.Add(time.Duration(i) * time.Hour)
	}
	return seq
}

// TestFoldRiskBuckets_ZeroFillsAbsentBuckets is the direct regression guard
// for the contiguity contract: the sequence, not the query rows, decides how
// many buckets the response carries, so a bucket nothing renewed in reads as
// four explicit zeros rather than going missing.
func TestFoldRiskBuckets_ZeroFillsAbsentBuckets(t *testing.T) {
	is := is.New(t)

	seq := hourly(4)
	got := foldRiskBuckets(seq, []addressHistoryRiskBucketRow{
		bucketRow(seq[0], TTLRiskOK, 3),
		bucketRow(seq[3], TTLRiskBreached, 1),
	})

	is.Equal(len(got), 4)
	for i, b := range got {
		is.True(time.Time(b.Timestamp).Equal(seq[i])) // one bucket per sequence entry, in order
	}
	is.Equal(got[0].OkDeviceCount, 3)
	for _, b := range got[1:3] {
		is.Equal(b.OkDeviceCount, 0)
		is.Equal(b.ApproachingDeviceCount, 0)
		is.Equal(b.CriticalDeviceCount, 0)
		is.Equal(b.BreachedDeviceCount, 0)
	}
	is.Equal(got[3].BreachedDeviceCount, 1)
}

// TestFoldRiskBuckets_FoldsEveryBandIntoOneBucket is the direct regression
// guard for the one-row-per-(bucket, rank) shape the query emits: several
// rows sharing a bucket must collapse into a single response bucket carrying
// each band separately, never into several buckets with the same timestamp.
func TestFoldRiskBuckets_FoldsEveryBandIntoOneBucket(t *testing.T) {
	is := is.New(t)

	seq := hourly(1)
	got := foldRiskBuckets(seq, []addressHistoryRiskBucketRow{
		bucketRow(seq[0], TTLRiskOK, 7),
		bucketRow(seq[0], TTLRiskApproaching, 3),
		bucketRow(seq[0], TTLRiskCritical, 2),
		bucketRow(seq[0], TTLRiskBreached, 1),
	})

	is.Equal(len(got), 1)
	is.Equal(got[0].OkDeviceCount, 7)
	is.Equal(got[0].ApproachingDeviceCount, 3)
	is.Equal(got[0].CriticalDeviceCount, 2)
	is.Equal(got[0].BreachedDeviceCount, 1)
}

// TestFoldRiskBuckets_RowOrderDoesNotAffectResult guards the response
// ordering against the query's: buckets come out oldest-first because the
// sequence is, whatever order the rows arrived in.
func TestFoldRiskBuckets_RowOrderDoesNotAffectResult(t *testing.T) {
	is := is.New(t)

	seq := hourly(3)
	got := foldRiskBuckets(seq, []addressHistoryRiskBucketRow{
		bucketRow(seq[2], TTLRiskCritical, 5),
		bucketRow(seq[0], TTLRiskOK, 1),
	})

	is.Equal(len(got), 3)
	is.True(time.Time(got[0].Timestamp).Before(time.Time(got[2].Timestamp)))
	is.Equal(got[0].OkDeviceCount, 1)
	is.Equal(got[2].CriticalDeviceCount, 5)
}

// TestFoldRiskBuckets_IgnoresRowsOutsideSequence pins the behaviour when a
// row lands outside the window: it is dropped rather than appended, so the
// bucket count always matches the requested window.
func TestFoldRiskBuckets_IgnoresRowsOutsideSequence(t *testing.T) {
	is := is.New(t)

	seq := hourly(2)
	stray := seq[0].Add(-24 * time.Hour)
	got := foldRiskBuckets(seq, []addressHistoryRiskBucketRow{
		bucketRow(stray, TTLRiskBreached, 9),
		bucketRow(seq[1], TTLRiskOK, 2),
	})

	is.Equal(len(got), 2)
	for _, b := range got {
		is.True(!time.Time(b.Timestamp).Equal(stray))
		is.Equal(b.BreachedDeviceCount, 0)
	}
	is.Equal(got[1].OkDeviceCount, 2)
}

// TestFoldRiskBuckets_EmptyWindow keeps the response a non-nil empty list
// rather than a JSON null when the sequence itself is empty.
func TestFoldRiskBuckets_EmptyWindow(t *testing.T) {
	is := is.New(t)

	got := foldRiskBuckets([]time.Time{}, nil)

	is.Equal(len(got), 0)
	is.True(got != nil)
}

// TestTTLRiskOrder_CoversEveryClassifiedRisk is the drift guard ttlRiskOrder's
// contract depends on: the ranking, the buckets and the band mapping all
// index into ttlRiskOrder, so a level ttlRiskCase can emit that is missing
// from the list would rank as some other level or panic on lookup.
func TestTTLRiskOrder_CoversEveryClassifiedRisk(t *testing.T) {
	is := is.New(t)

	// Both arms count: the CASE emits its healthy level from ELSE, not THEN.
	emitted := regexp.MustCompile(`(?:THEN|ELSE) '([a-z_]+)'`).FindAllStringSubmatch(ttlRiskCase, -1)
	is.True(len(emitted) > 0) // the CASE must be parseable, or this guard proves nothing

	for _, m := range emitted {
		is.True(slices.Contains(ttlRiskOrder, TTLRisk(m[1]))) // every classified level is rankable
	}
	is.Equal(len(emitted), len(ttlRiskOrder)) // …and ttlRiskOrder carries nothing the CASE cannot emit
}

// TestTTLRiskRanks_SeverityOrder pins the two thresholds the endpoint's
// selection rules are written against: unknown sorts below ok, which the
// buckets include, which sorts below approaching, where the ranking starts.
func TestTTLRiskRanks_SeverityOrder(t *testing.T) {
	is := is.New(t)

	is.Equal(ttlRiskOrder[0], TTLRiskUnknown) // rank 0 — excluded from both buckets and ranking
	is.True(ttlRiskOKRank < ttlRiskApproachingRank)
	is.True(slices.Index(ttlRiskOrder, TTLRiskCritical) > ttlRiskApproachingRank)
	is.True(slices.Index(ttlRiskOrder, TTLRiskBreached) > slices.Index(ttlRiskOrder, TTLRiskCritical))
}

// TestTTLRiskRankExpr_MapsEveryLevel checks the generated CASE names every
// level at its ttlRiskOrder index, since the SQL rank and the Go-side band
// lookup must agree on what a rank means.
func TestTTLRiskRankExpr_MapsEveryLevel(t *testing.T) {
	is := is.New(t)

	expr := ttlRiskRankExpr("e.ttl_risk")
	matches := regexp.MustCompile(`WHEN '([a-z_]+)' THEN (\d+)`).FindAllStringSubmatch(expr, -1)

	is.Equal(len(matches), len(ttlRiskOrder))
	for i, m := range matches {
		is.Equal(m[1], string(ttlRiskOrder[i]))
		is.Equal(m[2], string(rune('0'+i))) // ranks are the list indices, in order
	}
}

// TestFoldRiskBuckets_TimestampsAreUTC guards the response timestamps against
// the DBTime zone the driver happens to scan in: the caller charts them
// against a UTC window.
func TestFoldRiskBuckets_TimestampsAreUTC(t *testing.T) {
	is := is.New(t)

	utc := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	offset := utc.In(time.FixedZone("X", 3600))
	got := foldRiskBuckets([]time.Time{utc}, []addressHistoryRiskBucketRow{
		bucketRow(offset, TTLRiskCritical, 4),
	})

	is.Equal(len(got), 1)
	is.Equal(got[0].CriticalDeviceCount, 4) // matched on the instant, not the wall clock
	is.True(time.Time(got[0].Timestamp).Equal(utc))
	is.Equal(time.Time(got[0].Timestamp).Location(), time.UTC)
}
