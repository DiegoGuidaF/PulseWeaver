//go:build test

package queries

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// ── helpers ────────────────────────────────────────────────────────────────────

var (
	userAlice   = auth.UserID(1)
	userBob     = auth.UserID(2)
	userCharlie = auth.UserID(3) // no-access user

	devAlice1  = device.DeviceID(10)
	devAlice2  = device.DeviceID(11)
	devBob1    = device.DeviceID(20)
	addrAlice1 = device.AddressID(100)
	addrAlice2 = device.AddressID(101)
	addrBob1   = device.AddressID(200)

	baseTime = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
)

func makeEnrichment(addressID device.AddressID, deviceID device.DeviceID, deviceName string, userID auth.UserID, userName string, updatedAt time.Time) policyEnrichmentRow {
	return policyEnrichmentRow{
		AddressID:        addressID,
		AddressUpdatedAt: updatedAt,
		DeviceID:         deviceID,
		DeviceName:       deviceName,
		UserID:           userID,
		UserName:         userName,
	}
}

// ── tests ──────────────────────────────────────────────────────────────────────

// TestAssemblePolicyUserMap_SingleUser verifies single user, single IP, no intersection.
func TestAssemblePolicyUserMap_SingleUser(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		LastRefreshedAt:       baseTime,
		LastRefreshDurationMs: 42,
		Entries: []policy.PolicyMapEntry{
			{
				IP:                  "1.2.3.4",
				BypassAllowlist:     false,
				AllowedHosts:        []string{"a.com", "b.com"},
				IntersectionApplied: false,
				Contributors: []policy.ContributorAccess{
					{
						DeviceID:         devAlice1,
						AddressID:        addrAlice1,
						UserID:           userAlice,
						UserBypass:       false,
						UserAllowedHosts: []string{"a.com", "b.com"},
					},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-device", userAlice, "Alice", baseTime),
	}

	allUsers := []policyAuditUserRow{
		{UserID: userAlice, UserName: "Alice", BypassAllowlist: false},
	}

	allowedHosts := map[auth.UserID][]string{
		userAlice: {"a.com", "b.com"},
	}

	result := assemblePolicyUserMap(snap, enrichment, allUsers, allowedHosts)

	is.Equal(int(result.RefreshDurationMs), 42)
	is.Equal(result.TotalIpCount, 1)
	is.Equal(result.TotalDeviceCount, 1)
	is.Equal(result.TotalHostCount, 2)
	is.Equal(result.SharedIpCount, 0)
	is.Equal(len(result.Users), 1)

	u := result.Users[0]
	is.Equal(u.UserId, int64(userAlice))
	is.Equal(u.UserName, "Alice")
	is.Equal(u.IsAdmin, false)
	is.Equal(u.BypassAllowlist, false)
	is.Equal(u.DeviceCount, 1)
	is.Equal(u.IpCount, 1)
	is.Equal(u.AllowedHostCount, 2)
	is.Equal(u.OnSharedIp, false)
	is.Equal(u.IntersectionApplied, false)
	is.True(u.LastSeenAt != nil)

	is.Equal(len(u.Ips), 1)
	ip := u.Ips[0]
	is.Equal(ip.Ip, "1.2.3.4")
	is.Equal(ip.BypassAtIp, false)
	is.Equal(len(ip.SharedWithUsers), 0)
	is.Equal(ip.EffectiveHosts, []string{"a.com", "b.com"})
	is.Equal(len(ip.TrimmedHosts), 0)
	is.Equal(len(ip.Addresses), 1)
}

// TestAssemblePolicyUserMap_TwoDevicesSameNAT verifies two devices for the same
// user behind the same NAT: shared_with_user_ids must be empty.
func TestAssemblePolicyUserMap_TwoDevicesSameNAT(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{
				IP:           "10.0.0.1",
				AllowedHosts: []string{"a.com"},
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice, UserAllowedHosts: []string{"a.com"}},
					{DeviceID: devAlice2, AddressID: addrAlice2, UserID: userAlice, UserAllowedHosts: []string{"a.com"}},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev-1", userAlice, "Alice", baseTime),
		addrAlice2: makeEnrichment(addrAlice2, devAlice2, "alice-dev-2", userAlice, "Alice", baseTime.Add(time.Minute)),
	}

	allUsers := []policyAuditUserRow{
		{UserID: userAlice, UserName: "Alice"},
	}

	result := assemblePolicyUserMap(snap, enrichment, allUsers, map[auth.UserID][]string{
		userAlice: {"a.com"},
	})

	is.Equal(len(result.Users), 1)
	u := result.Users[0]
	is.Equal(u.DeviceCount, 2)
	is.Equal(u.IpCount, 1)
	is.Equal(u.OnSharedIp, false) // same user — not shared

	is.Equal(len(u.Ips), 1)
	ip := u.Ips[0]
	is.Equal(len(ip.SharedWithUsers), 0)
	is.Equal(len(ip.Addresses), 2)

	// Addresses must be sorted by address_id.
	is.True(ip.Addresses[0].AddressId < ip.Addresses[1].AddressId)
}

// TestAssemblePolicyUserMap_TwoUsersSharedIP_IntersectionTrims verifies that
// when two restricted users share an IP and the intersection trims one user's
// effective hosts, each user gets correct effective_hosts and trimmed_hosts.
func TestAssemblePolicyUserMap_TwoUsersSharedIP_IntersectionTrims(t *testing.T) {
	is := is.New(t)

	// Entry AllowedHosts is the deny-wins intersection = just "a.com".
	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{
				IP:                  "10.0.0.2",
				BypassAllowlist:     false,
				AllowedHosts:        []string{"a.com"}, // intersection result
				IntersectionApplied: true,
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice, UserAllowedHosts: []string{"a.com", "b.com"}},
					{DeviceID: devBob1, AddressID: addrBob1, UserID: userBob, UserAllowedHosts: []string{"a.com"}},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev", userAlice, "Alice", baseTime),
		addrBob1:   makeEnrichment(addrBob1, devBob1, "bob-dev", userBob, "Bob", baseTime),
	}

	// SQL order: Alice before Bob.
	allUsers := []policyAuditUserRow{
		{UserID: userAlice, UserName: "Alice", Username: "alice"},
		{UserID: userBob, UserName: "Bob", Username: "bob"},
	}

	allowedHosts := map[auth.UserID][]string{
		userAlice: {"a.com", "b.com"},
		userBob:   {"a.com"},
	}

	result := assemblePolicyUserMap(snap, enrichment, allUsers, allowedHosts)

	is.Equal(len(result.Users), 2)

	// Alice: effective = {a.com}, trimmed = {b.com}
	alice := result.Users[0]
	is.Equal(alice.UserName, "Alice")
	is.Equal(alice.IntersectionApplied, true)
	is.Equal(alice.OnSharedIp, true)
	is.Equal(len(alice.Ips), 1)
	aliceIP := alice.Ips[0]
	is.Equal(aliceIP.EffectiveHosts, []string{"a.com"})
	is.Equal(aliceIP.TrimmedHosts, []string{"b.com"})
	is.Equal(len(aliceIP.SharedWithUsers), 1)
	is.Equal(aliceIP.SharedWithUsers[0].UserId, int64(userBob))
	is.Equal(aliceIP.SharedWithUsers[0].Username, "bob")
	is.Equal(aliceIP.SharedWithUsers[0].UserName, "Bob")

	// Bob: effective = {a.com}, trimmed = {} (his set wasn't trimmed)
	bob := result.Users[1]
	is.Equal(bob.UserName, "Bob")
	is.Equal(bob.IntersectionApplied, false)
	is.Equal(bob.OnSharedIp, true)
	is.Equal(len(bob.Ips), 1)
	bobIP := bob.Ips[0]
	is.Equal(bobIP.EffectiveHosts, []string{"a.com"})
	is.Equal(len(bobIP.TrimmedHosts), 0)
	is.Equal(len(bobIP.SharedWithUsers), 1)
	is.Equal(bobIP.SharedWithUsers[0].UserId, int64(userAlice))
	is.Equal(bobIP.SharedWithUsers[0].Username, "alice")
	is.Equal(bobIP.SharedWithUsers[0].UserName, "Alice")
}

// TestAssemblePolicyUserMap_BypassUserOnSharedIP verifies that a bypass user
// sitting alongside a restricted user gets empty effective/trimmed hosts but
// bypass_at_ip remains false (not all contributors bypass).
func TestAssemblePolicyUserMap_BypassUserOnSharedIP(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{
				IP:              "10.0.0.3",
				BypassAllowlist: false, // NOT full-IP bypass
				AllowedHosts:    []string{"a.com"},
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice, UserBypass: true, UserAllowedHosts: []string{}},
					{DeviceID: devBob1, AddressID: addrBob1, UserID: userBob, UserBypass: false, UserAllowedHosts: []string{"a.com"}},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev", userAlice, "Alice", baseTime),
		addrBob1:   makeEnrichment(addrBob1, devBob1, "bob-dev", userBob, "Bob", baseTime),
	}

	allUsers := []policyAuditUserRow{
		{UserID: userAlice, UserName: "Alice", BypassAllowlist: true},
		{UserID: userBob, UserName: "Bob", BypassAllowlist: false},
	}

	result := assemblePolicyUserMap(snap, enrichment, allUsers, map[auth.UserID][]string{
		userAlice: {},
		userBob:   {"a.com"},
	})

	is.Equal(len(result.Users), 2)

	// Alice: bypass user — effective_hosts and trimmed_hosts empty
	alice := result.Users[0]
	is.Equal(alice.BypassAllowlist, true)
	is.Equal(alice.AllowedHostCount, 0)
	aliceIP := alice.Ips[0]
	is.Equal(aliceIP.BypassAtIp, false) // not full-IP bypass
	is.Equal(len(aliceIP.EffectiveHosts), 0)
	is.Equal(len(aliceIP.TrimmedHosts), 0)
	is.Equal(len(aliceIP.SharedWithUsers), 1)

	// Bob: restricted — normal effective hosts
	bob := result.Users[1]
	is.Equal(bob.BypassAllowlist, false)
	bobIP := bob.Ips[0]
	is.Equal(bobIP.EffectiveHosts, []string{"a.com"})
	is.Equal(len(bobIP.TrimmedHosts), 0)
}

// TestAssemblePolicyUserMap_NoAccessUser verifies that a user absent from the cache
// appears with empty ips, zero counts, nil last_seen_at, but populated bypass and hosts.
func TestAssemblePolicyUserMap_NoAccessUser(t *testing.T) {
	is := is.New(t)

	// Empty snapshot — no IP entries.
	snap := policy.PolicyMapSnapshot{
		LastRefreshedAt: baseTime,
	}

	allUsers := []policyAuditUserRow{
		{UserID: userCharlie, UserName: "Charlie", BypassAllowlist: false},
	}

	allowedHosts := map[auth.UserID][]string{
		userCharlie: {"x.com", "y.com"},
	}

	result := assemblePolicyUserMap(snap, map[device.AddressID]policyEnrichmentRow{}, allUsers, allowedHosts)

	is.Equal(len(result.Users), 1)
	charlie := result.Users[0]
	is.Equal(charlie.UserId, int64(userCharlie))
	is.Equal(charlie.UserName, "Charlie")
	is.Equal(charlie.BypassAllowlist, false)
	is.Equal(charlie.DeviceCount, 0)
	is.Equal(charlie.IpCount, 0)
	is.Equal(charlie.AllowedHostCount, 2)
	is.Equal(charlie.UserAllowedHosts, []string{"x.com", "y.com"})
	is.True(charlie.LastSeenAt == nil)
	is.Equal(len(charlie.Ips), 0)
}

// TestAssemblePolicyUserMap_SortOrder verifies alphabetic user ordering
// and lexicographic IP ordering.
func TestAssemblePolicyUserMap_SortOrder(t *testing.T) {
	is := is.New(t)

	addrZ := device.AddressID(300)
	addrA := device.AddressID(301)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{IP: "2.2.2.2", AllowedHosts: []string{}, Contributors: []policy.ContributorAccess{
				{DeviceID: devAlice1, AddressID: addrZ, UserID: userAlice, UserAllowedHosts: []string{}},
			}},
			{IP: "1.1.1.1", AllowedHosts: []string{}, Contributors: []policy.ContributorAccess{
				{DeviceID: devAlice2, AddressID: addrA, UserID: userAlice, UserAllowedHosts: []string{}},
			}},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrZ: makeEnrichment(addrZ, devAlice1, "dev-z", userAlice, "Alice", baseTime),
		addrA: makeEnrichment(addrA, devAlice2, "dev-a", userAlice, "Alice", baseTime),
	}

	// SQL order: Alice (1) before Bob (2) — Bob is no-access.
	allUsers := []policyAuditUserRow{
		{UserID: userAlice, UserName: "Alice"},
		{UserID: userBob, UserName: "Bob"},
	}

	result := assemblePolicyUserMap(snap, enrichment, allUsers, map[auth.UserID][]string{})

	// Users sorted alphabetically.
	is.Equal(result.Users[0].UserName, "Alice")
	is.Equal(result.Users[1].UserName, "Bob")

	// Alice's IPs sorted lexicographically.
	alice := result.Users[0]
	is.Equal(len(alice.Ips), 2)
	is.Equal(alice.Ips[0].Ip, "1.1.1.1")
	is.Equal(alice.Ips[1].Ip, "2.2.2.2")
}

// TestAssemblePolicyUserMap_FullIPBypass verifies that when entry.BypassAllowlist
// is true, every user at that IP gets BypassAtIp=true and empty effective/trimmed hosts,
// even if the user themselves is not a bypass user.
func TestAssemblePolicyUserMap_FullIPBypass(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{
				IP:              "192.168.1.1",
				BypassAllowlist: true,
				AllowedHosts:    []string{},
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice, UserBypass: false, UserAllowedHosts: []string{"a.com"}},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev", userAlice, "Alice", baseTime),
	}

	result := assemblePolicyUserMap(snap, enrichment,
		[]policyAuditUserRow{{UserID: userAlice, UserName: "Alice"}},
		map[auth.UserID][]string{},
	)

	is.Equal(len(result.Users), 1)
	aliceIP := result.Users[0].Ips[0]
	is.Equal(aliceIP.BypassAtIp, true)
	is.Equal(len(aliceIP.EffectiveHosts), 0)
	is.Equal(len(aliceIP.TrimmedHosts), 0)
}

// TestAssemblePolicyUserMap_DeviceDeduplicationAcrossIPs verifies that the same
// physical device contributing at two different IPs is counted once in DeviceCount.
func TestAssemblePolicyUserMap_DeviceDeduplicationAcrossIPs(t *testing.T) {
	is := is.New(t)

	addrAlice3 := device.AddressID(102) // second address for devAlice1, different IP

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{IP: "1.1.1.1", AllowedHosts: []string{}, Contributors: []policy.ContributorAccess{
				{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice},
			}},
			{IP: "2.2.2.2", AllowedHosts: []string{}, Contributors: []policy.ContributorAccess{
				{DeviceID: devAlice1, AddressID: addrAlice3, UserID: userAlice},
			}},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev", userAlice, "Alice", baseTime),
		addrAlice3: makeEnrichment(addrAlice3, devAlice1, "alice-dev", userAlice, "Alice", baseTime),
	}

	result := assemblePolicyUserMap(snap, enrichment,
		[]policyAuditUserRow{{UserID: userAlice, UserName: "Alice"}},
		map[auth.UserID][]string{},
	)

	alice := result.Users[0]
	is.Equal(alice.IpCount, 2)
	is.Equal(alice.DeviceCount, 1) // same device at two IPs
}

// TestAssemblePolicyUserMap_NoAccessBypassUser verifies that a bypass user absent
// from the cache gets AllowedHostCount=0 and empty UserAllowedHosts, not the DB list.
func TestAssemblePolicyUserMap_NoAccessBypassUser(t *testing.T) {
	is := is.New(t)

	result := assemblePolicyUserMap(
		policy.PolicyMapSnapshot{},
		map[device.AddressID]policyEnrichmentRow{},
		[]policyAuditUserRow{{UserID: userCharlie, UserName: "Charlie", BypassAllowlist: true}},
		map[auth.UserID][]string{userCharlie: {"x.com", "y.com"}},
	)

	charlie := result.Users[0]
	is.Equal(charlie.BypassAllowlist, true)
	is.Equal(charlie.AllowedHostCount, 0)
	is.Equal(charlie.UserAllowedHosts, []string{})
	is.Equal(len(charlie.Ips), 0)
}

// TestAssemblePolicyUserMap_Aggregates verifies the top-level aggregate counts:
// total_ip_count, total_device_count, total_host_count, shared_ip_count, and is_admin.
func TestAssemblePolicyUserMap_Aggregates(t *testing.T) {
	is := is.New(t)

	// Two users share one IP; Alice also has a second IP alone.
	// Alice: admin, hosts {a.com, b.com}
	// Bob: non-admin, hosts {a.com}
	// IPs: "10.0.0.1" (Alice+Bob shared), "10.0.0.2" (Alice only)
	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{
				IP:           "10.0.0.1",
				AllowedHosts: []string{"a.com"},
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice, UserAllowedHosts: []string{"a.com", "b.com"}},
					{DeviceID: devBob1, AddressID: addrBob1, UserID: userBob, UserAllowedHosts: []string{"a.com"}},
				},
			},
			{
				IP:           "10.0.0.2",
				AllowedHosts: []string{"a.com", "b.com"},
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice2, AddressID: addrAlice2, UserID: userAlice, UserAllowedHosts: []string{"a.com", "b.com"}},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev-1", userAlice, "Alice", baseTime),
		addrAlice2: makeEnrichment(addrAlice2, devAlice2, "alice-dev-2", userAlice, "Alice", baseTime),
		addrBob1:   makeEnrichment(addrBob1, devBob1, "bob-dev", userBob, "Bob", baseTime),
	}

	allUsers := []policyAuditUserRow{
		{UserID: userAlice, UserName: "Alice", Username: "alice", IsAdmin: true},
		{UserID: userBob, UserName: "Bob", Username: "bob", IsAdmin: false},
	}

	result := assemblePolicyUserMap(snap, enrichment, allUsers, map[auth.UserID][]string{})

	// Aggregate assertions.
	is.Equal(result.TotalIpCount, 2)     // "10.0.0.1" and "10.0.0.2"
	is.Equal(result.TotalDeviceCount, 3) // devAlice1 + devAlice2 + devBob1
	is.Equal(result.TotalHostCount, 2)   // union of {a.com, b.com} ∪ {a.com} = {a.com, b.com}
	is.Equal(result.SharedIpCount, 1)    // only "10.0.0.1" is shared

	// IsAdmin propagation.
	is.Equal(result.Users[0].IsAdmin, true)  // Alice
	is.Equal(result.Users[1].IsAdmin, false) // Bob

	// SharedWithUsers enrichment on "10.0.0.1": Alice sees Bob with his device.
	aliceAt1 := result.Users[0].Ips[0] // Alice's first IP is "10.0.0.1" (lexicographic)
	is.Equal(len(aliceAt1.SharedWithUsers), 1)
	is.Equal(aliceAt1.SharedWithUsers[0].UserId, int64(userBob))
	is.Equal(aliceAt1.SharedWithUsers[0].Username, "bob")
	is.Equal(aliceAt1.SharedWithUsers[0].UserName, "Bob")
	is.Equal(len(aliceAt1.SharedWithUsers[0].Devices), 1)
	is.Equal(aliceAt1.SharedWithUsers[0].Devices[0].DeviceId, int64(devBob1))
	is.Equal(aliceAt1.SharedWithUsers[0].Devices[0].DeviceName, "bob-dev")
}

// ── buildIPIndex unit tests ────────────────────────────────────────────────────

// TestBuildIPIndex_SkipsAbsentEnrichment verifies that a cache contributor whose
// address has no enrichment row (e.g. deleted address) is silently skipped.
func TestBuildIPIndex_SkipsAbsentEnrichment(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{IP: "1.2.3.4", Contributors: []policy.ContributorAccess{
				{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice},
			}},
		},
	}

	byUser, usersAtIP := buildIPIndex(snap, map[device.AddressID]policyEnrichmentRow{})

	is.Equal(len(byUser), 0)
	is.Equal(len(usersAtIP["1.2.3.4"]), 0)
}

// TestBuildIPIndex_UsersAtIPTracking verifies that every distinct user contributing
// at an IP is recorded in usersAtIP, regardless of enrichment gaps for other users.
func TestBuildIPIndex_UsersAtIPTracking(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{IP: "10.0.0.1", Contributors: []policy.ContributorAccess{
				{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice},
				{DeviceID: devBob1, AddressID: addrBob1, UserID: userBob},
			}},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev", userAlice, "Alice", baseTime),
		addrBob1:   makeEnrichment(addrBob1, devBob1, "bob-dev", userBob, "Bob", baseTime),
	}

	byUser, usersAtIP := buildIPIndex(snap, enrichment)

	is.Equal(len(usersAtIP["10.0.0.1"]), 2)
	_, alicePresent := usersAtIP["10.0.0.1"][userAlice]
	_, bobPresent := usersAtIP["10.0.0.1"][userBob]
	is.True(alicePresent)
	is.True(bobPresent)
	is.Equal(len(byUser[userAlice]["10.0.0.1"].addresses), 1)
	is.Equal(len(byUser[userBob]["10.0.0.1"].addresses), 1)
}

// TestBuildIPIndex_MultipleAddressesSameUserIP verifies that a user with two
// devices at the same IP gets one bucket with two addresses, and that the
// user-level fields (userBypass, userAllowedHosts) are set from the first contributor.
func TestBuildIPIndex_MultipleAddressesSameUserIP(t *testing.T) {
	is := is.New(t)

	snap := policy.PolicyMapSnapshot{
		Entries: []policy.PolicyMapEntry{
			{
				IP:           "10.0.0.1",
				AllowedHosts: []string{"a.com"},
				Contributors: []policy.ContributorAccess{
					{DeviceID: devAlice1, AddressID: addrAlice1, UserID: userAlice, UserBypass: false, UserAllowedHosts: []string{"a.com"}},
					{DeviceID: devAlice2, AddressID: addrAlice2, UserID: userAlice, UserBypass: false, UserAllowedHosts: []string{"a.com"}},
				},
			},
		},
	}

	enrichment := map[device.AddressID]policyEnrichmentRow{
		addrAlice1: makeEnrichment(addrAlice1, devAlice1, "alice-dev-1", userAlice, "Alice", baseTime),
		addrAlice2: makeEnrichment(addrAlice2, devAlice2, "alice-dev-2", userAlice, "Alice", baseTime.Add(time.Minute)),
	}

	byUser, _ := buildIPIndex(snap, enrichment)

	bucket := byUser[userAlice]["10.0.0.1"]
	is.True(bucket != nil)
	is.Equal(len(bucket.addresses), 2)
	is.Equal(bucket.userBypass, false)
	is.Equal(bucket.userAllowedHosts, []string{"a.com"})
}
