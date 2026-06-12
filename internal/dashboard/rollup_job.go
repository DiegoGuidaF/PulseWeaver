package dashboard

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// catchUpChunk bounds the span of a single RunRollup statement during catch-up
// so a long backfill never holds one large write transaction.
const catchUpChunk = 24 * time.Hour

// RollupJob rolls up every complete hour not yet covered by the aggregates:
// from lastRollupAt up to the current hour boundary. Hours during which the
// app was not running (downtime, restarts, bulk-seeded data) are therefore
// caught up on the next run instead of being permanently absent.
// lastRollupAt is seeded from the DB on the first run so the guard survives
// restarts; re-rolling an hour whose raw rows were already pruned is a no-op
// (INSERT OR REPLACE with no source rows), so catch-up never destroys
// aggregates it cannot rebuild.
type RollupJob struct {
	repo *Repository
	// lastRollupAt is the hour boundary up to which aggregates are complete
	// (exclusive end of the last rolled window).
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
	currentHour := time.Now().Truncate(time.Hour)

	if !j.initialized {
		last, err := j.repo.LastRollupAt(ctx)
		if err != nil {
			return err
		}
		if !last.IsZero() {
			// MAX(bucket_at) is the start of the last rolled bucket; everything
			// up to that bucket's end is covered.
			j.lastRollupAt = last.Add(time.Hour)
		}
		j.initialized = true
	}

	if !j.lastRollupAt.IsZero() && !j.lastRollupAt.Before(currentHour) {
		j.logger.DebugContext(ctx, "rollup already executed for current hour, skipping")
		return nil
	}

	// Hours before the earliest retained raw row cannot produce aggregates —
	// they bound the first-run backfill and skip pointless catch-up scans.
	earliest, err := j.repo.EarliestAccessLogAt(ctx)
	if err != nil {
		return err
	}
	if earliest.IsZero() {
		j.lastRollupAt = currentHour
		return nil
	}

	from := j.lastRollupAt
	if earliestHour := earliest.Truncate(time.Hour); from.Before(earliestHour) {
		from = earliestHour
	}
	if !from.Before(currentHour) {
		j.lastRollupAt = currentHour
		return nil
	}

	j.logger.InfoContext(ctx, "starting traffic rollup",
		slog.Time("from", from),
		slog.Time("to", currentHour),
	)

	for chunkFrom := from; chunkFrom.Before(currentHour); {
		chunkTo := chunkFrom.Add(catchUpChunk)
		if chunkTo.After(currentHour) {
			chunkTo = currentHour
		}
		if err := j.repo.RunRollup(ctx, chunkFrom, chunkTo); err != nil {
			return err
		}
		// Advance per chunk so a failed catch-up resumes where it stopped.
		j.lastRollupAt = chunkTo
		chunkFrom = chunkTo
	}

	j.logger.InfoContext(ctx, "traffic rollup completed")
	return nil
}
