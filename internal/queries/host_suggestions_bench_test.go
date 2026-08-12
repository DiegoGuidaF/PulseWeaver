//go:build test

package queries

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

// benchSuggestionHosts/benchSuggestionIPs bound the cardinality of seeded
// traffic. A wide window's raw row count still grows with n, but rollup
// collapses it onto at most len(hosts)*len(ips) aggregate rows for one hour —
// the compression the >24h path is built to exploit.
var (
	benchSuggestionHosts = []string{
		"svc1.internal", "svc2.internal", "svc3.internal", "svc4.internal", "svc5.internal",
		"svc6.internal", "svc7.internal", "svc8.internal", "svc9.internal", "svc10.internal",
	}
	benchSuggestionIPs = []string{
		"10.10.0.1", "10.10.0.2", "10.10.0.3", "10.10.0.4", "10.10.0.5",
		"10.10.0.6", "10.10.0.7", "10.10.0.8", "10.10.0.9", "10.10.0.10",
	}
)

func benchSuggestionEvent(i int, at time.Time) policy.DecisionEvent {
	host := benchSuggestionHosts[i%len(benchSuggestionHosts)]
	return policy.DecisionEvent{
		ClientIP:   benchSuggestionIPs[i%len(benchSuggestionIPs)],
		Outcome:    false,
		DenyReason: new(policy.DenyReasonIPNotRegistered),
		TargetHost: &host,
		CreatedAt:  at,
		Headers:    map[string][]string{},
	}
}

// BenchmarkPendingHostSuggestions compares the ≤24h raw path against the >24h
// aggregate path at increasing raw-table scale. The raw sub-benchmark scans n
// access_log rows directly; the aggregate sub-benchmark seeds the same n rows
// into one complete past hour, rolls them up, and prunes the raw rows before
// timing — so its cost is bounded by the rolled-up row count (≤100 here), not
// n, however large the raw table would have grown.
func BenchmarkPendingHostSuggestions(b *testing.B) {
	ctx := context.Background()

	for _, n := range []int{1_000, 20_000} {
		b.Run(fmt.Sprintf("raw/n=%d", n), func(b *testing.B) {
			db, cleanup := testdb.Setup(b)
			b.Cleanup(cleanup)
			repo := NewRepository(db.DB())
			accessLogRepo := accesslog.NewRepository(db.DB(), nil)

			base := time.Now().UTC().Add(-2 * time.Hour)
			events := make([]policy.DecisionEvent, n)
			for i := range n {
				events[i] = benchSuggestionEvent(i, base.Add(time.Duration(i)*time.Millisecond))
			}
			if err := accessLogRepo.BatchInsert(ctx, events); err != nil {
				b.Fatalf("seed: %v", err)
			}
			since := time.Now().UTC().Add(-3 * time.Hour)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := repo.pendingHostSuggestions(ctx, since, map[string]bool{}, map[string]bool{}); err != nil {
					b.Fatalf("pendingHostSuggestions: %v", err)
				}
			}
		})

		b.Run(fmt.Sprintf("aggregate/n=%d", n), func(b *testing.B) {
			db, cleanup := testdb.Setup(b)
			b.Cleanup(cleanup)
			repo := NewRepository(db.DB())
			accessLogRepo := accesslog.NewRepository(db.DB(), nil)
			rollupRepo := rollup.NewRepository(db.DB(), nil)

			currentHourStart := time.Now().UTC().Truncate(time.Hour)
			oldHour := currentHourStart.Add(-30 * time.Hour)
			events := make([]policy.DecisionEvent, n)
			for i := range n {
				events[i] = benchSuggestionEvent(i, oldHour.Add(time.Duration(i%3000)*time.Millisecond))
			}
			if err := accessLogRepo.BatchInsert(ctx, events); err != nil {
				b.Fatalf("seed: %v", err)
			}
			if err := rollupRepo.RunRollup(ctx, oldHour.Add(-time.Hour), currentHourStart); err != nil {
				b.Fatalf("rollup: %v", err)
			}
			if _, err := accessLogRepo.DeleteOlderThan(ctx, currentHourStart); err != nil {
				b.Fatalf("prune: %v", err)
			}
			since := time.Now().UTC().Add(-48 * time.Hour)

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := repo.pendingHostSuggestions(ctx, since, map[string]bool{}, map[string]bool{}); err != nil {
					b.Fatalf("pendingHostSuggestions: %v", err)
				}
			}
		})
	}
}
