//go:build test

package policy

import (
	"context"
	"errors"
	"net/netip"
	"sort"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/matryer/is"
)

// ── Deny-wins intersection ────────────────────────────────────────────────────

func TestDecide_IntersectionApplied_TwoRestrictedUsers(t *testing.T) {
	is := is.New(t)
	// Two users share 1.2.3.4; userA allows {a.com, b.com}, userB allows {b.com}.
	// Intersection → {b.com}; IntersectionApplied must be true.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)

	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.True(entry.IntersectionApplied)
	is.Equal(entry.AllowedHosts, []string{"b.com"})
}

func TestDecide_IntersectionNotApplied_SingleRestrictedUser(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)}}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	is.True(!s.Entries[0].IntersectionApplied)
}

func TestDecide_IntersectionNotApplied_BypassUserShared(t *testing.T) {
	is := is.New(t)
	// Bypass user + restricted user on same IP: bypass doesn't intersect, so no shrink.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: true},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	is.True(!s.Entries[0].IntersectionApplied)
}

func TestCache_IntersectionApplied_ThreeRestrictedUsers(t *testing.T) {
	is := is.New(t)
	// A∩B narrows to {b,c}, then ∩C narrows to {c}. Intersection is chained.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
		{IP: "1.2.3.4", DeviceID: 3, AddressID: 3, UserID: auth.UserID(3)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com", "c.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com", "c.com"}},
		{UserID: auth.UserID(3), BypassAllowlist: false, AllowedHosts: []string{"c.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.True(entry.IntersectionApplied)
	is.Equal(entry.AllowedHosts, []string{"c.com"})
}

func TestCache_IntersectionApplied_DisjointSets_EmptyResult(t *testing.T) {
	is := is.New(t)
	// No host in common: intersection shrinks from size 1 → 0; IntersectionApplied = true.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.True(entry.IntersectionApplied)
	is.Equal(entry.AllowedHosts, []string{})
}

func TestCache_IntersectionNotApplied_IdenticalSets(t *testing.T) {
	is := is.New(t)
	// Same host set for both users: intersection doesn't shrink → IntersectionApplied = false.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.True(!entry.IntersectionApplied)
	is.Equal(entry.AllowedHosts, []string{"a.com", "b.com"})
}

// ── BypassAllowlist entry flag ────────────────────────────────────────────────

func TestCache_AllBypass_EntryBypassIsTrue(t *testing.T) {
	is := is.New(t)
	// allBypass AND-reduces across contributors; unanimous bypass → entry flag true.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: true},
		{UserID: auth.UserID(2), BypassAllowlist: true},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.True(entry.BypassAllowlist)
	// Bypass path never sets AllowedHosts — sorted nil map returns empty slice.
	is.Equal(entry.AllowedHosts, []string{})
}

func TestCache_MixedBypass_EntryBypassIsFalse(t *testing.T) {
	is := is.New(t)
	// One bypass + one restricted: AND-reduction breaks unanimity → entry flag false.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: true},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	is.True(!s.Entries[0].BypassAllowlist)
}

// ── Contributors slice ────────────────────────────────────────────────────────

func TestCache_Contributors_SingleUser_FieldsCorrect(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 42, AddressID: 99, UserID: auth.UserID(7)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(7), BypassAllowlist: false, AllowedHosts: []string{"z.com", "a.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	is.Equal(len(s.Entries[0].Contributors), 1)
	c := s.Entries[0].Contributors[0]
	is.Equal(int64(c.DeviceID), int64(42))
	is.Equal(int64(c.AddressID), int64(99))
	is.Equal(int64(c.UserID), int64(7))
	is.True(!c.UserBypass)
	// UserAllowedHosts is sorted lexicographically regardless of input order.
	is.Equal(c.UserAllowedHosts, []string{"a.com", "z.com"})
}

func TestCache_Contributors_BypassUser_UserBypassTrue(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 5, AddressID: 6, UserID: auth.UserID(3)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(3), BypassAllowlist: true},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries[0].Contributors), 1)
	c := s.Entries[0].Contributors[0]
	is.True(c.UserBypass)
	is.Equal(c.UserAllowedHosts, []string{})
}

func TestCache_Contributors_TwoUsers_BothPresent(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	is.Equal(len(s.Entries[0].Contributors), 2)
}

func TestCache_Contributors_SameUser_TwoDevices_BothPresent(t *testing.T) {
	is := is.New(t)
	// Same user, two devices at the same IP. Second intersection with identical set is
	// a no-op: firstRestrictedSize == len(result) → IntersectionApplied = false.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 10, AddressID: 20, UserID: auth.UserID(5)},
		{IP: "1.2.3.4", DeviceID: 11, AddressID: 21, UserID: auth.UserID(5)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(5), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.Equal(len(entry.Contributors), 2)
	is.True(!entry.IntersectionApplied)
	is.Equal(entry.AllowedHosts, []string{"a.com"})
}

// ── UserAllowedHosts is pre-intersection state ────────────────────────────────

func TestCache_ContributorHosts_PreIntersectionState(t *testing.T) {
	is := is.New(t)
	// After intersection the entry allows only {b.com}, but each contributor's
	// UserAllowedHosts must still reflect their individual pre-intersection grants.
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com", "c.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.Equal(entry.AllowedHosts, []string{"b.com"})
	is.True(entry.IntersectionApplied)

	byUser := make(map[auth.UserID]ContributorAccess, 2)
	for _, c := range entry.Contributors {
		byUser[c.UserID] = c
	}
	is.Equal(byUser[auth.UserID(1)].UserAllowedHosts, []string{"a.com", "b.com", "c.com"})
	is.Equal(byUser[auth.UserID(2)].UserAllowedHosts, []string{"b.com"})
}

// ── Multiple independent IPs ──────────────────────────────────────────────────

func TestCache_MultipleIPs_IndependentEntries(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
		{IP: "5.6.7.8", DeviceID: 2, AddressID: 2, UserID: auth.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 2)

	// Sort by IP so assertions are deterministic despite map iteration order.
	sort.Slice(s.Entries, func(i, j int) bool { return s.Entries[i].IP < s.Entries[j].IP })

	is.Equal(s.Entries[0].IP, "1.2.3.4")
	is.Equal(s.Entries[0].AllowedHosts, []string{"a.com"})
	is.Equal(len(s.Entries[0].Contributors), 1)
	is.Equal(int64(s.Entries[0].Contributors[0].UserID), int64(1))

	is.Equal(s.Entries[1].IP, "5.6.7.8")
	is.Equal(s.Entries[1].AllowedHosts, []string{"b.com"})
	is.Equal(len(s.Entries[1].Contributors), 1)
	is.Equal(int64(s.Entries[1].Contributors[0].UserID), int64(2))
}

// ── Error propagation ─────────────────────────────────────────────────────────

func TestCache_HostProviderError_Propagated(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
	}}
	hostProv := &errHostProvider{err: errors.New("db unavailable")}
	svc, err := NewService(provider, hostProv, &geoip.Lookup{}, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
	// Cache must stay empty: the failed refresh must not partially populate it.
	is.Equal(len(svc.GetPolicyMap().Entries), 0)
}

// ── Nil hostProvider ──────────────────────────────────────────────────────────

func TestCache_NilHostProvider_AllUsersTreatedAsRestricted(t *testing.T) {
	is := is.New(t)
	// With no hostProvider the fetch is skipped, so every user maps to the zero-value
	// UserHostAccess{BypassAllowlist: false}. The entry is not bypass and has no allowed
	// hosts, meaning all host checks will be denied.
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: auth.UserID(1)},
	}}
	svc, err := NewService(provider, nil, &geoip.Lookup{}, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	s := svc.GetPolicyMap()
	is.Equal(len(s.Entries), 1)
	entry := s.Entries[0]
	is.True(!entry.BypassAllowlist)
	is.Equal(entry.AllowedHosts, []string{})
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

// ── cloneHostSet ──────────────────────────────────────────────────────────────

func TestCloneHostSet_EmptyMap(t *testing.T) {
	is := is.New(t)
	result := cloneHostSet(map[string]struct{}{})
	is.Equal(len(result), 0)
	// Must not return nil; callers rely on len/range being safe.
	is.True(result != nil)
}

func TestCloneHostSet_Independence(t *testing.T) {
	is := is.New(t)
	src := map[string]struct{}{"a.com": {}, "b.com": {}}
	clone := cloneHostSet(src)

	// Mutating the clone must not affect the source.
	clone["c.com"] = struct{}{}
	is.Equal(len(src), 2)

	// Mutating the source must not affect the clone (which now has 3 entries).
	src["d.com"] = struct{}{}
	is.Equal(len(clone), 3)
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
