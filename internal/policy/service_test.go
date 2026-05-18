//go:build test

package policy

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/matryer/is"
)

func TestService_Initialize_PopulatesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
		{IP: "10.0.0.1", DeviceID: device.DeviceID(2), AddressID: device.AddressID(2)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
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
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
}

func TestService_AddDecisionObserver_NilIgnored(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	// Adding nil must not panic.
	svc.AddDecisionObserver(nil)
	is.Equal(len(svc.observers), 0)
}

func TestService_NotifyDecisionObservers_AllowEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(10), AddressID: device.AddressID(20)},
	}}
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
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
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
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
	svc, err := NewService(provider, &bypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", noopLogger(), netip.Addr{})
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
