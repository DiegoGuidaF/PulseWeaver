//go:build test

package policy_test

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/policy"
	"github.com/matryer/is"
)

type mockProvider struct {
	ips []string
	err error
}

func (m *mockProvider) GetEnabledUniqueIPs(_ context.Context) ([]string, error) {
	return m.ips, m.err
}

func noopLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestService_Initialize_PopulatesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{ips: []string{"192.168.1.1", "10.0.0.1"}}
	svc, err := policy.NewService(provider, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.NoErr(err)
	is.True(svc.ContainsIP("192.168.1.1"))
	is.True(svc.ContainsIP("10.0.0.1"))
	is.True(!svc.ContainsIP("1.2.3.4"))
}

func TestService_Initialize_PropagatesError(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{err: errors.New("db error")}
	svc, err := policy.NewService(provider, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	err = svc.Initialize(context.Background())
	is.True(err != nil)
}

func TestService_OnAddressEvent_RefreshesCache(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{ips: []string{"192.168.1.1"}}
	svc, err := policy.NewService(provider, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)

	is.NoErr(svc.Initialize(context.Background()))
	is.True(svc.ContainsIP("192.168.1.1"))

	// Update provider to return different IPs
	provider.ips = []string{"10.0.0.2"}

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

	is.True(!svc.ContainsIP("192.168.1.1"))
	is.True(svc.ContainsIP("10.0.0.2"))
}

func TestService_ContainsIP_Empty(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{ips: []string{}}
	svc, err := policy.NewService(provider, "secret", noopLogger(), netip.Addr{})
	is.NoErr(err)
	is.NoErr(svc.Initialize(context.Background()))
	is.True(!svc.ContainsIP("1.2.3.4"))
}

func TestService_ContainsIP_RejectsTrustedProxyIP(t *testing.T) {
	is := is.New(t)
	provider := &mockProvider{ips: []string{"127.0.0.1"}}
	svc, err := policy.NewService(provider, "secret", noopLogger(), netip.MustParseAddr("127.0.0.1"))
	is.NoErr(err)

	is.NoErr(svc.Initialize(context.Background()))
	is.True(!svc.ContainsIP("127.0.0.1"))
}

func TestService_VerifyAccess(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{ips: []string{"1.2.3.4"}}
		svc, err := policy.NewService(provider, "mysecret", noopLogger(), netip.Addr{})
		is.NoErr(err)
		is.NoErr(svc.Initialize(context.Background()))

		err = svc.VerifyAccess(context.Background(), "mysecret", "1.2.3.4")
		is.NoErr(err)
	})

	t.Run("missing secret", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{ips: []string{"1.2.3.4"}}
		_, err := policy.NewService(provider, "", noopLogger(), netip.Addr{})
		is.True(errors.Is(err, policy.ErrSecretNotConfigured))
	})

	t.Run("invalid token", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{ips: []string{"1.2.3.4"}}
		svc, err := policy.NewService(provider, "mysecret", noopLogger(), netip.Addr{})
		is.NoErr(err)
		is.NoErr(svc.Initialize(context.Background()))

		err = svc.VerifyAccess(context.Background(), "wrong", "1.2.3.4")
		is.True(errors.Is(err, policy.ErrInvalidBearerToken))
	})

	t.Run("ip not enabled", func(t *testing.T) {
		is := is.New(t)
		provider := &mockProvider{ips: []string{"1.2.3.4"}}
		svc, err := policy.NewService(provider, "mysecret", noopLogger(), netip.Addr{})
		is.NoErr(err)
		is.NoErr(svc.Initialize(context.Background()))

		err = svc.VerifyAccess(context.Background(), "mysecret", "9.9.9.9")
		is.True(errors.Is(err, policy.ErrIPNotEnabled))
	})
}
