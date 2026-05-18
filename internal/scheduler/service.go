package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type AddressDisabler interface {
	DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source device.EventSource) error
}

type ExpiredAddressFinder interface {
	GetExpiredAddressIDs(ctx context.Context) ([]ids.AddressID, error)
}

type Service struct {
	expiredAddressFinder ExpiredAddressFinder
	addressDisabler      AddressDisabler
	rollupExecutor       RollupExecutor
	lastRollupHour       time.Time
	logger               *slog.Logger
}

func NewService(expiredAddressFinder ExpiredAddressFinder, addressDisabler AddressDisabler, rollupExecutor RollupExecutor, logger *slog.Logger) (*Service, error) {
	if expiredAddressFinder == nil {
		return nil, errors.New("expired address finder not configured")
	}
	if addressDisabler == nil {
		return nil, errors.New("address disabler not configured")
	}
	return &Service{
		expiredAddressFinder: expiredAddressFinder,
		addressDisabler:      addressDisabler,
		rollupExecutor:       rollupExecutor,
		logger:               logger.With(slog.String(logging.AttrKeyComponent, "rule_scheduler")),
	}, nil
}

// RunSchedule starts the background worker that evaluates time-based rules.
// It runs an immediate sweep on startup, then ticks at the given interval.
func (s *Service) RunSchedule(ctx context.Context, interval time.Duration) error {
	s.logger.InfoContext(ctx, "starting rule engine scheduler", slog.Duration("interval", interval))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = s.ExecuteScheduledRules(ctx)
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "rule engine scheduler stopped")
			return nil
		}
	}
}

// ExecuteScheduledRules runs all rules that require periodic background checks.
func (s *Service) ExecuteScheduledRules(ctx context.Context) error {
	if err := s.executeAutoExpiry(ctx); err != nil {
		s.logger.ErrorContext(ctx, "auto-expiry rule execution failed", slog.Any(AttrKeyError, err))
		return err
	}
	if err := s.executeRollup(ctx); err != nil {
		s.logger.ErrorContext(ctx, "traffic rollup execution failed", slog.Any(AttrKeyError, err))
		return err
	}
	return nil
}
