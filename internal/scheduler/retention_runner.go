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

type AggregatePruner interface {
	DeleteAggregatesOlderThan(ctx context.Context, before time.Time) (int64, error)
	DeleteAttributionAggregatesOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type AnomalyPruner interface {
	DeleteAnomaliesOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// RetentionJob deletes rows older than the configured retention windows.
// Raw data (access_log, address_events) is pruned at retentionDays; the
// hourly traffic aggregates serving wide dashboard windows — and anomalies,
// which summarize that aggregate-era data — are pruned at the independent,
// longer aggregateRetentionDays horizon. A days value of 0 disables the
// corresponding deletion entirely.
type RetentionJob struct {
	accessLogPruner        AccessLogPruner
	addressEventPruner     AddressEventPruner
	aggregatePruner        AggregatePruner
	anomalyPruner          AnomalyPruner
	retentionDays          int
	aggregateRetentionDays int
	lastRanAt              time.Time
	logger                 *slog.Logger
}

func NewRetentionJob(
	accessLogPruner AccessLogPruner,
	addressEventPruner AddressEventPruner,
	aggregatePruner AggregatePruner,
	anomalyPruner AnomalyPruner,
	retentionDays int,
	aggregateRetentionDays int,
	logger *slog.Logger,
) *RetentionJob {
	return &RetentionJob{
		accessLogPruner:        accessLogPruner,
		addressEventPruner:     addressEventPruner,
		aggregatePruner:        aggregatePruner,
		anomalyPruner:          anomalyPruner,
		retentionDays:          retentionDays,
		aggregateRetentionDays: aggregateRetentionDays,
		logger:                 logger.With(slog.String(AttrKeyComponent, "retention_job")),
	}
}

func (j *RetentionJob) Run(ctx context.Context) error {
	if j.retentionDays == 0 && j.aggregateRetentionDays == 0 {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	if j.lastRanAt.Equal(today) {
		return nil
	}

	if j.retentionDays > 0 {
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
	}

	if j.aggregateRetentionDays > 0 {
		aggregateCutoff := time.Now().AddDate(0, 0, -j.aggregateRetentionDays)

		deletedAggregates, err := j.aggregatePruner.DeleteAggregatesOlderThan(ctx, aggregateCutoff)
		if err != nil {
			j.logger.ErrorContext(ctx, "traffic aggregate retention failed", slog.Any(AttrKeyError, err))
			return err
		}
		j.logger.InfoContext(ctx, "traffic aggregate retention complete",
			slog.Int64(AttrKeyCount, deletedAggregates),
			slog.Int("retention_days", j.aggregateRetentionDays),
		)

		deletedAttributionAggregates, err := j.aggregatePruner.DeleteAttributionAggregatesOlderThan(ctx, aggregateCutoff)
		if err != nil {
			j.logger.ErrorContext(ctx, "attribution aggregate retention failed", slog.Any(AttrKeyError, err))
			return err
		}
		j.logger.InfoContext(ctx, "attribution aggregate retention complete",
			slog.Int64(AttrKeyCount, deletedAttributionAggregates),
			slog.Int("retention_days", j.aggregateRetentionDays),
		)

		deletedAnomalies, err := j.anomalyPruner.DeleteAnomaliesOlderThan(ctx, aggregateCutoff)
		if err != nil {
			j.logger.ErrorContext(ctx, "anomaly retention failed", slog.Any(AttrKeyError, err))
			return err
		}
		j.logger.InfoContext(ctx, "anomaly retention complete",
			slog.Int64(AttrKeyCount, deletedAnomalies),
			slog.Int("retention_days", j.aggregateRetentionDays),
		)
	}

	j.lastRanAt = today
	return nil
}
