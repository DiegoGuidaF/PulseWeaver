package rule

import (
	"context"
	"errors"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

type repository interface {
	GetRuleByDeviceAndType(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error)
	EnableDeviceAddressLeaseRuleConfig(ctx context.Context, deviceID device.DeviceID, config *DeviceAddressLeaseConfig) (*Rule, error)
	DisableRule(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error)
}

type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetDeviceAddressLeaseTTLSeconds returns the TTL in seconds to apply for address leases
// for the given device, or nil if no active rule exists.
func (s *Service) GetDeviceAddressLeaseTTLSeconds(ctx context.Context, deviceID device.DeviceID) (*int, error) {
	ctx, logger := logging.Enrich(ctx,
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyRuleType, string(RuleTypeDeviceAddressLease)),
	)

	logger.Debug("evaluating device lease rule")

	//TODO: Call other service method to retrieve rule, less duplicated code
	rule, err := s.repo.GetRuleByDeviceAndType(ctx, deviceID, RuleTypeDeviceAddressLease)
	if err != nil {
		if errors.Is(err, ErrRuleNotFound) {
			// No rule configured for this device.
			return nil, nil
		}
		logger.Error("failed to load rule", slog.Any(AttrKeyError, err))
		return nil, err
	}

	if !rule.Enabled {
		return nil, nil
	}

	addressLeaseRule, err := rule.ToDeviceAddressLeaseRule()
	if err != nil {
		logger.Error("invalid device lease rule config",
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
	config, err := NewDeviceAddressLeaseConfig(ttlSeconds)
	if err != nil {
		return nil, err
	}

	newRule, err := s.repo.EnableDeviceAddressLeaseRuleConfig(ctx, deviceID, config)
	if err != nil {
		return nil, err
	}

	return newRule.ToDeviceAddressLeaseRule()
}

// DisableDeviceAddressLeaseRule sets enabled to false for the device lease rule for the device.
// Returns the updated rule or ErrRuleNotFound if no rule exists.
func (s *Service) DisableDeviceAddressLeaseRule(ctx context.Context, deviceID device.DeviceID) (*DeviceAddressLeaseRule, error) {
	rule, err := s.repo.DisableRule(ctx, deviceID, RuleTypeDeviceAddressLease)
	if err != nil {
		return nil, err
	}
	return rule.ToDeviceAddressLeaseRule()
}
