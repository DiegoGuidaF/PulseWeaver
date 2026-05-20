package scheduler

import (
	"context"
	"log/slog"
	"time"
)

type AccessLogPruner interface {
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type AddressEventPruner interface {
	DeleteAddressEventsOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// RetentionJob deletes rows older than the configured retention window.
// A retentionDays value of 0 disables deletion entirely.
type RetentionJob struct {
	accessLogPruner    AccessLogPruner
	addressEventPruner AddressEventPruner
	retentionDays      int
	lastRanAt          time.Time
	logger             *slog.Logger
}

func NewRetentionJob(
	accessLogPruner AccessLogPruner,
	addressEventPruner AddressEventPruner,
	retentionDays int,
	logger *slog.Logger,
) *RetentionJob {
	return &RetentionJob{
		accessLogPruner:    accessLogPruner,
		addressEventPruner: addressEventPruner,
		retentionDays:      retentionDays,
		logger:             logger.With(slog.String(AttrKeyComponent, "retention_job")),
	}
}

func (j *RetentionJob) Run(ctx context.Context) error {
	if j.retentionDays == 0 {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	if j.lastRanAt.Equal(today) {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -j.retentionDays)

	deleted, err := j.accessLogPruner.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		j.logger.ErrorContext(ctx, "access log retention failed", slog.Any(AttrKeyError, err))
		return err
	}
	j.logger.InfoContext(ctx, "access log retention complete",
		slog.Int64(AttrKeyCount, deleted),
		slog.Int("retention_days", j.retentionDays),
	)

	deletedEvents, err := j.addressEventPruner.DeleteAddressEventsOlderThan(ctx, cutoff)
	if err != nil {
		j.logger.ErrorContext(ctx, "address event retention failed", slog.Any(AttrKeyError, err))
		return err
	}
	j.logger.InfoContext(ctx, "address event retention complete",
		slog.Int64(AttrKeyCount, deletedEvents),
		slog.Int("retention_days", j.retentionDays),
	)

	j.lastRanAt = today
	return nil
}
