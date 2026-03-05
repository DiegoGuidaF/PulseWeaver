package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
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
	logger               *slog.Logger
}

func NewService(expiredAddressFinder ExpiredAddressFinder, addressDisabler AddressDisabler, logger *slog.Logger) (*Service, error) {
	if expiredAddressFinder == nil {
		return nil, errors.New("expired address finder not configured")
	}
	if addressDisabler == nil {
		return nil, errors.New("address disabler not configured")
	}
	return &Service{
		expiredAddressFinder: expiredAddressFinder,
		addressDisabler:      addressDisabler,
		logger:               logger.With(slog.String(logging.AttrKeyComponent, "rule_scheduler")),
	}, nil
}

// RunSchedule starts the background worker that evaluates time-based rules.
// It runs an immediate sweep on startup, then ticks at the given interval.
func (s *Service) RunSchedule(ctx context.Context, interval time.Duration) error {
	s.logger.InfoContext(ctx, "starting rule engine scheduler", slog.Duration("interval", interval))

	// Run all time-based rules immediately on startup
	s.executeScheduledRules(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.executeScheduledRules(ctx)
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "rule engine scheduler stopped")
			return nil
		}
	}
}

// executeScheduledRules runs all rules that require periodic background checks.
func (s *Service) executeScheduledRules(ctx context.Context) {
	if err := s.executeAutoExpiry(ctx); err != nil {
		s.logger.ErrorContext(ctx, "auto-expiry rule execution failed", slog.Any(AttrKeyError, err))
	}
}
