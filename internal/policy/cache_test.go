//go:build test

package policy

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/matryer/is"
)

// ── buildIPSet: deny-wins intersection ───────────────────────────────────────

func TestBuildIPSet_IntersectionApplied_TwoRestrictedUsers(t *testing.T) {
	is := is.New(t)
	// userA allows {a.com, b.com}, userB allows {b.com} → intersection {b.com}.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.True(entry.IntersectionApplied)
	is.Equal(sortedKeys(entry.AllowedHosts), []string{"b.com"})
}

func TestBuildIPSet_IntersectionNotApplied_SingleRestrictedUser(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)}}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	is.True(!result[mustAddr("1.2.3.4")].IntersectionApplied)
}

func TestBuildIPSet_IntersectionNotApplied_BypassUserShared(t *testing.T) {
	is := is.New(t)
	// Bypass users are intersection-neutral; the restricted user's host set is left untouched.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: true},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	is.True(!result[mustAddr("1.2.3.4")].IntersectionApplied)
}

func TestBuildIPSet_IntersectionApplied_ThreeRestrictedUsers(t *testing.T) {
	is := is.New(t)
	// A∩B narrows to {b,c}, then ∩C narrows to {c}. Intersection is chained.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
		{IP: "1.2.3.4", DeviceID: 3, AddressID: 3, UserID: ids.UserID(3)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com", "c.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com", "c.com"}},
		{UserID: ids.UserID(3), BypassAllowlist: false, AllowedHosts: []string{"c.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.True(entry.IntersectionApplied)
	is.Equal(sortedKeys(entry.AllowedHosts), []string{"c.com"})
}

func TestBuildIPSet_IntersectionApplied_DisjointSets_EmptyResult(t *testing.T) {
	is := is.New(t)
	// No host in common: intersection shrinks from size 1 → 0; IntersectionApplied = true.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.True(entry.IntersectionApplied)
	is.Equal(sortedKeys(entry.AllowedHosts), []string{})
}

func TestBuildIPSet_IntersectionNotApplied_IdenticalSets(t *testing.T) {
	is := is.New(t)
	// Same host set for both users: intersection doesn't shrink → IntersectionApplied = false.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.True(!entry.IntersectionApplied)
	is.Equal(sortedKeys(entry.AllowedHosts), []string{"a.com", "b.com"})
}

// ── buildIPSet: BypassAllowlist entry flag ───────────────────────────────────

func TestBuildIPSet_AllBypass_EntryBypassIsTrue(t *testing.T) {
	is := is.New(t)
	// allBypass AND-reduces across contributors; unanimous bypass → entry flag true.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: true},
		{UserID: ids.UserID(2), BypassAllowlist: true},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.True(entry.BypassAllowlist)
	// Bypass path never populates AllowedHosts.
	is.Equal(len(entry.AllowedHosts), 0)
}

func TestBuildIPSet_MixedBypass_EntryBypassIsFalse(t *testing.T) {
	is := is.New(t)
	// One bypass + one restricted: AND-reduction breaks unanimity → entry flag false.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: true},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	is.True(!result[mustAddr("1.2.3.4")].BypassAllowlist)
}

// ── buildIPSet: Contributors slice ───────────────────────────────────────────

func TestBuildIPSet_Contributors_SingleUser_FieldsCorrect(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 42, AddressID: 99, UserID: ids.UserID(7)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(7), BypassAllowlist: false, AllowedHosts: []string{"z.com", "a.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	is.Equal(len(result[mustAddr("1.2.3.4")].Contributors), 1)
	c := result[mustAddr("1.2.3.4")].Contributors[0]
	is.Equal(int64(c.DeviceID), int64(42))
	is.Equal(int64(c.AddressID), int64(99))
	is.Equal(int64(c.UserID), int64(7))
	is.True(!c.UserBypass)
	// UserAllowedHosts is sorted lexicographically regardless of input order.
	is.Equal(c.UserAllowedHosts, []string{"a.com", "z.com"})
}

func TestBuildIPSet_Contributors_BypassUser_UserBypassTrue(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 5, AddressID: 6, UserID: ids.UserID(3)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(3), BypassAllowlist: true},
	}
	result := buildIPSet(entries, hostAccess)
	c := result[mustAddr("1.2.3.4")].Contributors[0]
	is.True(c.UserBypass)
	is.Equal(c.UserAllowedHosts, []string{})
}

func TestBuildIPSet_Contributors_TwoUsers_BothPresent(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	is.Equal(len(result[mustAddr("1.2.3.4")].Contributors), 2)
}

func TestBuildIPSet_Contributors_SameUser_TwoDevices_BothPresent(t *testing.T) {
	is := is.New(t)
	// Same user, two devices at the same IP. Second intersection with identical set is
	// a no-op: initialHostsLen == len(allowedHosts) → IntersectionApplied = false.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 10, AddressID: 20, UserID: ids.UserID(5)},
		{IP: "1.2.3.4", DeviceID: 11, AddressID: 21, UserID: ids.UserID(5)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(5), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.Equal(len(entry.Contributors), 2)
	is.True(!entry.IntersectionApplied)
	is.Equal(sortedKeys(entry.AllowedHosts), []string{"a.com"})
}

// ── buildIPSet: UserAllowedHosts reflects pre-intersection state ──────────────

func TestBuildIPSet_ContributorHosts_PreIntersectionState(t *testing.T) {
	is := is.New(t)
	// After intersection the entry allows only {b.com}, but each contributor's
	// UserAllowedHosts must still reflect their individual pre-intersection grants.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com", "c.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	entry := result[mustAddr("1.2.3.4")]
	is.Equal(sortedKeys(entry.AllowedHosts), []string{"b.com"})
	is.True(entry.IntersectionApplied)

	byUser := make(map[ids.UserID]ContributorAccess, 2)
	for _, c := range entry.Contributors {
		byUser[c.UserID] = c
	}
	is.Equal(byUser[ids.UserID(1)].UserAllowedHosts, []string{"a.com", "b.com", "c.com"})
	is.Equal(byUser[ids.UserID(2)].UserAllowedHosts, []string{"b.com"})
}

// ── buildIPSet: multiple independent IPs ─────────────────────────────────────

func TestBuildIPSet_MultipleIPs_IndependentEntries(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
		{IP: "5.6.7.8", DeviceID: 2, AddressID: 2, UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	result := buildIPSet(entries, hostAccess)
	is.Equal(len(result), 2)

	e1 := result[mustAddr("1.2.3.4")]
	is.Equal(sortedKeys(e1.AllowedHosts), []string{"a.com"})
	is.Equal(len(e1.Contributors), 1)
	is.Equal(int64(e1.Contributors[0].UserID), int64(1))

	e2 := result[mustAddr("5.6.7.8")]
	is.Equal(sortedKeys(e2.AllowedHosts), []string{"b.com"})
	is.Equal(len(e2.Contributors), 1)
	is.Equal(int64(e2.Contributors[0].UserID), int64(2))
}

func TestBuildIPSet_NoHostAccess_UserTreatedAsRestricted(t *testing.T) {
	is := is.New(t)
	// With no host access data every user maps to zero-value UserHostAccess{BypassAllowlist: false}.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
	}
	result := buildIPSet(entries, nil)
	entry := result[mustAddr("1.2.3.4")]
	is.True(!entry.BypassAllowlist)
	is.Equal(len(entry.AllowedHosts), 0)
}

// ── buildNetworkPolicyCache ───────────────────────────────────────────────────

func TestBuildNetworkPolicyCache_SortedMostSpecificFirst(t *testing.T) {
	is := is.New(t)
	entries := []networkpolicies.CacheEntry{
		{PolicyID: 1, PolicyName: "broad", CIDR: "10.0.0.0/8"},
		{PolicyID: 2, PolicyName: "specific", CIDR: "10.1.2.0/24"},
		{PolicyID: 3, PolicyName: "mid", CIDR: "10.1.0.0/16"},
	}
	result := buildNetworkPolicyCache(context.Background(), entries, noopLogger())
	is.Equal(len(result), 3)
	is.Equal(result[0].Prefix.Bits(), 24)
	is.Equal(result[1].Prefix.Bits(), 16)
	is.Equal(result[2].Prefix.Bits(), 8)
}

func TestBuildNetworkPolicyCache_InvalidCIDR_Skipped(t *testing.T) {
	is := is.New(t)
	entries := []networkpolicies.CacheEntry{
		{PolicyID: 1, CIDR: "not-a-cidr"},
		{PolicyID: 2, CIDR: "192.168.1.0/24"},
	}
	result := buildNetworkPolicyCache(context.Background(), entries, noopLogger())
	is.Equal(len(result), 1)
	is.Equal(result[0].PolicyID.Int64(), int64(2))
}

func TestBuildNetworkPolicyCache_HostBitsNormalized(t *testing.T) {
	is := is.New(t)
	// "192.168.1.5/24" has host bits set; Masked() must normalize it to 192.168.1.0/24.
	entries := []networkpolicies.CacheEntry{
		{PolicyID: 1, CIDR: "192.168.1.5/24"},
	}
	result := buildNetworkPolicyCache(context.Background(), entries, noopLogger())
	is.Equal(len(result), 1)
	is.Equal(result[0].Prefix.String(), "192.168.1.0/24")
}

func TestBuildNetworkPolicyCache_AllowedHostFQDNs_DeduplicatedToSet(t *testing.T) {
	is := is.New(t)
	entries := []networkpolicies.CacheEntry{
		{PolicyID: 1, CIDR: "10.0.0.0/8", AllowedHostFQDNs: []string{"a.com", "b.com", "a.com"}},
	}
	result := buildNetworkPolicyCache(context.Background(), entries, noopLogger())
	is.Equal(len(result[0].AllowedHosts), 2)
}

func TestBuildNetworkPolicyCache_BypassHostCheck_Preserved(t *testing.T) {
	is := is.New(t)
	entries := []networkpolicies.CacheEntry{
		{PolicyID: 1, CIDR: "10.0.0.0/8", BypassHostCheck: true},
		{PolicyID: 2, CIDR: "192.168.0.0/16", BypassHostCheck: false},
	}
	result := buildNetworkPolicyCache(context.Background(), entries, noopLogger())
	is.Equal(len(result), 2)
	// After sort: /16 is more specific, comes first.
	is.Equal(result[0].Prefix.Bits(), 16)
	is.True(!result[0].BypassHostCheck)
	is.Equal(result[1].Prefix.Bits(), 8)
	is.True(result[1].BypassHostCheck)
}

// ── Service-level: error propagation ─────────────────────────────────────────

func TestCache_HostProviderError_Propagated(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: ids.UserID(1)},
	}}
	hostProv := &errHostProvider{err: errors.New("db unavailable")}
	svc, err := NewService(provider, hostProv, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
	// Cache must stay empty: the failed refresh must not partially populate it.
	is.Equal(len(svc.GetPolicyMap().Entries), 0)
}

// ── sortedKeys ────────────────────────────────────────────────────────────────

func TestSortedKeys_NilMap(t *testing.T) {
	is := is.New(t)
	result := sortedKeys(nil)
	// Must return an empty slice (not nil) to keep callers' equality checks clean.
	is.Equal(result, []string{})
}

func TestSortedKeys_MultipleKeys_Sorted(t *testing.T) {
	is := is.New(t)
	m := map[string]struct{}{"z.com": {}, "a.com": {}, "m.com": {}}
	result := sortedKeys(m)
	is.Equal(result, []string{"a.com", "m.com", "z.com"})
}

// ── intersectHostSets ─────────────────────────────────────────────────────────

func TestIntersectHostSets_RemovesAbsent(t *testing.T) {
	is := is.New(t)
	dst := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	src := map[string]struct{}{"b": {}, "c": {}, "d": {}}
	intersectHostSets(dst, src)
	is.Equal(len(dst), 2)
	_, hasA := dst["a"]
	_, hasB := dst["b"]
	_, hasC := dst["c"]
	is.True(!hasA)
	is.True(hasB)
	is.True(hasC)
}

func TestIntersectHostSets_EmptySrc_ClearsDst(t *testing.T) {
	is := is.New(t)
	dst := map[string]struct{}{"a": {}, "b": {}}
	intersectHostSets(dst, map[string]struct{}{})
	is.Equal(len(dst), 0)
}

func TestIntersectHostSets_EmptyDst_NoOp(t *testing.T) {
	is := is.New(t)
	dst := map[string]struct{}{}
	intersectHostSets(dst, map[string]struct{}{"a": {}})
	is.Equal(len(dst), 0)
}
