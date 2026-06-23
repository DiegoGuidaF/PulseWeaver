//go:build test

package policy

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
)

// Benchmarks for the in-memory forward-auth hot path. Fixtures are hand-built and
// owned here (never the shared seeder, which grows over time and would inject false
// benchstat regressions). Run with:
//
//	go test -tags=test -run=^$ -bench=. -benchmem ./internal/policy/
//
// Do not commit raw ns/op numbers — they are host-specific and only comparable
// same-machine, back-to-back. The benchmark code is the durable record of what is
// measured; record meaningful before/after deltas in the commit message.

var benchScale = []int{10, 100, 1000}

// benchIP returns a deterministic 10.x.x.x address for index i (i up to ~16M).
func benchIP(i int) netip.Addr {
	return netip.AddrFrom4([4]byte{10, byte(i >> 16), byte(i >> 8), byte(i)})
}

// benchHosts returns k deterministic case-folded FQDNs.
func benchHosts(k int) []string {
	hs := make([]string, k)
	for i := range k {
		hs[i] = fmt.Sprintf("host%d.example.com", i)
	}
	return hs
}

// benchIPEntries builds n enabled IP rows, each a distinct IP owned by a distinct user.
func benchIPEntries(n int) []device.IPEntry {
	entries := make([]device.IPEntry, n)
	for i := range n {
		entries[i] = device.IPEntry{
			IP:        benchIP(i).String(),
			DeviceID:  ids.DeviceID(i + 1),
			AddressID: ids.AddressID(i + 1),
			UserID:    ids.UserID(i + 1),
		}
	}
	return entries
}

// benchHostAccessBypass grants allowlist bypass to users 1..n.
func benchHostAccessBypass(n int) []UserHostAccess {
	ha := make([]UserHostAccess, n)
	for i := range n {
		ha[i] = UserHostAccess{UserID: ids.UserID(i + 1), BypassAllowlist: true}
	}
	return ha
}

// benchHostAccessRestricted grants users 1..n a fixed host set (no bypass).
func benchHostAccessRestricted(n, hostsPerUser int) []UserHostAccess {
	ha := make([]UserHostAccess, n)
	for i := range n {
		ha[i] = UserHostAccess{UserID: ids.UserID(i + 1), AllowedHosts: benchHosts(hostsPerUser)}
	}
	return ha
}

// benchSharedIPFixtures builds n IPs, each contributed by two distinct restricted
// users with overlapping host sets, so buildIPSet runs the deny-wins intersection
// (maps.Clone + intersectHostSets) once per IP.
func benchSharedIPFixtures(n, hostsPerUser int) ([]device.IPEntry, []UserHostAccess) {
	entries := make([]device.IPEntry, 0, n*2)
	ha := make([]UserHostAccess, 0, n*2)
	for i := range n {
		ip := benchIP(i).String()
		u1 := ids.UserID(2*i + 1)
		u2 := ids.UserID(2*i + 2)
		entries = append(entries,
			device.IPEntry{IP: ip, DeviceID: ids.DeviceID(2*i + 1), AddressID: ids.AddressID(2*i + 1), UserID: u1},
			device.IPEntry{IP: ip, DeviceID: ids.DeviceID(2*i + 2), AddressID: ids.AddressID(2*i + 2), UserID: u2},
		)
		ha = append(ha,
			UserHostAccess{UserID: u1, AllowedHosts: benchHosts(hostsPerUser)},
			UserHostAccess{UserID: u2, AllowedHosts: benchHosts(hostsPerUser)},
		)
	}
	return entries, ha
}

// benchCacheEntries builds n network policy rows, each a distinct /24 in 10.0.0.0/8.
func benchCacheEntries(n int) []networkpolicies.CacheEntry {
	entries := make([]networkpolicies.CacheEntry, n)
	for i := range n {
		entries[i] = networkpolicies.CacheEntry{
			PolicyID:         ids.NetworkPolicyID(i + 1),
			PolicyName:       fmt.Sprintf("policy-%d", i),
			CIDR:             fmt.Sprintf("10.%d.%d.0/24", i/256, i%256),
			AllowedHostFQDNs: benchHosts(4),
		}
	}
	return entries
}

// benchSecret is the API secret the benchmark services are built with; the same
// value is used as the bearer token in BenchmarkVerifyAccess's allowed path.
const benchSecret = "secret"

// benchService builds a Service with caches set directly (no providers, no DB).
func benchService(tb testing.TB, ipSet map[netip.Addr]ipSetEntry, nps []networkPolicyCacheEntry) *Service {
	tb.Helper()
	svc, err := NewService(nil, nil, nil, nil, benchSecret, noopLogger(), netip.Addr{})
	if err != nil {
		tb.Fatalf("NewService: %v", err)
	}
	if ipSet != nil {
		svc.ipSet = ipSet
	}
	svc.networkPolicies = nps
	return svc
}

// noopObserver is a zero-allocation DecisionObserver so VerifyAccess benchmarks
// measure the decision path, not observer bookkeeping.
type noopObserver struct{}

func (noopObserver) OnDecision(context.Context, DecisionEvent) {}

func BenchmarkDecide(b *testing.B) {
	ctx := context.Background()
	const host = "host0.example.com"

	// Exact IP hit: the registered, bypass IP resolves on the map lookup.
	b.Run("hit", func(b *testing.B) {
		svc := benchService(b, buildIPSet(benchIPEntries(1), benchHostAccessBypass(1)), nil)
		ip := benchIP(0)
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = svc.Decide(ctx, ip, host)
		}
	})

	// Miss → CIDR loop: an IP outside 10.0.0.0/8 traverses all N policies before denying.
	missIP := netip.MustParseAddr("203.0.113.7")
	for _, n := range benchScale {
		b.Run(fmt.Sprintf("miss-cidr/%d", n), func(b *testing.B) {
			svc := benchService(b, nil, buildNetworkPolicyCache(ctx, benchCacheEntries(n), noopLogger()))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = svc.Decide(ctx, missIP, host)
			}
		})
	}

	// Deny: empty caches, immediate ip_not_registered.
	b.Run("deny", func(b *testing.B) {
		svc := benchService(b, nil, nil)
		ip := netip.MustParseAddr("198.51.100.1")
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = svc.Decide(ctx, ip, host)
		}
	})
}

func BenchmarkVerifyAccess(b *testing.B) {
	ctx := context.Background()
	host := "host0.example.com"

	// Allowed: valid token + registered bypass IP. Isolates sha256 + constant-time
	// compare + DecisionEvent construction (nil geo resolver, no-op observer).
	b.Run("allowed", func(b *testing.B) {
		svc := benchService(b, buildIPSet(benchIPEntries(1), benchHostAccessBypass(1)), nil)
		svc.AddDecisionObserver(noopObserver{})
		req := &VerifyRequest{Token: benchSecret, ClientIP: benchIP(0), TargetHost: &host}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = svc.VerifyAccess(ctx, req)
		}
	})

	// Denied: valid token, unregistered IP → ip_not_registered.
	b.Run("denied", func(b *testing.B) {
		svc := benchService(b, nil, nil)
		svc.AddDecisionObserver(noopObserver{})
		ip := netip.MustParseAddr("198.51.100.9")
		req := &VerifyRequest{Token: benchSecret, ClientIP: ip, TargetHost: &host}
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = svc.VerifyAccess(ctx, req)
		}
	})
}

func BenchmarkBuildIPSet(b *testing.B) {
	const hostsPerUser = 8
	for _, n := range benchScale {
		// One restricted user per IP: maps.Clone once per IP, no intersection.
		b.Run(fmt.Sprintf("no-intersection/%d", n), func(b *testing.B) {
			entries := benchIPEntries(n)
			hostAccess := benchHostAccessRestricted(n, hostsPerUser)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = buildIPSet(entries, hostAccess)
			}
		})

		// Two restricted users per IP: deny-wins intersection runs once per IP.
		b.Run(fmt.Sprintf("intersection/%d", n), func(b *testing.B) {
			entries, hostAccess := benchSharedIPFixtures(n, hostsPerUser)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = buildIPSet(entries, hostAccess)
			}
		})
	}
}

// benchRequestHeader builds a representative forward-auth request header set:
// the X-Forwarded-* forwarding context plus the everyday client headers a proxied
// request carries. Mirrors the width seen in the profiled prod-like DB (~10 keys).
func benchRequestHeader() http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer some-long-opaque-token-value-1234567890")
	h.Set("Cookie", "session=abc; theme=dark")
	h.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	h.Set("Accept-Encoding", "gzip, deflate, br")
	h.Set("Content-Type", "text/html")
	h.Set("Via", "1.1 Caddy")
	h.Set("X-Forwarded-For", "203.0.113.7")
	h.Set("X-Forwarded-Host", "whoami-gated.localhost")
	h.Set("X-Forwarded-Method", "GET")
	h.Set("X-Forwarded-Proto", "https")
	h.Set("X-Forwarded-Uri", "/")
	h.Set("X-Real-Ip", "203.0.113.7")
	return h
}

// BenchmarkEnrichmentHeaders contrasts the full-clone (allow) path against the
// minimal forwarding-subset (deny) path — the per-request allocation lever that
// PW-95 flagged as policy.enrichmentHeaders churn.
func BenchmarkEnrichmentHeaders(b *testing.B) {
	h := benchRequestHeader()

	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = fullEnrichmentHeaders(h)
		}
	})

	b.Run("minimal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = minimalEnrichmentHeaders(h)
		}
	})
}

func BenchmarkBuildNetworkPolicyCache(b *testing.B) {
	ctx := context.Background()
	logger := noopLogger()
	for _, n := range benchScale {
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			entries := benchCacheEntries(n)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = buildNetworkPolicyCache(ctx, entries, logger)
			}
		})
	}
}
