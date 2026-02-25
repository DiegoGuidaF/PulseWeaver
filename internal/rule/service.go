package rule

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

// repository is the narrow interface the rule service depends on.
type repository interface {
	GetRuleByDeviceAndType(ctx context.Context, deviceID device.DeviceID, ruleType RuleType) (*Rule, error)
}

// Service owns rule evaluation and CRUD operations.
// It also implements the device.RuleEvaluator interface.
type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetAddressTTL implements device.RuleEvaluator.
// It returns the TTL to apply for IP auto-expiry for the given device,
// or nil if no active rule exists.
func (s *Service) GetAddressTTL(ctx context.Context, deviceID device.DeviceID) (*time.Duration, error) {
	ctx, logger := logging.Enrich(ctx,
		slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
		slog.String(AttrKeyRuleType, string(RuleTypeIPAutoExpiry)),
	)

	logger.Debug("evaluating ip auto expiry rule")

	rule, err := s.repo.GetRuleByDeviceAndType(ctx, deviceID, RuleTypeIPAutoExpiry)
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

	cfg, err := parseRuleConfigIPAutoExpiry(rule.Config)
	if err != nil {
		logger.Error("invalid ip auto expiry rule config",
			slog.Int64(AttrKeyRuleID, int64(rule.ID)),
			slog.Any(AttrKeyError, err),
		)
		return nil, ErrInvalidRuleConfig
	}

	ttl := time.Duration(cfg.TTLSeconds) * time.Second
	if ttl <= 0 {
		logger.Error("ip auto expiry rule produced non-positive ttl",
			slog.Int64(AttrKeyRuleID, int64(rule.ID)),
		)
		return nil, ErrInvalidRuleConfig
	}

	logger.Debug("ip auto expiry rule evaluated",
		slog.Int64(AttrKeyRuleID, int64(rule.ID)),
		slog.Int("ttl_seconds", cfg.TTLSeconds),
	)

	return &ttl, nil
}
