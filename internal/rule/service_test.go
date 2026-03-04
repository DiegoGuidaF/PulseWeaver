package rule

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"github.com/matryer/is"
)

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
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
		ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.True(errors.Is(err, ErrInvalidRuleConfig))
		is.True(ttl == nil)
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		is := is.New(t)
		repoErr := errors.New("db error")
		repo := &fakeRepository{getRuleErr: repoErr}
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
		out, err := svc.GetDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(out != nil)
		is.Equal(out.Config.TTLSeconds, 120)
	})

	t.Run("not_found", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{getRuleErr: ErrRuleNotFound}
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), 300)
		is.NoErr(err)
		is.True(out != nil)
		is.Equal(out.DeviceID, device.DeviceID(1))
		is.Equal(out.Config.TTLSeconds, 300)
	})

	t.Run("negative_ttl_returns_err", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{}
		svc := NewService(repo, nil)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), -1)
		is.True(err != nil)
		is.True(errors.Is(err, ErrInvalidRuleConfig))
		is.True(out == nil)
	})

	t.Run("device_not_found_propagated", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{enableErr: device.ErrDeviceNotFound}
		svc := NewService(repo, nil)
		out, err := svc.EnableDeviceAddressLeaseRule(ctx, device.DeviceID(1), 60)
		is.True(err != nil)
		is.Equal(err, device.ErrDeviceNotFound)
		is.True(out == nil)
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		is := is.New(t)
		repoErr := errors.New("db error")
		repo := &fakeRepository{enableErr: repoErr}
		svc := NewService(repo, nil)
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
		svc := NewService(repo, nil)
		out, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.NoErr(err)
		is.True(out != nil)
		is.True(!out.Enabled)
	})

	t.Run("not_found", func(t *testing.T) {
		is := is.New(t)
		repo := &fakeRepository{disableErr: ErrRuleNotFound}
		svc := NewService(repo, nil)
		out, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.Equal(err, ErrRuleNotFound)
		is.True(out == nil)
	})

	t.Run("repo_error_propagated", func(t *testing.T) {
		is := is.New(t)
		repoErr := errors.New("db error")
		repo := &fakeRepository{disableErr: repoErr}
		svc := NewService(repo, nil)
		out, err := svc.DisableDeviceAddressLeaseRule(ctx, device.DeviceID(1))
		is.True(err != nil)
		is.Equal(err, repoErr)
		is.True(out == nil)
	})
}
