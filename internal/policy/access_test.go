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
	"github.com/matryer/is"
)

// ── lookupIP ──────────────────────────────────────────────────────────────────

func TestLookupIP_EmptySet(t *testing.T) {
	is := is.New(t)
	svc := newRestrictedService(nil, nil)
	_, ok := svc.lookupIP(context.Background(), "1.2.3.4")
	is.True(!ok)
}

func TestLookupIP_IPFound(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 1}}
	hostAccess := []UserHostAccess{{UserID: 1, BypassAllowlist: true}}
	svc := newRestrictedService(entries, hostAccess)
	entry, ok := svc.lookupIP(context.Background(), "1.2.3.4")
	is.True(ok)
	is.True(entry.BypassAllowlist)
}

func TestLookupIP_IPNotFound(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 1}}
	svc := newRestrictedService(entries, nil)
	_, ok := svc.lookupIP(context.Background(), "9.9.9.9")
	is.True(!ok)
}

func TestLookupIP_RejectsTrustedProxy(t *testing.T) {
	is := is.New(t)
	proxy := netip.MustParseAddr("10.0.0.1")
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "10.0.0.1", DeviceID: 1, AddressID: 1, UserID: 1},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), proxy)
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))
	_, ok := svc.lookupIP(context.Background(), "10.0.0.1")
	is.True(!ok)
}

// ── toIPContributors ──────────────────────────────────────────────────────────

func TestToIPContributors_Empty(t *testing.T) {
	is := is.New(t)
	is.True(toIPContributors(nil) == nil)
	is.True(toIPContributors([]ContributorAccess{}) == nil)
}

func TestToIPContributors_ProjectsFields(t *testing.T) {
	is := is.New(t)
	cs := []ContributorAccess{
		{DeviceID: ids.DeviceID(10), AddressID: ids.AddressID(20), UserID: ids.UserID(30), UserBypass: true, UserAllowedHosts: []string{"x.com"}},
		{DeviceID: ids.DeviceID(11), AddressID: ids.AddressID(21), UserID: ids.UserID(31)},
	}
	result := toIPContributors(cs)
	is.Equal(len(result), 2)
	is.Equal(int64(result[0].DeviceID), int64(10))
	is.Equal(int64(result[0].AddressID), int64(20))
	is.Equal(int64(result[0].UserID), int64(30))
	is.Equal(int64(result[1].DeviceID), int64(11))
	is.Equal(int64(result[1].AddressID), int64(21))
	is.Equal(int64(result[1].UserID), int64(31))
}

// ── Decide ────────────────────────────────────────────────────────────────────

func TestDecide_IPNotRegistered(t *testing.T) {
	is := is.New(t)
	svc := newRestrictedService(nil, nil)
	result := svc.Decide(context.Background(), "1.2.3.4", "example.com")
	is.True(!result.Allowed)
	is.True(result.DenyReason != nil)
	is.Equal(*result.DenyReason, DenyReasonIPNotRegistered)
	is.True(result.Contributors == nil)
}

func TestDecide_BypassAllowlist(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 10}}
	hostAccess := []UserHostAccess{{UserID: 10, BypassAllowlist: true}}
	svc := newRestrictedService(entries, hostAccess)
	result := svc.Decide(context.Background(), "1.2.3.4", "anything.example.com")
	is.True(result.Allowed)
	is.True(result.DenyReason == nil)
	is.Equal(len(result.Contributors), 1)
}

func TestDecide_HostAllowed(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 10}}
	hostAccess := []UserHostAccess{{UserID: 10, BypassAllowlist: false, AllowedHosts: []string{"allowed.example.com"}}}
	svc := newRestrictedService(entries, hostAccess)
	result := svc.Decide(context.Background(), "1.2.3.4", "allowed.example.com")
	is.True(result.Allowed)
	is.True(result.DenyReason == nil)
}

func TestDecide_HostNotAllowed(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 10}}
	hostAccess := []UserHostAccess{{UserID: 10, BypassAllowlist: false, AllowedHosts: []string{"allowed.example.com"}}}
	svc := newRestrictedService(entries, hostAccess)
	result := svc.Decide(context.Background(), "1.2.3.4", "denied.example.com")
	is.True(!result.Allowed)
	is.Equal(*result.DenyReason, DenyReasonHostNotAllowed)
	is.Equal(len(result.Contributors), 1)
}

func TestDecide_HostCaseFolding(t *testing.T) {
	is := is.New(t)
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 10}}
	hostAccess := []UserHostAccess{{UserID: 10, BypassAllowlist: false, AllowedHosts: []string{"allowed.example.com"}}}
	svc := newRestrictedService(entries, hostAccess)
	result := svc.Decide(context.Background(), "1.2.3.4", "ALLOWED.EXAMPLE.COM")
	is.True(result.Allowed)
}

func TestDecide_UnconfiguredUser(t *testing.T) {
	// UserID 99 has a device at 1.2.3.4 but no UserHostAccess entry.
	// Zero-value UserHostAccess{BypassAllowlist: false} → deny all hosts.
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1), UserID: ids.UserID(99)},
	}
	svc := newRestrictedService(entries, nil)
	result := svc.Decide(context.Background(), "1.2.3.4", "example.com")
	is.True(!result.Allowed)
	is.Equal(*result.DenyReason, DenyReasonHostNotAllowed)
}

func TestDecide_HostIntersection_DenyWins(t *testing.T) {
	// Two users share the same IP; intersection of their host sets applies.
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1), UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(2), AddressID: ids.AddressID(2), UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com", "c.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)

	// Only "b.com" survives the intersection.
	is.True(svc.Decide(context.Background(), "1.2.3.4", "b.com").Allowed)
	is.Equal(*svc.Decide(context.Background(), "1.2.3.4", "a.com").DenyReason, DenyReasonHostNotAllowed)
	is.Equal(*svc.Decide(context.Background(), "1.2.3.4", "c.com").DenyReason, DenyReasonHostNotAllowed)
}

func TestDecide_BypassAndNonBypass_SharedIP(t *testing.T) {
	// Bypass user is intersection-neutral: result follows the non-bypass user's grants.
	is := is.New(t)
	entries := []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1), UserID: ids.UserID(1)},
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(2), AddressID: ids.AddressID(2), UserID: ids.UserID(2)},
	}
	hostAccess := []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: true},
		{UserID: ids.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"allowed.com"}},
	}
	svc := newRestrictedService(entries, hostAccess)

	is.True(svc.Decide(context.Background(), "1.2.3.4", "allowed.com").Allowed)
	is.Equal(*svc.Decide(context.Background(), "1.2.3.4", "other.com").DenyReason, DenyReasonHostNotAllowed)
}

func TestDecide_DoesNotNotifyObservers(t *testing.T) {
	is := is.New(t)
	obs := &fakeObserver{}
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 1}}
	hostAccess := []UserHostAccess{{UserID: 1, BypassAllowlist: true}}
	svc := newRestrictedService(entries, hostAccess)
	svc.AddDecisionObserver(obs)

	_ = svc.Decide(context.Background(), "1.2.3.4", "example.com")
	_ = svc.Decide(context.Background(), "9.9.9.9", "example.com")

	is.Equal(len(obs.received()), 0)
}

// ── VerifyAccess ──────────────────────────────────────────────────────────────

func TestVerifyAccess_Success(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "1.2.3.4"})
	is.NoErr(err)
}

func TestVerifyAccess_InvalidToken(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "wrong", ClientIP: "1.2.3.4"})
	is.True(errors.Is(err, ErrInvalidBearerToken))
}

func TestVerifyAccess_IPNotEnabled(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "9.9.9.9"})
	is.True(errors.Is(err, ErrIPNotEnabled))
}

func TestVerifyAccess_HostDenied_EmitsEvent(t *testing.T) {
	is := is.New(t)
	svc := newHostRestrictedSvc(t, ids.UserID(1), "1.2.3.4", []string{"example.com"})

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	host := "other.com"
	err := svc.VerifyAccess(context.Background(), &VerifyRequest{
		Token:      "mysecret",
		ClientIP:   "1.2.3.4",
		TargetHost: &host,
	})
	is.True(errors.Is(err, ErrHostNotAllowed))

	events := obs.received()
	is.Equal(len(events), 1)
	is.True(!events[0].Outcome)
	is.Equal(*events[0].DenyReason, DenyReasonHostNotAllowed)
	is.Equal(len(events[0].IPContributors), 1)
}

func TestVerifyAccess_EmitsAllowEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(10), AddressID: ids.AddressID(20)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "1.2.3.4", TargetHost: new("example.com")})
	is.NoErr(err)

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(e.Outcome)
	is.True(e.DenyReason == nil)
	is.Equal(e.ClientIP, "1.2.3.4")
	is.Equal(len(e.IPContributors), 1)
	is.Equal(int64(e.IPContributors[0].DeviceID), int64(10))
	is.Equal(int64(e.IPContributors[0].AddressID), int64(20))
	is.True(!e.CreatedAt.IsZero())
}

func TestVerifyAccess_EmitsInvalidTokenEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "wrongtoken", ClientIP: "1.2.3.4"})
	is.True(errors.Is(err, ErrInvalidBearerToken))

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(!e.Outcome)
	is.Equal(*e.DenyReason, DenyReasonInvalidToken)
	is.Equal(len(e.IPContributors), 0)
}

func TestVerifyAccess_EmitsIPNotRegisteredEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "9.9.9.9"})
	is.True(errors.Is(err, ErrIPNotEnabled))

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(!e.Outcome)
	is.Equal(*e.DenyReason, DenyReasonIPNotRegistered)
	is.Equal(len(e.IPContributors), 0)
}

func TestVerifyAccess_AttachesGeoIP(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "8.8.8.8", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	resolver := stubResolver{result: geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 15169, ASNOrg: "Google LLC"}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, resolver, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "8.8.8.8"})
	is.NoErr(err)

	events := obs.received()
	is.Equal(len(events), 1)
	is.Equal(events[0].GeoIP.CountryCode, "US")
	is.Equal(events[0].GeoIP.ContinentCode, "NA")
	is.Equal(events[0].GeoIP.ASN, uint(15169))
}

func TestVerifyAccess_NilResolver(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "8.8.8.8", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	// Must not panic; GeoIP must be zero value.
	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "8.8.8.8"})
	is.NoErr(err)

	events := obs.received()
	is.Equal(len(events), 1)
	is.True(events[0].GeoIP.IsEmpty())
}

func TestVerifyAccess_GeoIPOnDeny(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{}}
	resolver := stubResolver{result: geoip.Result{CountryCode: "DE", ContinentCode: "EU"}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, resolver, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	// IP not in set → denied, but GeoIP must still be attached.
	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "9.9.9.9"})
	is.True(errors.Is(err, ErrIPNotEnabled))

	events := obs.received()
	is.Equal(len(events), 1)
	is.Equal(events[0].GeoIP.CountryCode, "DE")
	is.True(!events[0].Outcome)
}
