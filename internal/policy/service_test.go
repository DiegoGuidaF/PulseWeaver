//go:build test

package policy

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/matryer/is"
)

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
