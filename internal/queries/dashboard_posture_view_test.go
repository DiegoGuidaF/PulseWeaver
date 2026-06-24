//go:build test

package queries

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

// TestFoldUserStatuses verifies the histogram fold classifies each user via the
// shared deriveUserStatus: bypass short-circuits, and the live-IP set crossed with
// the has-grants flag produces the four reachability/authorization buckets.
func TestFoldUserStatuses(t *testing.T) {
	is := is.New(t)

	users := []postureUserRow{
		{UserID: 1, Bypass: true, HasGrants: false},  // bypass
		{UserID: 2, Bypass: false, HasGrants: true},  // live + grants  → live_with_access
		{UserID: 3, Bypass: false, HasGrants: false}, // live, no grants → live_no_host_access
		{UserID: 4, Bypass: false, HasGrants: true},  // no live, grants → no_live_ips
		{UserID: 5, Bypass: false, HasGrants: false}, // neither         → no_access
	}
	liveIPUsers := map[ids.UserID]struct{}{
		2: {},
		3: {},
	}

	got := foldUserStatuses(users, liveIPUsers)

	is.Equal(got.Bypass, 1)
	is.Equal(got.LiveWithAccess, 1)
	is.Equal(got.LiveNoHostAccess, 1)
	is.Equal(got.NoLiveIps, 1)
	is.Equal(got.NoAccess, 1)
}

// TestSummarizeLiveIPs verifies the live-user set, the shared-IP count, and that IP
// canonicalization matches the cache: an IPv4-mapped IPv6 address collapses onto its
// plain twin, and unparseable IPs are skipped (no phantom live user).
func TestSummarizeLiveIPs(t *testing.T) {
	is := is.New(t)

	entries := []device.IPEntry{
		{IP: "10.0.0.1", UserID: 1},        // user 1 alone on this IP
		{IP: "10.0.0.2", UserID: 2},        // shared IP, user 2 ...
		{IP: "10.0.0.2", UserID: 3},        // ... and user 3
		{IP: "::ffff:10.0.0.2", UserID: 4}, // mapped twin of 10.0.0.2 → same key, user 4
		{IP: "not-an-ip", UserID: 5},       // skipped: never becomes live
	}

	liveUsers, shared := summarizeLiveIPs(entries)

	is.Equal(len(liveUsers), 4) // users 1-4; user 5 excluded
	_, has5 := liveUsers[5]
	is.True(!has5)

	// 10.0.0.2 carries users 2, 3, 4 (mapped twin folded in) → one shared IP.
	// 10.0.0.1 has a single user → not shared.
	is.Equal(shared, 1)
}
