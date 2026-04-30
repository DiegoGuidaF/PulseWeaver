//go:build test

package policy

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
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
