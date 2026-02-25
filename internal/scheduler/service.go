package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

type AddressDisabler interface {
	DisableAddresses(ctx context.Context, addressIDs []device.AddressID, source device.StatusSource) error
}

type ExpiredAddressFinder interface {
	GetExpiredAddressIDs(ctx context.Context) ([]device.AddressID, error)
}

type Service struct {
	expiredAddressFinder ExpiredAddressFinder
	addressDisabler      AddressDisabler
}

func NewService(expiredAddressFinder ExpiredAddressFinder, addressDisabler AddressDisabler) (*Service, error) {
	if expiredAddressFinder == nil {
		return nil, errors.New("expired address finder not configured")
	}
	if addressDisabler == nil {
		return nil, errors.New("address disabler not configured")
	}
	return &Service{
		expiredAddressFinder: expiredAddressFinder,
		addressDisabler:      addressDisabler,
	}, nil
}

// RunSchedule starts the background worker that evaluates time-based rules.
// It runs an immediate sweep on startup, then ticks at the given interval.
func (s *Service) RunSchedule(ctx context.Context, interval time.Duration) error {
	ctx, logger := logging.Enrich(ctx, slog.String(AttrKeyComponent, "rule_scheduler"))

	logger.Info("starting rule engine scheduler", slog.Duration("interval", interval))

	// Run all time-based rules immediately on startup
	s.executeScheduledRules(ctx, logger)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.executeScheduledRules(ctx, logger)
		case <-ctx.Done():
			logger.Info("rule engine scheduler stopped")
			return nil
		}
	}
}

// executeScheduledRules runs all rules that require periodic background checks.
func (s *Service) executeScheduledRules(ctx context.Context, logger *slog.Logger) {
	if err := s.executeAutoExpiry(ctx); err != nil {
		logger.Error("auto-expiry rule execution failed", slog.Any(AttrKeyError, err))
	}
}
