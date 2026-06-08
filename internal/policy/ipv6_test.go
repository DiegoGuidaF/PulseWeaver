//go:build test

package policy

import (
	"context"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/matryer/is"
)

// mapped4in6 returns the IPv4-mapped IPv6 form (::ffff:a.b.c.d) of a plain IPv4
// address, so tests can present the same address in both representations.
func mapped4in6(t *testing.T, v4 string) netip.Addr {
	t.Helper()
	return netip.AddrFrom16(netip.MustParseAddr(v4).As16())
}

// ── PW-67: mapped-v4 / plain-v4 symmetry ─────────────────────────────────────

func TestDecide_Symmetry_MappedV4_DeviceMatch(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "10.1.0.1", DeviceID: 1, AddressID: 1, UserID: 1}}
	hostAccess := []UserHostAccess{{UserID: 1, BypassAllowlist: true}}
	svc := newRestrictedService(entries, hostAccess)

	plain := svc.Decide(context.Background(), netip.MustParseAddr("10.1.0.1"), "api.internal")
	mapped := svc.Decide(context.Background(), mapped4in6(t, "10.1.0.1"), "api.internal")

	is.True(plain.Allowed) // sanity: plain form is allowed
	is.Equal(plain.Allowed, mapped.Allowed)
	is.Equal(plain.MatchSource, mapped.MatchSource)
}

func TestDecide_Symmetry_MappedV4_CIDRMatch(t *testing.T) {
	is := is.New(t)
	svc := newServiceWithNetworkPolicies(nil, nil, []networkpolicies.CacheEntry{
		{PolicyID: ids.NetworkPolicyID(1), PolicyName: "corp", CIDR: "192.168.0.0/16", BypassHostCheck: true},
	})

	plain := svc.Decide(context.Background(), netip.MustParseAddr("192.168.5.5"), "any.host")
	mapped := svc.Decide(context.Background(), mapped4in6(t, "192.168.5.5"), "any.host")

	is.True(plain.Allowed)
	is.Equal(plain.Allowed, mapped.Allowed)
	is.Equal(plain.MatchSource, mapped.MatchSource)
}

// A network policy stored as an IPv4-mapped IPv6 prefix must still match plain v4.
func TestDecide_Mapped4in6CIDRPolicy_MatchesPlainV4(t *testing.T) {
	is := is.New(t)
	svc := newServiceWithNetworkPolicies(nil, nil, []networkpolicies.CacheEntry{
		{PolicyID: ids.NetworkPolicyID(1), PolicyName: "weird", CIDR: "::ffff:10.0.0.0/104", BypassHostCheck: true},
	})

	res := svc.Decide(context.Background(), netip.MustParseAddr("10.5.5.5"), "any.host")
	is.True(res.Allowed)
	is.Equal(res.MatchSource, MatchSourceNetworkPolicy)
}

// A stored 4-in-6 device row and its plain-v4 twin collapse onto one cache entry.
func TestBuildIPSet_MappedAndPlainTwin_CollapseToOneEntry(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "10.1.0.1", DeviceID: 1, AddressID: 1, UserID: 1},
		{IP: "::ffff:10.1.0.1", DeviceID: 2, AddressID: 2, UserID: 2},
	}
	hostAccess := []UserHostAccess{
		{UserID: 1, BypassAllowlist: true},
		{UserID: 2, BypassAllowlist: true},
	}
	result := buildIPSet(entries, hostAccess)
	is.Equal(len(result), 1)
	is.Equal(len(result[mustAddr("10.1.0.1")].Contributors), 2)
}

// ── PW-67: native IPv6 support ───────────────────────────────────────────────

func TestDecide_NativeV6_DeviceMatch(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "2001:db8::1", DeviceID: 1, AddressID: 1, UserID: 1}}
	hostAccess := []UserHostAccess{{UserID: 1, BypassAllowlist: true}}
	svc := newRestrictedService(entries, hostAccess)

	res := svc.Decide(context.Background(), netip.MustParseAddr("2001:db8::1"), "api.internal")
	is.True(res.Allowed)
	is.Equal(res.MatchSource, MatchSourceDevice)

	// A different v6 address is not registered.
	is.True(!svc.Decide(context.Background(), netip.MustParseAddr("2001:db8::2"), "api.internal").Allowed)
}

func TestDecide_NativeV6_CIDRMatch(t *testing.T) {
	is := is.New(t)
	svc := newServiceWithNetworkPolicies(nil, nil, []networkpolicies.CacheEntry{
		{PolicyID: ids.NetworkPolicyID(1), PolicyName: "v6-net", CIDR: "2001:db8::/32", BypassHostCheck: true},
	})

	res := svc.Decide(context.Background(), netip.MustParseAddr("2001:db8::dead:beef"), "any.host")
	is.True(res.Allowed)
	is.Equal(res.MatchSource, MatchSourceNetworkPolicy)

	// Outside the prefix is denied.
	is.True(!svc.Decide(context.Background(), netip.MustParseAddr("2001:dead::1"), "any.host").Allowed)
}

// ── PW-67: fail-closed on invalid address ────────────────────────────────────

func TestDecide_InvalidAddr_Denied(t *testing.T) {
	is := is.New(t)
	svc := newServiceWithNetworkPolicies(nil, nil, []networkpolicies.CacheEntry{
		{PolicyID: ids.NetworkPolicyID(1), CIDR: "0.0.0.0/0", BypassHostCheck: true},
	})
	// The zero Addr (e.g. from an unparseable inbound IP) must deny even though a
	// catch-all CIDR policy is present.
	res := svc.Decide(context.Background(), netip.Addr{}, "any.host")
	is.True(!res.Allowed)
	is.Equal(*res.DenyReason, DenyReasonIPNotRegistered)
}

// ── PW-67: fuzz — symmetry + fail-closed property ────────────────────────────

func FuzzDecide_MappedSymmetryAndFailClosed(f *testing.F) {
	svc := newServiceWithNetworkPolicies(
		[]device.IPEntry{{IP: "10.1.0.1", DeviceID: 1, AddressID: 1, UserID: 1}},
		[]UserHostAccess{{UserID: 1, BypassAllowlist: true}},
		[]networkpolicies.CacheEntry{
			{PolicyID: ids.NetworkPolicyID(1), PolicyName: "corp", CIDR: "192.168.0.0/16", BypassHostCheck: true},
			{PolicyID: ids.NetworkPolicyID(2), PolicyName: "v6", CIDR: "2001:db8::/32", BypassHostCheck: true},
		},
	)

	for _, s := range []string{
		"10.1.0.1", "192.168.5.5", "9.9.9.9", "2001:db8::1",
		"2001:dead::1", "::ffff:10.1.0.1", "", "not-an-ip", "256.0.0.1",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, ipStr string) {
		addr, err := netip.ParseAddr(ipStr)
		if err != nil {
			// Unparseable input yields the zero Addr, which must always deny.
			if svc.Decide(context.Background(), addr, "any.host").Allowed {
				t.Fatalf("invalid IP %q was allowed", ipStr)
			}
			return
		}

		// The plain (unmapped) and 4-in-6 forms of the same address must decide
		// identically — representation never changes the outcome.
		plain := svc.Decide(context.Background(), addr.Unmap(), "any.host")
		mapped := svc.Decide(context.Background(), netip.AddrFrom16(addr.As16()), "any.host")
		if plain.Allowed != mapped.Allowed {
			t.Fatalf("asymmetric decision for %q: plain=%v mapped=%v", ipStr, plain.Allowed, mapped.Allowed)
		}
		if plain.MatchSource != mapped.MatchSource {
			t.Fatalf("asymmetric match source for %q: plain=%v mapped=%v", ipStr, plain.MatchSource, mapped.MatchSource)
		}
	})
}
