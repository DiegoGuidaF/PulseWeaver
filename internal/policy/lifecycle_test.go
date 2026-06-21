//go:build test

package policy

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/matryer/is"
)

// TestService_ConcurrentDecideDuringRebuild exercises the copy-on-write read
// path under contention: many concurrent Decide calls (both an exact-IP hit and
// a CIDR-fallback miss) while the cache is repeatedly rebuilt from scratch.
// Its value is under the race detector:
//
//	go test -race -tags=test ./internal/policy/...
//
// A regression that mutates a published snapshot in place, or reads it without
// the brief RLock, would trip -race here.
func TestService_ConcurrentDecideDuringRebuild(t *testing.T) {
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	netProv := &mockNetworkPoliciesProvider{entries: []networkpolicies.CacheEntry{
		{PolicyID: ids.NetworkPolicyID(1), PolicyName: "p", CIDR: "10.0.0.0/8", BypassHostCheck: true},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, netProv, "secret", noopLogger(), netip.Addr{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := svc.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	stop := make(chan struct{})

	for range 8 {
		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					svc.Decide(ctx, mustAddr("1.2.3.4"), "example.com")  // device hit
					svc.Decide(ctx, mustAddr("10.1.2.3"), "example.com") // CIDR fallback
				}
			}
		})
	}

	wg.Go(func() {
		for range 500 {
			_ = svc.refreshCache(ctx)
		}
		close(stop)
	})

	wg.Wait()
}

func TestService_OnAddressEvent_RefreshesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	is.NoErr(svc.Initialize(context.Background()))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("192.168.1.1")}))

	// Update provider to return different IPs
	provider.entries = []device.IPEntry{
		{IP: "10.0.0.2", DeviceID: ids.DeviceID(3), AddressID: ids.AddressID(3)},
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

	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("192.168.1.1")}), ErrIPNotEnabled))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("10.0.0.2")}))
}

// TestService_PeriodicReconcile_RebuildsWithoutEvent proves the staleness
// backstop: with no change event fired, the periodic ticker alone must pick up
// provider changes — the scenario a dropped or failed event would otherwise
// leave stale (and, for a revoked grant, stale-allow).
func TestService_PeriodicReconcile_RebuildsWithoutEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))
	svc.reconcileInterval = 20 * time.Millisecond

	// Change the provider's data but deliberately fire NO change event, so only
	// the periodic reconcile can pick it up.
	provider.entries = []device.IPEntry{
		{IP: "10.0.0.2", DeviceID: ids.DeviceID(3), AddressID: ids.AddressID(3)},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.RunListener(ctx)
	}()

	time.Sleep(100 * time.Millisecond) // allow several reconcile ticks
	cancel()
	<-done

	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("192.168.1.1")}), ErrIPNotEnabled))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("10.0.0.2")}))
}

func TestService_OnHostAccessChanged_RefreshesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1), UserID: ids.UserID(1)},
	}}
	hostProvider := &fixedHostProvider{entries: []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"a.com"}},
	}}
	svc, err := NewService(provider, hostProvider, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	aHost := "a.com"
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: mustAddr("1.2.3.4"), TargetHost: &aHost}))

	// Change allowed hosts from a.com → b.com
	hostProvider.entries = []UserHostAccess{
		{UserID: ids.UserID(1), BypassAllowlist: false, AllowedHosts: []string{"b.com"}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = svc.RunListener(ctx)
	}()

	svc.OnHostAccessChanged(context.Background())
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	// a.com should now be denied, b.com allowed
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: mustAddr("1.2.3.4"), TargetHost: &aHost}), ErrHostNotAllowed))
	bHost := "b.com"
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "mysecret", ClientIP: mustAddr("1.2.3.4"), TargetHost: &bHost}))
}
