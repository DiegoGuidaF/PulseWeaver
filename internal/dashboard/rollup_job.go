package dashboard

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// RollupJob computes the previous complete hour boundary and runs the rollup.
// It is a no-op if the rollup already ran for the current hour.
// lastRollupAt is seeded from the DB on the first run so the guard survives restarts.
type RollupJob struct {
	repo         *Repository
	lastRollupAt time.Time
	initialized  bool
	logger       *slog.Logger
}

func (r *Repository) NewRollupJob(logger *slog.Logger) *RollupJob {
	return &RollupJob{
		repo:   r,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "rollup_job")),
	}
}

func (j *RollupJob) Run(ctx context.Context) error {
	now := time.Now()
	currentHour := now.Truncate(time.Hour)

	if !j.initialized {
		last, err := j.repo.LastRollupAt(ctx)
		if err != nil {
			return err
		}
		j.lastRollupAt = last
		j.initialized = true
	}

	if j.lastRollupAt.Equal(currentHour) {
		j.logger.DebugContext(ctx, "rollup already executed for current hour, skipping")
		return nil
	}

	from := currentHour.Add(-time.Hour)
	to := currentHour

	j.logger.InfoContext(ctx, "starting traffic rollup",
		slog.Time("from", from),
		slog.Time("to", to),
	)

	if err := j.repo.RunRollup(ctx, from, to); err != nil {
		return err
	}

	j.lastRollupAt = currentHour
	j.logger.InfoContext(ctx, "traffic rollup completed")
	return nil
}
