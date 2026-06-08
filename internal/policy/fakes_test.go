//go:build test

package policy

import (
	"context"
	"log/slog"
	"net/netip"
	"sync"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
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

// stubResolver returns a fixed geoip.Result for any IP.
type stubResolver struct {
	result geoip.Result
}

func (s stubResolver) Resolve(string) geoip.Result { return s.result }

// fixedHostProvider returns a fixed list of UserHostAccess entries.
type fixedHostProvider struct {
	entries []UserHostAccess
}

func (f *fixedHostProvider) GetAllUserHostAccess(_ context.Context) ([]UserHostAccess, error) {
	return f.entries, nil
}

// restrictedHostProvider grants a fixed set of hosts to every user.
type restrictedHostProvider struct {
	users []UserHostAccess
}

var _ HostAccessProvider = (*restrictedHostProvider)(nil)

func (p *restrictedHostProvider) GetAllUserHostAccess(_ context.Context) ([]UserHostAccess, error) {
	return p.users, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// mustAddr parses s into a canonical (unmapped) netip.Addr for tests, mirroring
// how the engine keys its IP set. Panics on invalid input.
func mustAddr(s string) netip.Addr {
	return netip.MustParseAddr(s).Unmap()
}

// newHostRestrictedSvc builds a Service where userID owns the given IP and
// is restricted to allowedHosts (empty slice = no hosts granted).
func newHostRestrictedSvc(t *testing.T, userID ids.UserID, ip string, allowedHosts []string) *Service {
	t.Helper()
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: ip, DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1), UserID: userID},
	}}
	hostProvider := &fixedHostProvider{entries: []UserHostAccess{
		{UserID: userID, BypassAllowlist: false, AllowedHosts: allowedHosts},
	}}
	svc, err := NewService(provider, hostProvider, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	return svc
}

// errHostProvider returns a fixed error from GetAllUserHostAccess.
type errHostProvider struct{ err error }

func (e *errHostProvider) GetAllUserHostAccess(_ context.Context) ([]UserHostAccess, error) {
	return nil, e.err
}

// mockNetworkPoliciesProvider returns a fixed list of CacheEntry rows.
type mockNetworkPoliciesProvider struct {
	entries []networkpolicies.CacheEntry
	err     error
}

var _ NetworkPoliciesProvider = (*mockNetworkPoliciesProvider)(nil)

func (m *mockNetworkPoliciesProvider) GetEnabledCacheEntries(_ context.Context) ([]networkpolicies.CacheEntry, error) {
	return m.entries, m.err
}

// newServiceWithNetworkPolicies builds a Service pre-populated with the given IP entries,
// host access grants, and network policy CIDR entries. hostAccess may be nil (deny-all for
// every user). The service is fully initialized before returning.
func newServiceWithNetworkPolicies(
	ipEntries []device.IPEntry,
	hostAccess []UserHostAccess,
	networkPolicyCacheEntries []networkpolicies.CacheEntry,
) *Service {
	provider := &mockProvider{entries: ipEntries}
	hostProv := &restrictedHostProvider{users: hostAccess}
	netProv := &mockNetworkPoliciesProvider{entries: networkPolicyCacheEntries}
	svc, err := NewService(provider, hostProv, &geoip.Lookup{}, netProv, "secret", noopLogger(), netip.Addr{})
	if err != nil {
		panic(err)
	}
	if err := svc.Initialize(context.Background()); err != nil {
		panic(err)
	}
	return svc
}

func newRestrictedService(entries []device.IPEntry, hostAccess []UserHostAccess) *Service {
	provider := &mockProvider{entries: entries}
	hostProv := &restrictedHostProvider{users: hostAccess}
	svc, err := NewService(provider, hostProv, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	if err != nil {
		panic(err)
	}
	if err := svc.Initialize(context.Background()); err != nil {
		panic(err)
	}
	return svc
}
