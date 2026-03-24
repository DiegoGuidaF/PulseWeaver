package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// RollupExecutor aggregates raw audit log rows into hourly traffic buckets.
type RollupExecutor interface {
	RunRollup(ctx context.Context, from, to time.Time) error
}

// executeRollup computes the previous complete hour boundary and runs the rollup.
// It is a no-op if the rollup already ran for this hour (tracked via lastRollupHour).
func (s *Service) executeRollup(ctx context.Context) error {
	if s.rollupExecutor == nil {
		return nil
	}

	now := time.Now()
	currentHour := now.Truncate(time.Hour)

	// Skip if we already ran for this hour.
	if s.lastRollupHour.Equal(currentHour) {
		s.logger.DebugContext(ctx, "rollup already executed for current hour, skipping")
		return nil
	}

	// Roll up the previous complete hour: [currentHour-1h, currentHour)
	from := currentHour.Add(-time.Hour)
	to := currentHour

	s.logger.InfoContext(ctx, "starting traffic rollup",
		slog.Time("from", from),
		slog.Time("to", to),
	)

	if err := s.rollupExecutor.RunRollup(ctx, from, to); err != nil {
		return err
	}

	s.lastRollupHour = currentHour
	s.logger.InfoContext(ctx, "traffic rollup completed")
	return nil
}
