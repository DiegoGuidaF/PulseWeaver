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

func TestService_Initialize_PopulatesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
		{IP: "10.0.0.1", DeviceID: ids.DeviceID(2), AddressID: ids.AddressID(2)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.NoErr(err)
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("192.168.1.1")}))
	is.NoErr(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("10.0.0.1")}))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: mustAddr("1.2.3.4")}), ErrIPNotEnabled))
}

func TestService_Initialize_PropagatesError(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{err: errors.New("db error")}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
}

func TestService_AddDecisionObserver_NilIgnored(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	// Adding nil must not panic.
	svc.AddDecisionObserver(nil)
	is.Equal(len(svc.observers), 0)
}
