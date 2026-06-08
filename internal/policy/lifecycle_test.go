//go:build test

package policy

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

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
