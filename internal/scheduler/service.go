package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// Job is a unit of work the scheduler executes on every tick.
type Job interface {
	Run(ctx context.Context) error
}

type Service struct {
	jobs   []Job
	logger *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
	return &Service{
		logger: logger.With(slog.String(logging.AttrKeyComponent, "rule_scheduler")),
	}
}

// AddJob registers a job to be executed on every scheduler tick.
func (s *Service) AddJob(job Job) {
	s.jobs = append(s.jobs, job)
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

// ExecuteScheduledRules runs all registered jobs in order.
func (s *Service) ExecuteScheduledRules(ctx context.Context) error {
	for _, job := range s.jobs {
		if err := job.Run(ctx); err != nil {
			return err
		}
	}
	return nil
}
