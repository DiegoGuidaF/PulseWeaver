//go:build test

package accesslog_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

// Fixtures are hand-built and owned here (never the shared seeder — see the note
// in internal/policy/bench_test.go). They model the two access-log write shapes:
// a childless deny carrying only the minimal forwarding headers (the flood case),
// and an allow with a contributor + geoip child and the full header set.
//
//	go test -tags=test -run=^$ -bench=BatchInsert -benchmem ./internal/accesslog/
//
// Do not commit raw ns/op numbers — record before/after deltas in the commit message.

func benchDenyEvent() policy.DecisionEvent {
	return policy.DecisionEvent{
		ClientIP:   "203.0.113.7",
		Outcome:    false,
		DenyReason: new(policy.DenyReasonIPNotRegistered),
		CreatedAt:  time.Now().UTC(),
		TargetHost: new("whoami-gated.localhost"),
		TargetURI:  new("/"),
		HTTPMethod: new("GET"),
		XFFChain:   new("203.0.113.7"),
		Headers: map[string][]string{
			"X-Forwarded-Host":   {"whoami-gated.localhost"},
			"X-Forwarded-Uri":    {"/"},
			"X-Forwarded-Method": {"GET"},
			"X-Forwarded-Proto":  {"https"},
			"X-Forwarded-For":    {"203.0.113.7"},
			"X-Real-Ip":          {"203.0.113.7"},
		},
	}
}

func benchAllowEvent(devID ids.DeviceID, addrID ids.AddressID, userID ids.UserID) policy.DecisionEvent {
	return policy.DecisionEvent{
		ClientIP:    "203.0.113.7",
		Outcome:     true,
		CreatedAt:   time.Now().UTC(),
		TargetHost:  new("whoami-gated.localhost"),
		TargetURI:   new("/"),
		HTTPMethod:  new("GET"),
		XFFChain:    new("203.0.113.7"),
		MatchSource: policy.MatchSourceDevice,
		Headers: map[string][]string{
			"User-Agent":         {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"},
			"Accept-Encoding":    {"gzip, deflate, br"},
			"Content-Type":       {"text/html"},
			"Via":                {"1.1 Caddy"},
			"X-Forwarded-Host":   {"whoami-gated.localhost"},
			"X-Forwarded-Uri":    {"/"},
			"X-Forwarded-Method": {"GET"},
			"X-Forwarded-Proto":  {"https"},
			"X-Forwarded-For":    {"203.0.113.7"},
			"X-Real-Ip":          {"203.0.113.7"},
		},
		IPContributors: []policy.IPContributor{{DeviceID: devID, AddressID: addrID, UserID: userID}},
		GeoIP:          geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 15169, ASNOrg: "Google LLC"},
	}
}

func benchRepo(tb testing.TB) (*accesslog.Repository, *database.DB) {
	tb.Helper()
	db, cleanup := testdb.Setup(tb)
	tb.Cleanup(cleanup)
	return accesslog.NewRepository(db.DB()), db.DB()
}

// benchOwnedAddress inserts a user/device/address triple so allow events can
// satisfy the contributor foreign keys, and returns their IDs.
func benchOwnedAddress(tb testing.TB, db *database.DB) (ids.DeviceID, ids.AddressID, ids.UserID) {
	tb.Helper()
	ctx := context.Background()
	var userID ids.UserID
	if err := db.QueryRowxContext(ctx,
		`INSERT INTO users (username, display_name, password_hash, role) VALUES ('owner', 'Owner', NULL, 'admin') RETURNING id`,
	).Scan(&userID); err != nil {
		tb.Fatalf("insert owner: %v", err)
	}
	var devID ids.DeviceID
	if err := db.QueryRowxContext(ctx,
		`INSERT INTO devices (name, owner_id) VALUES ('bench-device', ?) RETURNING id`, userID,
	).Scan(&devID); err != nil {
		tb.Fatalf("insert device: %v", err)
	}
	var addrID ids.AddressID
	if err := db.QueryRowxContext(ctx,
		`INSERT INTO addresses (device_id, ip, source, is_enabled) VALUES (?, '203.0.113.7', 'manual', 1) RETURNING id`, devID,
	).Scan(&addrID); err != nil {
		tb.Fatalf("insert address: %v", err)
	}
	return devID, addrID, userID
}

func benchBatch(n int, mk func() policy.DecisionEvent) []policy.DecisionEvent {
	events := make([]policy.DecisionEvent, n)
	for i := range n {
		events[i] = mk()
	}
	return events
}

func BenchmarkBatchInsert(b *testing.B) {
	ctx := context.Background()

	// Deny shape: childless rows with minimal headers — the flood case. The
	// Sink's real batch size is 50; 1 isolates per-row cost.
	for _, n := range []int{1, 50} {
		b.Run(fmt.Sprintf("deny/%d", n), func(b *testing.B) {
			repo, _ := benchRepo(b)
			events := benchBatch(n, benchDenyEvent)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if err := repo.BatchInsert(ctx, events); err != nil {
					b.Fatalf("BatchInsert: %v", err)
				}
			}
		})
	}

	// Allow shape: full headers + one contributor + geoip child per row.
	b.Run("allow/50", func(b *testing.B) {
		repo, db := benchRepo(b)
		devID, addrID, userID := benchOwnedAddress(b, db)
		events := benchBatch(50, func() policy.DecisionEvent { return benchAllowEvent(devID, addrID, userID) })
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if err := repo.BatchInsert(ctx, events); err != nil {
				b.Fatalf("BatchInsert: %v", err)
			}
		}
	})
}
