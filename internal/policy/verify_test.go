//go:build test

package policy

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/matryer/is"
)

// ── lookupIP ─────────────────────────────────────────────────────────────────

func TestService_LookupIP_Empty(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "1.2.3.4"}), ErrIPNotEnabled))
}

func TestService_LookupIP_RejectsTrustedProxyIP(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "127.0.0.1", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "secret", noopLogger(), netip.MustParseAddr("127.0.0.1"))
	is.NoErr(err)

	is.NoErr(svc.Initialize(context.Background()))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "127.0.0.1"}), ErrIPNotEnabled))
}

// ── VerifyAccess ─────────────────────────────────────────────────────────────

func TestService_VerifyAccess_Success(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "1.2.3.4"})
	is.NoErr(err)
}

func TestService_VerifyAccess_InvalidToken(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "wrong", ClientIP: "1.2.3.4"})
	is.True(errors.Is(err, ErrInvalidBearerToken))
}

func TestService_VerifyAccess_IPNotEnabled(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "9.9.9.9"})
	is.True(errors.Is(err, ErrIPNotEnabled))
}

// ── GeoIP enrichment ─────────────────────────────────────────────────────────

func TestService_VerifyAccess_AttachesGeoIP(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "8.8.8.8", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	resolver := stubResolver{result: geoip.Result{CountryCode: "US", CountryName: "United States", ContinentCode: "NA", ASN: 15169, ASNOrg: "Google LLC"}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, resolver, "mysecret", noopLogger(), netip.Addr{})
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

func TestService_VerifyAccess_NilResolver(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "8.8.8.8", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	// Must not panic, GeoIP should be zero value.
	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "8.8.8.8"})
	is.NoErr(err)

	events := obs.received()
	is.Equal(len(events), 1)
	is.True(events[0].GeoIP.IsEmpty())
}

func TestService_VerifyAccess_GeoIPOnDeny(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{}}
	resolver := stubResolver{result: geoip.Result{CountryCode: "DE", ContinentCode: "EU"}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, resolver, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	// IP not in set → denied, but GeoIP should still be attached.
	err = svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: "9.9.9.9"})
	is.True(errors.Is(err, ErrIPNotEnabled))

	events := obs.received()
	is.Equal(len(events), 1)
	is.Equal(events[0].GeoIP.CountryCode, "DE")
	is.True(!events[0].Outcome)
}

// ── Host allowlist ────────────────────────────────────────────────────────────

func TestService_VerifyAccess_HostAllowed(t *testing.T) {
	is := is.New(t)
	svc := newHostRestrictedSvc(t, auth.UserID(1), "1.2.3.4", []string{"example.com"})

	host := "example.com"
	err := svc.VerifyAccess(context.Background(), &VerifyRequest{
		Token:      "mysecret",
		ClientIP:   "1.2.3.4",
		TargetHost: &host,
	})
	is.NoErr(err)
}

func TestService_VerifyAccess_HostDenied_WrongHost(t *testing.T) {
	is := is.New(t)
	svc := newHostRestrictedSvc(t, auth.UserID(1), "1.2.3.4", []string{"example.com"})

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
	is.True(events[0].DenyReason != nil)
	is.Equal(*events[0].DenyReason, DenyReasonHostNotAllowed)
	is.Equal(len(events[0].IPContributors), 1)
}

func TestService_VerifyAccess_HostDenied_UnconfiguredUser(t *testing.T) {
	// UserID 99 has a device at 1.2.3.4 but no UserHostAccess entry.
	// Policy treats them as deny-all (zero-value accumulator: bypassAll=true initially
	// but no non-bypass user contributes, so bypass stays true... wait, actually:
	// ua = accessByUser[99] → zero value UserHostAccess{BypassAllowlist: false}
	// acc.bypassAll = true AND false = false
	// no host merge (empty AllowedHosts)
	// → any host denied.
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1), UserID: auth.UserID(99)},
	}}
	// No UserHostAccess entry for user 99 → zero value applied.
	hostProvider := &fixedHostProvider{entries: []UserHostAccess{}}
	svc, err := NewService(provider, hostProvider, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	host := "example.com"
	err = svc.VerifyAccess(context.Background(), &VerifyRequest{
		Token:      "mysecret",
		ClientIP:   "1.2.3.4",
		TargetHost: &host,
	})
	is.True(errors.Is(err, ErrHostNotAllowed))
}

func TestService_VerifyAccess_HostIntersection_DenyWins(t *testing.T) {
	// Two users share same IP. Intersection of allowed hosts applies.
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1), UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: device.DeviceID(2), AddressID: device.AddressID(2), UserID: auth.UserID(2)},
	}}
	hostProvider := &fixedHostProvider{entries: []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com", "b.com"}},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"b.com", "c.com"}},
	}}
	svc, err := NewService(provider, hostProvider, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	req := func(host string) *VerifyRequest {
		return &VerifyRequest{Token: "mysecret", ClientIP: "1.2.3.4", TargetHost: &host}
	}

	// Only "b.com" is in both sets.
	is.NoErr(svc.VerifyAccess(context.Background(), req("b.com")))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), req("a.com")), ErrHostNotAllowed))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), req("c.com")), ErrHostNotAllowed))
}

func TestService_VerifyAccess_BypassAndNonBypass_SharedIP(t *testing.T) {
	// Bypass user is intersection-neutral: deny-wins means result follows non-bypass user's grants.
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1), UserID: auth.UserID(1)},
		{IP: "1.2.3.4", DeviceID: device.DeviceID(2), AddressID: device.AddressID(2), UserID: auth.UserID(2)},
	}}
	hostProvider := &fixedHostProvider{entries: []UserHostAccess{
		{UserID: auth.UserID(1), BypassAllowlist: true},
		{UserID: auth.UserID(2), BypassAllowlist: false, AllowedHosts: []string{"allowed.com"}},
	}}
	svc, err := NewService(provider, hostProvider, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	req := func(host string) *VerifyRequest {
		return &VerifyRequest{Token: "mysecret", ClientIP: "1.2.3.4", TargetHost: &host}
	}

	// Non-bypass user's allowlist wins: only "allowed.com" passes.
	is.NoErr(svc.VerifyAccess(context.Background(), req("allowed.com")))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), req("other.com")), ErrHostNotAllowed))
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
	is.True(result.DenyReason != nil)
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

func TestDecide_DoesNotNotifyObservers(t *testing.T) {
	is := is.New(t)
	obs := &fakeObserver{}
	entries := []device.IPEntry{{IP: "1.2.3.4", DeviceID: 1, AddressID: 1, UserID: 1}}
	hostAccess := []UserHostAccess{{UserID: 1, BypassAllowlist: true}}
	svc := newRestrictedService(entries, hostAccess)
	svc.AddDecisionObserver(obs)

	// Call Decide directly — no observers should fire.
	_ = svc.Decide(context.Background(), "1.2.3.4", "example.com")
	_ = svc.Decide(context.Background(), "9.9.9.9", "example.com")

	is.Equal(len(obs.received()), 0)
}

func TestDecide_TrustedProxyRejected(t *testing.T) {
	is := is.New(t)
	proxy := netip.MustParseAddr("10.0.0.1")
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "10.0.0.1", DeviceID: 1, AddressID: 1, UserID: 1},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "secret", noopLogger(), proxy)
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	result := svc.Decide(context.Background(), "10.0.0.1", "example.com")
	is.True(!result.Allowed)
	is.True(result.DenyReason != nil)
	is.Equal(*result.DenyReason, DenyReasonIPNotRegistered)
}
