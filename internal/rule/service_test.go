//go:build test

package rule

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/matryer/is"
)

func newTestService(repo repository) *Service {
	return NewService(repo, slog.New(slog.DiscardHandler))
}

// fakeRepository returns only pre-set values; no internal logic.
type fakeRepository struct {
	getRuleResult *Rule
	getRuleErr    error
	enableResult  *Rule
	enableErr     error
	disableResult *Rule
	disableErr    error
}

var _ repository = (*fakeRepository)(nil)

type mockRuleObserver struct {
	events []RuleEvent
}

func (m *mockRuleObserver) OnRuleEvent(_ context.Context, event RuleEvent) {
	m.events = append(m.events, event)
}

func (f *fakeRepository) GetRuleByDeviceAndType(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error) {
	if f.getRuleErr != nil {
		return nil, f.getRuleErr
	}
	return f.getRuleResult, nil
}

func (f *fakeRepository) EnableDeviceAddressLeaseRuleConfig(ctx context.Context, deviceID device.DeviceID, config *DeviceAddressLeaseConfig) (*Rule, error) {
	if f.enableErr != nil {
		return nil, f.enableErr
	}
	return f.enableResult, nil
}

func (f *fakeRepository) DisableRule(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error) {
	if f.disableErr != nil {
		return nil, f.disableErr
	}
	return f.disableResult, nil
}

func TestService_GetDeviceAddressLeaseTTLSeconds(t *testing.T) {
	ctx := context.Background()

	t.Run("no_rule_returns_nil_ttl", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{getRuleErr: ErrRuleNotFound}
		svc := newTestService(repo)
		ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(ttl == nil)
	})

	t.Run("disabled_rule_returns_nil_ttl", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			getRuleResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: false, Config: json.RawMessage(`{"ttl_seconds":300}`),
			},
		}
		svc := newTestService(repo)
		ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(ttl == nil)
	})

	t.Run("enabled_valid_rule_returns_ttl", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			getRuleResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: true, Config: json.RawMessage(`{"ttl_seconds":300}`),
			},
		}
		svc := newTestService(repo)
		ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(ttl != nil)
		is.Equal(*ttl, 300)
	})

	t.Run("invalid_config_returns_err", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			getRuleResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: true, Config: json.RawMessage(`{"ttl_seconds":-1}`),
			},
		}
		svc := newTestService(repo)
		ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.True(errors.Is(err, ErrInvalidRuleConfig))
		is.True(ttl == nil)
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		is := is.New(t)
		repoErr := errors.New("db error")
		repo := &fakeRepository{getRuleErr: repoErr}
		svc := newTestService(repo)
		ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.Equal(err, repoErr)
		is.True(ttl == nil)
	})
}

func TestService_GetDeviceAddressLeaseRule(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_rule", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			getRuleResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: true, Config: json.RawMessage(`{"ttl_seconds":120}`),
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			},
		}
		svc := newTestService(repo)
		out, err := svc.GetDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(out != nil)
		is.Equal(out.Config.TTLSeconds, 120)
	})

	t.Run("not_found", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{getRuleErr: ErrRuleNotFound}
		svc := newTestService(repo)
		out, err := svc.GetDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.Equal(err, ErrRuleNotFound)
		is.True(out == nil)
	})

	t.Run("invalid_config_returns_err", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			getRuleResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: true, Config: json.RawMessage(`{"ttl_seconds":-1}`),
			},
		}
		svc := newTestService(repo)
		out, err := svc.GetDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.True(errors.Is(err, ErrInvalidRuleConfig))
		is.True(out == nil)
	})
}

func TestService_EnableDeviceAddressLeaseRule(t *testing.T) {
	ctx := context.Background()

	t.Run("valid_ttl_returns_rule", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			enableResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: true, Config: json.RawMessage(`{"ttl_seconds":300}`),
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			},
		}
		svc := newTestService(repo)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), 300)
		is.NoErr(err)
		is.True(out != nil)
		is.Equal(out.DeviceID, device.DeviceID(1))
		is.Equal(out.Config.TTLSeconds, 300)
	})

	t.Run("negative_ttl_returns_err", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{}
		svc := newTestService(repo)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), -1)
		is.True(err != nil)
		is.True(errors.Is(err, ErrInvalidRuleConfig))
		is.True(out == nil)
	})

	t.Run("device_not_found_propagated", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{enableErr: device.ErrDeviceNotFound}
		svc := newTestService(repo)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), 60)
		is.True(err != nil)
		is.Equal(err, device.ErrDeviceNotFound)
		is.True(out == nil)
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		is := is.New(t)
		repoErr := errors.New("db error")
		repo := &fakeRepository{enableErr: repoErr}
		svc := newTestService(repo)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), 60)
		is.True(err != nil)
		is.Equal(err, repoErr)
		is.True(out == nil)
	})
}

func TestService_DisableDeviceAddressLeaseRule(t *testing.T) {
	ctx := context.Background()

	t.Run("returns_disabled_rule", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{
			disableResult: &Rule{
				ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
				Enabled: false, Config: json.RawMessage(`{"ttl_seconds":100}`),
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			},
		}
		svc := newTestService(repo)
		out, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(out != nil)
		is.True(!out.Enabled)
	})

	t.Run("not_found", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{disableErr: ErrRuleNotFound}
		svc := newTestService(repo)
		out, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.Equal(err, ErrRuleNotFound)
		is.True(out == nil)
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		is := is.New(t)
		repoErr := errors.New("db error")
		repo := &fakeRepository{disableErr: repoErr}
		svc := newTestService(repo)
		out, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.Equal(err, repoErr)
		is.True(out == nil)
	})
}

func TestService_EnableDeviceAddressLeaseRule_EmitsRuleEnabledEvent(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	repo := &fakeRepository{
		enableResult: &Rule{
			ID: 1, DeviceID: 55, RuleType: RuleTypeDeviceAddressLease,
			Enabled: true, Config: json.RawMessage(`{"ttl_seconds":300}`),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		},
	}
	svc := newTestService(repo)
	observer := &mockRuleObserver{}
	svc.AddRuleObserver(observer)

	_, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(55), 300)
	is.NoErr(err)
	is.Equal(len(observer.events), 1)
	is.Equal(observer.events[0].Type, RuleEventTypeEnabled)
	is.Equal(observer.events[0].RuleType, RuleTypeDeviceAddressLease)
	is.Equal(observer.events[0].DeviceID, device.DeviceID(55))
	is.True(observer.events[0].TTLSeconds != nil)
	is.Equal(*observer.events[0].TTLSeconds, 300)
}

func TestService_DisableDeviceAddressLeaseRule_EmitsRuleDisabledEvent(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	repo := &fakeRepository{
		disableResult: &Rule{
			ID: 1, DeviceID: 55, RuleType: RuleTypeDeviceAddressLease,
			Enabled: false, Config: json.RawMessage(`{"ttl_seconds":300}`),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		},
	}
	svc := newTestService(repo)
	observer := &mockRuleObserver{}
	svc.AddRuleObserver(observer)

	_, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(55))
	is.NoErr(err)
	is.Equal(len(observer.events), 1)
	is.Equal(observer.events[0].Type, RuleEventTypeDisabled)
	is.Equal(observer.events[0].RuleType, RuleTypeDeviceAddressLease)
	is.Equal(observer.events[0].DeviceID, device.DeviceID(55))
	is.True(observer.events[0].TTLSeconds == nil)
}
