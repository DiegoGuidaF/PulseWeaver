//go:build test

package rule

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
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

func (f *fakeRepository) EnableDeviceAddressLeaseRuleConfig(ctx context.Context, deviceID device.DeviceID, config DeviceAddressLeaseConfig) (*Rule, error) {
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

// GetDeviceAddressLeaseTTLSeconds

func TestService_GetDeviceAddressLeaseTTLSeconds_NoRule_ReturnsNilTTL(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{getRuleErr: ErrRuleNotFound}
	svc := newTestService(repo)
	ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(context.Background(), device.DeviceID(1))
	is.NoErr(err)
	is.True(ttl == nil)
}

func TestService_GetDeviceAddressLeaseTTLSeconds_DisabledRule_ReturnsNilTTL(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		getRuleResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: false, Config: json.RawMessage(`{"ttl_seconds":300}`),
		},
	}
	svc := newTestService(repo)
	ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(context.Background(), device.DeviceID(1))
	is.NoErr(err)
	is.True(ttl == nil)
}

func TestService_GetDeviceAddressLeaseTTLSeconds_EnabledValidRule_ReturnsTTL(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		getRuleResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: true, Config: json.RawMessage(`{"ttl_seconds":300}`),
		},
	}
	svc := newTestService(repo)
	ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(context.Background(), device.DeviceID(1))
	is.NoErr(err)
	is.True(ttl != nil)
	is.Equal(*ttl, 300)
}

func TestService_GetDeviceAddressLeaseTTLSeconds_InvalidConfig_ReturnsErr(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		getRuleResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: true, Config: json.RawMessage(`{"ttl_seconds":-1}`),
		},
	}
	svc := newTestService(repo)
	ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(context.Background(), device.DeviceID(1))
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidRuleConfig))
	is.True(ttl == nil)
}

func TestService_GetDeviceAddressLeaseTTLSeconds_RepoError_Propagated(t *testing.T) {
	is := is.New(t)
	repoErr := errors.New("db error")
	repo := &fakeRepository{getRuleErr: repoErr}
	svc := newTestService(repo)
	ttl, err := svc.GetDeviceAddressLeaseTTLSeconds(context.Background(), device.DeviceID(1))
	is.True(err != nil)
	is.Equal(err, repoErr)
	is.True(ttl == nil)
}

// GetDeviceAddressLeaseRule

func TestService_GetDeviceAddressLeaseRule_ReturnsRule(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		getRuleResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: true, Config: json.RawMessage(`{"ttl_seconds":120}`),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		},
	}
	svc := newTestService(repo)
	out, err := svc.GetDeviceAddressLeaseRule(context.Background(), device.DeviceID(1))
	is.NoErr(err)
	is.True(out != nil)
	is.Equal(out.Config.TTLSeconds, 120)
}

func TestService_GetDeviceAddressLeaseRule_NotFound(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{getRuleErr: ErrRuleNotFound}
	svc := newTestService(repo)
	out, err := svc.GetDeviceAddressLeaseRule(context.Background(), device.DeviceID(1))
	is.True(err != nil)
	is.Equal(err, ErrRuleNotFound)
	is.True(out == nil)
}

func TestService_GetDeviceAddressLeaseRule_InvalidConfig_ReturnsErr(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		getRuleResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: true, Config: json.RawMessage(`{"ttl_seconds":-1}`),
		},
	}
	svc := newTestService(repo)
	out, err := svc.GetDeviceAddressLeaseRule(context.Background(), device.DeviceID(1))
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidRuleConfig))
	is.True(out == nil)
}

// EnableDeviceAddressLeaseRule

func TestService_EnableDeviceAddressLeaseRule_ValidTTL_ReturnsRule(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		enableResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: true, Config: json.RawMessage(`{"ttl_seconds":300}`),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		},
	}
	svc := newTestService(repo)
	out, err := svc.EnableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1), 300)
	is.NoErr(err)
	is.True(out != nil)
	is.Equal(out.DeviceID, device.DeviceID(1))
	is.Equal(out.Config.TTLSeconds, 300)
}

func TestService_EnableDeviceAddressLeaseRule_NegativeTTL_ReturnsErr(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{}
	svc := newTestService(repo)
	out, err := svc.EnableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1), -1)
	is.True(err != nil)
	is.True(errors.Is(err, ErrInvalidRuleConfig))
	is.True(out == nil)
}

func TestService_EnableDeviceAddressLeaseRule_DeviceNotFound_Propagated(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{enableErr: device.ErrDeviceNotFound}
	svc := newTestService(repo)
	out, err := svc.EnableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1), 60)
	is.True(err != nil)
	is.Equal(err, device.ErrDeviceNotFound)
	is.True(out == nil)
}

func TestService_EnableDeviceAddressLeaseRule_RepoError_Propagated(t *testing.T) {
	is := is.New(t)
	repoErr := errors.New("db error")
	repo := &fakeRepository{enableErr: repoErr}
	svc := newTestService(repo)
	out, err := svc.EnableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1), 60)
	is.True(err != nil)
	is.Equal(err, repoErr)
	is.True(out == nil)
}

// DisableDeviceAddressLeaseRule

func TestService_DisableDeviceAddressLeaseRule_ReturnsDisabledRule(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{
		disableResult: &Rule{
			ID: 1, DeviceID: 1, RuleType: RuleTypeDeviceAddressLease,
			Enabled: false, Config: json.RawMessage(`{"ttl_seconds":100}`),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		},
	}
	svc := newTestService(repo)
	out, err := svc.DisableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1))
	is.NoErr(err)
	is.True(out != nil)
	is.True(!out.Enabled)
}

func TestService_DisableDeviceAddressLeaseRule_NotFound(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepository{disableErr: ErrRuleNotFound}
	svc := newTestService(repo)
	out, err := svc.DisableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1))
	is.True(err != nil)
	is.Equal(err, ErrRuleNotFound)
	is.True(out == nil)
}

func TestService_DisableDeviceAddressLeaseRule_RepoError_Propagated(t *testing.T) {
	is := is.New(t)
	repoErr := errors.New("db error")
	repo := &fakeRepository{disableErr: repoErr}
	svc := newTestService(repo)
	out, err := svc.DisableDeviceAddressLeaseRule(context.Background(), device.DeviceID(1))
	is.True(err != nil)
	is.Equal(err, repoErr)
	is.True(out == nil)
}

// Observer events

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
