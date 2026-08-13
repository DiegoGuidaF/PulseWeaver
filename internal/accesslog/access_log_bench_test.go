//go:build test

package accesslog_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/filterx"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

// The access-log page issues two reads per render, and this file measures both:
// ListAccessLog for the table, GetAccessLogHistogram for the chart above it. They
// answer the same filter set but scale on opposite levers — the table's cost rides
// its sort, the chart's rides its window — so each benchmark varies its own.
//
// Fixtures are hand-built and owned here (never the shared seeder — see the note in
// internal/policy/bench_test.go); they are written through the production BatchInsert
// path so the on-disk shape matches what verify-ip actually persists.
//
//	go test -tags=test -run=^$ -bench=AccessLog -benchmem ./internal/accesslog/
//
// Do not commit raw ns/op numbers — record before/after deltas in the commit message.

var benchCountries = []string{"US", "DE", "FR", "GB", "JP", "BR", "IN", "AU"}

// benchAccessEvent builds one varied access-log row at the given instant: a distinct
// client IP, a spread of durations and outcomes, and partial geoip coverage — so
// non-time sorts must actually reorder the window rather than read it back in
// insertion order, and country filters match only part of it.
func benchAccessEvent(i int, at time.Time) policy.DecisionEvent {
	ev := policy.DecisionEvent{
		ClientIP:   fmt.Sprintf("198.51.%d.%d", (i>>8)&0xff, i&0xff),
		Outcome:    i%4 != 0,
		CreatedAt:  at,
		DurationUs: int64((i * 7919) % 250_000),
		Headers:    map[string][]string{},
	}
	if i%4 == 0 {
		ev.DenyReason = new(policy.DenyReasonIPNotRegistered)
	}
	// Two-thirds carry geoip, mirroring the partial coverage of real traffic.
	if i%3 != 0 {
		cc := benchCountries[i%len(benchCountries)]
		ev.GeoIP = geoip.Result{CountryCode: cc, CountryName: cc, ContinentCode: "NA", ASN: uint(1000 + i%50), ASNOrg: "Bench"}
	}
	return ev
}

// benchSeedAccessLog seeds n rows one second apart. The sort benchmarks care only
// that every row falls inside their window, so the span is incidental there.
func benchSeedAccessLog(tb testing.TB, repo *accesslog.Repository, n int) {
	tb.Helper()
	benchSeedAccessLogOver(tb, repo, n, time.Duration(n)*time.Second)
}

// benchSeedAccessLogOver seeds n rows spread evenly across span, ending at now.
// Spreading matters to any benchmark that varies its window: rows packed into a
// few minutes make every window scan the same set, which hides the very cost the
// window is meant to expose.
func benchSeedAccessLogOver(tb testing.TB, repo *accesslog.Repository, n int, span time.Duration) {
	tb.Helper()
	ctx := context.Background()
	base := time.Now().UTC().Add(-span)
	// Divide before multiplying: span in nanoseconds times a five-figure row index
	// overflows the int64 behind time.Duration, which would silently stack every row
	// on one instant and leave the narrow windows matching nothing.
	step := span / time.Duration(max(n, 1))
	const chunk = 200 // bound the per-insert parameter count, like the real Sink batches
	events := make([]policy.DecisionEvent, 0, chunk)
	flush := func() {
		if len(events) == 0 {
			return
		}
		if err := repo.BatchInsert(ctx, events); err != nil {
			tb.Fatalf("seed BatchInsert: %v", err)
		}
		events = events[:0]
	}
	for i := range n {
		events = append(events, benchAccessEvent(i, base.Add(step*time.Duration(i))))
		if len(events) == chunk {
			flush()
		}
	}
	flush()
}

func benchAccessLogRepo(tb testing.TB) *accesslog.Repository {
	tb.Helper()
	db, cleanup := testdb.Setup(tb)
	tb.Cleanup(cleanup)
	return accesslog.NewRepository(db.DB())
}

// BenchmarkListAccessLog measures the table: a filtered, sorted, keyset-paged
// window over access_log with two 1:1 joins and a second contributor fetch. Its
// cost is dominated by the sort — when the sort column is not created_at, SQLite
// cannot satisfy the order from an index and materialises every row in the window
// into a temp B-tree before taking LIMIT. The window is held fixed so that lever
// stays isolated.
func BenchmarkListAccessLog(b *testing.B) {
	ctx := context.Background()
	// created_at is the indexed default (cheap); client_ip and duration_us force the
	// temp-B-tree sort whose cost grows with the window.
	sorts := []string{"created_at", "client_ip", "duration_us"}
	for _, n := range []int{1_000, 10_000} {
		for _, sort := range sorts {
			b.Run(fmt.Sprintf("%s/n=%d", sort, n), func(b *testing.B) {
				repo := benchAccessLogRepo(b)
				benchSeedAccessLog(b, repo, n)
				q := accesslog.AccessLogQuery{
					From:  time.Now().UTC().Add(-90 * 24 * time.Hour),
					To:    time.Now().UTC().Add(24 * time.Hour),
					Sort:  sort,
					Order: "desc",
					Limit: 50,
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if _, _, err := repo.ListAccessLog(ctx, q); err != nil {
						b.Fatalf("ListAccessLog: %v", err)
					}
				}
			})
		}
	}
}

// benchRetention is the history the histogram fixtures span. Windows narrower
// than it scan a slice of the table rather than all of it, which is the shape
// retention leaves behind: a month of rows on disk, a chart asking for a day.
const benchRetention = 30 * 24 * time.Hour

// BenchmarkGetAccessLogHistogram measures the chart. It aggregates raw access_log
// at every window width, so its cost tracks the rows *inside* the window and not
// the handful of buckets it returns — a month folds the whole table into ~31 rows,
// having read all of it. The window is therefore the lever, and it is one click in
// the preset bar, so widening it must stay affordable rather than merely correct.
//
// The fixtures are far smaller than production, where a month is millions of rows;
// read these numbers as a relative gate on the query shape, not as a latency budget.
func BenchmarkGetAccessLogHistogram(b *testing.B) {
	ctx := context.Background()
	cases := []struct {
		name    string
		window  time.Duration
		filters []filterx.Filter
	}{
		// The ladder buckets the first two hourly and the third daily, so these also
		// cover both live rungs of GranularityForWindow.
		{name: "24h", window: 24 * time.Hour},
		{name: "7d", window: 7 * 24 * time.Hour},
		{name: "30d", window: benchRetention},
		// A country filter joins access_log_geoip and narrows the result to a fraction
		// of the window — the shape the hourly rollups cannot answer at all.
		{
			name:   "30d-filtered",
			window: benchRetention,
			filters: []filterx.Filter{
				{Column: "country_code", Op: filterx.OpIn, Values: []any{"US", "DE"}},
			},
		},
	}

	for _, n := range []int{10_000, 100_000} {
		for _, tc := range cases {
			b.Run(fmt.Sprintf("%s/n=%d", tc.name, n), func(b *testing.B) {
				repo := benchAccessLogRepo(b)
				benchSeedAccessLogOver(b, repo, n, benchRetention)
				now := time.Now().UTC()
				q := accesslog.AccessLogQuery{
					From:    now.Add(-tc.window),
					To:      now,
					Filters: tc.filters,
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if _, err := repo.GetAccessLogHistogram(ctx, q); err != nil {
						b.Fatalf("GetAccessLogHistogram: %v", err)
					}
				}
			})
		}
	}
}
