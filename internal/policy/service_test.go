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

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

// fakeObserver records every DecisionEvent it receives.
type fakeObserver struct {
	mu     sync.Mutex
	events []DecisionEvent
}

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

func (m *mockProvider) GetEnabledIPEntries(_ context.Context) ([]device.IPEntry, error) {
	return m.entries, m.err
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
	svc, err := NewService(provider, "secret", noopLogger(), netip.Addr{})
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
	svc, err := NewService(provider, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
}

func TestService_OnAddressEvent_RefreshesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "192.168.1.1", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, "secret", noopLogger(), netip.Addr{})
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
	svc, err := NewService(provider, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "1.2.3.4"}), ErrIPNotEnabled))
}

func TestService_LookupIP_RejectsTrustedProxyIP(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "127.0.0.1", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, "secret", noopLogger(), netip.MustParseAddr("127.0.0.1"))
	is.NoErr(err)

	is.NoErr(svc.Initialize(context.Background()))
	is.True(errors.Is(svc.VerifyAccess(context.Background(), &VerifyRequest{Token: "secret", ClientIP: "127.0.0.1"}), ErrIPNotEnabled))
}

func TestService_NotifyDecisionObservers_AllowEvent(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(10), AddressID: device.AddressID(20)},
	}}
	svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))

	obs := &fakeObserver{}
	svc.AddDecisionObserver(obs)

	targetHost := "example.com"
	req := &VerifyRequest{
		Token:      "mysecret",
		ClientIP:   "1.2.3.4",
		TargetHost: &targetHost,
	}
	err = svc.VerifyAccess(context.Background(), req)
	is.NoErr(err)

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(e.Outcome)
	is.True(e.DenyReason == nil)
	is.Equal(e.ClientIP, "1.2.3.4")
	is.True(e.DeviceID != nil)
	is.Equal(int64(*e.DeviceID), int64(10))
	is.True(e.AddressID != nil)
	is.Equal(int64(*e.AddressID), int64(20))
	is.True(!e.CreatedAt.IsZero())
}

func TestService_NotifyDecisionObservers_DenyInvalidToken(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
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
	is.True(e.DeviceID == nil)
	is.True(e.AddressID == nil)
}

func TestService_NotifyDecisionObservers_DenyIPNotRegistered(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
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
	is.True(e.DeviceID == nil)
	is.True(e.AddressID == nil)
}

func TestService_AddDecisionObserver_NilIgnored(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{entries: []device.IPEntry{
		{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
	}}
	svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	// Adding nil must not panic.
	svc.AddDecisionObserver(nil)
	is.Equal(len(svc.observers), 0)
}

func TestService_VerifyAccess(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{entries: []device.IPEntry{
			{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
		}}
		svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
		is.NoErr(err)
		is.NoErr(svc.Initialize(context.Background()))

		req := &VerifyRequest{
			Token:    "mysecret",
			ClientIP: "1.2.3.4",
		}
		err = svc.VerifyAccess(context.Background(), req)
		is.NoErr(err)
	})

	t.Run("missing secret", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{}
		_, err := NewService(provider, "", noopLogger(), netip.Addr{})
		is.True(errors.Is(err, ErrSecretNotConfigured))
	})

	t.Run("invalid token", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{entries: []device.IPEntry{
			{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
		}}
		svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
		is.NoErr(err)
		is.NoErr(svc.Initialize(context.Background()))

		req := &VerifyRequest{
			Token:    "wrong",
			ClientIP: "1.2.3.4",
		}
		err = svc.VerifyAccess(context.Background(), req)
		is.True(errors.Is(err, ErrInvalidBearerToken))
	})

	t.Run("ip not enabled", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{entries: []device.IPEntry{
			{IP: "1.2.3.4", DeviceID: device.DeviceID(1), AddressID: device.AddressID(1)},
		}}
		svc, err := NewService(provider, "mysecret", noopLogger(), netip.Addr{})
		is.NoErr(err)
		is.NoErr(svc.Initialize(context.Background()))

		req := &VerifyRequest{
			Token:    "mysecret",
			ClientIP: "9.9.9.9",
		}
		err = svc.VerifyAccess(context.Background(), req)
		is.True(errors.Is(err, ErrIPNotEnabled))
	})
}
