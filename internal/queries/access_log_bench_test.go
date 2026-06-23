//go:build test

package queries_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

// ListAccessLog is the heaviest read in the app: a filtered, sorted, keyset-paged
// window over access_log with two 1:1 joins and a second contributor fetch. Its cost
// is dominated by the sort — when the sort column is not created_at, SQLite cannot
// satisfy the order from an index and materialises every row in the window into a
// temp B-tree before taking LIMIT. This benchmark holds the window fixed and varies
// the sort so that lever is visible and any future fix is gated by benchstat.
//
// Fixtures are hand-built and owned here (never the shared seeder — see the note in
// internal/policy/bench_test.go); they are written through the production BatchInsert
// path so the on-disk shape matches what verify-ip actually persists.
//
//	go test -tags=test -run=^$ -bench=ListAccessLog -benchmem ./internal/queries/
//
// Do not commit raw ns/op numbers — record before/after deltas in the commit message.

var benchCountries = []string{"US", "DE", "FR", "GB", "JP", "BR", "IN", "AU"}

// benchAccessEvent builds one varied access-log row: a distinct client IP, a spread of
// durations and outcomes, and partial geoip coverage — so non-time sorts must actually
// reorder the window rather than read it back in insertion order.
func benchAccessEvent(i int, base time.Time) policy.DecisionEvent {
	ev := policy.DecisionEvent{
		ClientIP:   fmt.Sprintf("198.51.%d.%d", (i>>8)&0xff, i&0xff),
		Outcome:    i%4 != 0,
		CreatedAt:  base.Add(time.Duration(i) * time.Second),
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

func benchSeedAccessLog(tb testing.TB, repo *accesslog.Repository, n int) {
	tb.Helper()
	ctx := context.Background()
	base := time.Now().UTC().Add(-30 * 24 * time.Hour)
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
		events = append(events, benchAccessEvent(i, base))
		if len(events) == chunk {
			flush()
		}
	}
	flush()
}

func benchQueriesRepos(tb testing.TB) (*queries.Repository, *accesslog.Repository) {
	tb.Helper()
	db, cleanup := testdb.Setup(tb)
	tb.Cleanup(cleanup)
	sqlxDB := db.DB()
	return queries.NewRepository(sqlxDB), accesslog.NewRepository(sqlxDB)
}

func BenchmarkListAccessLog(b *testing.B) {
	ctx := context.Background()
	// created_at is the indexed default (cheap); client_ip and duration_us force the
	// temp-B-tree sort whose cost grows with the window.
	sorts := []string{"created_at", "client_ip", "duration_us"}
	for _, n := range []int{1_000, 10_000} {
		for _, sort := range sorts {
			b.Run(fmt.Sprintf("%s/n=%d", sort, n), func(b *testing.B) {
				qRepo, alRepo := benchQueriesRepos(b)
				benchSeedAccessLog(b, alRepo, n)
				q := queries.AccessLogQuery{
					From:  time.Now().UTC().Add(-90 * 24 * time.Hour),
					To:    time.Now().UTC().Add(24 * time.Hour),
					Sort:  sort,
					Order: "desc",
					Limit: 50,
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					if _, _, err := qRepo.ListAccessLog(ctx, q); err != nil {
						b.Fatalf("ListAccessLog: %v", err)
					}
				}
			})
		}
	}
}
