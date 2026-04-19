//go:build test

package policy

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/matryer/is"
)

// fakeObserver records every DecisionEvent it receives.
type fakeObserver struct {
	mu     sync.Mutex
	events []DecisionEvent
}

var _ DecisionObserver = (*fakeObserver)(nil)

func (f *fakeObserver) OnDecision(_ context.Context, e DecisionEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func (f *fakeObserver) received() []DecisionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]DecisionEvent, len(f.events))
	copy(out, f.events)
	return out
}

type mockProvider struct {
	entries []device.IPEntry
	err     error
}

var _ EnabledIPsProvider = (*mockProvider)(nil)

func (m *mockProvider) GetEnabledIPEntries(_ context.Context) ([]device.IPEntry, error) {
	return m.entries, m.err
}

// bypassAllHostProvider implements HostAccessProvider and grants bypass to every UserID seen.
type bypassAllHostProvider struct{}

var _ HostAccessProvider = (*bypassAllHostProvider)(nil)

func (b *bypassAllHostProvider) GetAllUserHostAccess(_ context.Context) ([]UserHostAccess, error) {
	return []UserHostAccess{{UserID: 0, BypassAllowlist: true}}, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestService_Initialize_PopulatesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
		{IP: "10.0.0.1", DeviceID: device.DeviceID(2), AddressID: device.AddressID(2)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.NoErr(err)
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "192.168.1.1"}))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "10.0.0.1"}))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "1.2.3.4"}), ErrIPNotEnabled))
}

func TestService_Initialize_PropagatesError(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{err: errors.New("db error")}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
}

func TestService_OnAddressEvent_RefreshesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	is.NoErr(svc.Initialize(context.Background()))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "192.168.1.1"}))

	// Update provider to return different IPs
	provider.entries = []device.IPEntry{
		{IP: "10.0.0.2", DeviceID: device.DeviceID(3), AddressID: device.AddressID(3)},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.RunListener(ctx)
	}()

	// Send event and wait for refresh
	svc.OnAddressEvent(context.Background(), device.AddressEvent{})
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "192.168.1.1"}), ErrIPNotEnabled))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "10.0.0.2"}))
}

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

func TestService_NotifyDecisionObservers_AllowEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(10), AddressID: device.AddressID(20)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	req := &VerifyRequest{
		Token:      "mysecret",
		ClientIP:   "1.2.3.4",
		TargetHost: new("example.com"),
	}
	err = svc.VerifyAccess(context.Background(), req)
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

func TestService_NotifyDecisionObservers_DenyInvalidToken(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	req := &VerifyRequest{
		Token:    "wrongtoken",
		ClientIP: "1.2.3.4",
	}
	err = svc.VerifyAccess(context.Background(), req)
	is.True(errors.Is(err, ErrInvalidBearerToken))

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(!e.Outcome)
	is.True(e.DenyReason != nil)
	is.Equal(*e.DenyReason, DenyReasonInvalidToken)
	is.Equal(len(e.IPContributors), 0)
}

func TestService_NotifyDecisionObservers_DenyIPNotRegistered(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	req := &VerifyRequest{
		Token:    "mysecret",
		ClientIP: "9.9.9.9",
	}
	err = svc.VerifyAccess(context.Background(), req)
	is.True(errors.Is(err, ErrIPNotEnabled))

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(!e.Outcome)
	is.True(e.DenyReason != nil)
	is.Equal(*e.DenyReason, DenyReasonIPNotRegistered)
	is.Equal(len(e.IPContributors), 0)
}

func TestService_AddDecisionObserver_NilIgnored(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	// Adding nil must not panic.
	svc.AddDecisionObserver(nil)
	is.Equal(len(svc.observers), 0)
}

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

// GeoIP integration tests

type stubResolver struct {
	result geoip.Result
}

func (s stubResolver) Resolve(string) geoip.Result { return s.result }

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

// ── Host allowlist tests ──────────────────────────────────────────────────────

// fixedHostProvider returns a fixed list of UserHostAccess entries.
type fixedHostProvider struct {
	entries []UserHostAccess
}

func (f *fixedHostProvider) GetAllUserHostAccess(_ context.Context) ([]UserHostAccess, error) {
	return f.entries, nil
}

// newHostRestrictedSvc builds a Service where user userID owns the given IP and
// is restricted to allowedHosts (empty slice = no hosts granted).
func newHostRestrictedSvc(t *testing.T, userID auth.UserID, ip string, allowedHosts []string) *Service {
	t.Helper()
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: ip, DeviceID: device.DeviceID(1), AddressID: device.AddressID(1), UserID: userID},
	}}
	hostProvider := &fixedHostProvider{entries: []UserHostAccess{
		{UserID: userID, BypassAllowlist: false, AllowedHosts: allowedHosts},
	}}
	svc, err := NewService(provider, hostProvider, &geoip.Lookup{}, "mysecret", noopLogger(), netip.Addr{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return svc
}

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
