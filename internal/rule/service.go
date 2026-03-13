package rule

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type repository interface {
	GetRuleByDeviceAndType(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error)
	EnableDeviceAddressLeaseRuleConfig(ctx context.Context, deviceID device.DeviceID, config *DeviceAddressLeaseConfig) (*Rule, error)
	DisableRule(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error)
}

type Service struct {
	repo      repository
	observers []RuleObserver
	logger    *slog.Logger
}

func NewService(repo repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "rule")),
	}
}

func (s *Service) AddRuleObserver(o RuleObserver) {
	if o == nil {
		return
	}
	s.observers = append(s.observers, o)
}

func (s *Service) notifyRuleObservers(ctx context.Context, event RuleEvent) {
	for _, o := range s.observers {
		o.OnRuleEvent(ctx, event)
	}
}

// GetDeviceAddressLeaseTTLSeconds returns the TTL in seconds to apply for address leases
// for the given device, or nil if no active rule exists.
func (s *Service) GetDeviceAddressLeaseTTLSeconds(ctx context.Context, deviceID device.DeviceID) (*int, error) {
	logger := s.logger.With(
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyRuleType, string(RuleTypeDeviceAddressLease)),
	)

	//TODO: Call other service method to retrieve rule, less duplicated code
	rule, err := s.repo.GetRuleByDeviceAndType(ctx, deviceID, RuleTypeDeviceAddressLease)
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			// No rule configured for this device.
			return nil, nil
		}
		logger.ErrorContext(ctx, "failed to load rule", slog.Any(AttrKeyError, err))
		return nil, err
	}

	if !rule.Enabled {
		return nil, nil
	}

	addressLeaseRule, err := rule.ToDeviceAddressLeaseRule()
	if err != nil {
		logger.ErrorContext(ctx, "invalid device lease rule config",
			slog.Any(AttrKeyError, err),
		)
		return nil, ErrInvalidRuleConfig
	}

	return &addressLeaseRule.Config.TTLSeconds, nil
}

// GetDeviceAddressLeaseRule returns the device lease rule for the device, or ErrRuleNotFound if none exists.
// If the rule exists but has invalid config, returns ErrInvalidRuleConfig.
func (s *Service) GetDeviceAddressLeaseRule(ctx context.Context, deviceID device.DeviceID) (*DeviceAddressLeaseRule, error) {
	rule, err := s.repo.GetRuleByDeviceAndType(ctx, deviceID, RuleTypeDeviceAddressLease)
	if err != nil {
		return nil, err
	}

	return rule.ToDeviceAddressLeaseRule()
}

// EnableDeviceAddressLeaseRule creates or updates the device lease rule for the device.
// ttlSeconds must be positive; enabled controls whether the rule is active.
func (s *Service) EnableDeviceAddressLeaseRule(
	ctx context.Context,
	deviceID device.DeviceID,
	ttlSeconds int,
) (*DeviceAddressLeaseRule, error) {
	logger := s.logger.With(
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyRuleType, string(RuleTypeDeviceAddressLease)),
	)

	config, err := NewDeviceAddressLeaseConfig(ttlSeconds)
	if err != nil {
		return nil, err
	}

	newRule, err := s.repo.EnableDeviceAddressLeaseRuleConfig(ctx, deviceID, config)
	if err != nil {
		return nil, err
	}
	logger.InfoContext(ctx, "enabled device address lease rule successfully", slog.Int64(AttrKeyRuleID, int64(newRule.ID)))

	s.notifyRuleObservers(ctx, RuleEvent{
		Type:       RuleEventTypeEnabled,
		DeviceID:   deviceID,
		RuleType:   RuleTypeDeviceAddressLease,
		TTLSeconds: new(config.TTLSeconds),
		OccurredAt: time.Now().UTC(),
	})

	return newRule.ToDeviceAddressLeaseRule()
}

// DisableDeviceAddressLeaseRule sets enabled to false for the device lease rule for the device.
// Returns the updated rule or ErrRuleNotFound if no rule exists.
func (s *Service) DisableDeviceAddressLeaseRule(ctx context.Context, deviceID device.DeviceID) (*DeviceAddressLeaseRule, error) {
	logger := s.logger.With(
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyRuleType, string(RuleTypeDeviceAddressLease)),
	)
	rule, err := s.repo.DisableRule(ctx, deviceID, RuleTypeDeviceAddressLease)
	if err != nil {
		return nil, err
	}
	logger.InfoContext(ctx, "disabled device address lease rule successfully", slog.Int64(AttrKeyRuleID, int64(rule.ID)))

	s.notifyRuleObservers(ctx, RuleEvent{
		Type:       RuleEventTypeDisabled,
		DeviceID:   deviceID,
		RuleType:   RuleTypeDeviceAddressLease,
		TTLSeconds: nil,
		OccurredAt: time.Now().UTC(),
	})

	return rule.ToDeviceAddressLeaseRule()
}
