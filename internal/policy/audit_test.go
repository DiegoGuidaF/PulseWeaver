//go:build test

package policy

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

func TestGetPolicyMap_RefreshMetadata(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 1}}
	before := time.Now()
	svc := newRestrictedService(entries, []UserHostAccess{{UserID: 1, BypassAllowlist: true}})
	after := time.Now()
	snap := svc.GetPolicyMap()
	is.True(!snap.LastRefreshedAt.IsZero())
	is.True(!snap.LastRefreshedAt.Before(before))
	is.True(!snap.LastRefreshedAt.After(after))
	is.True(snap.LastRefreshDurationMs >= 0)
}

func TestGetPolicyMap_ContributorPreIntersectionState(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1), UserID: auth.UserID(1)},
	}
	hostAccess := []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"b.com", "a.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)
	snap := svc.GetPolicyMap()
	is.Equal(len(snap.Entries), 1)
	is.Equal(len(snap.Entries[0].Contributors), 1)
	c := snap.Entries[0].Contributors[0]
	is.True(!c.UserBypass)
	// UserAllowedHosts must be sorted
	is.Equal(c.UserAllowedHosts, []string{"a.com", "b.com"})
}
