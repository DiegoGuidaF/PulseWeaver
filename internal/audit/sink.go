package audit

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// Sink receives DecisionEvents on a buffered channel and batch-inserts them.
type Sink struct {
	ch     chan policy.DecisionEvent
	repo   repository
	logger *slog.Logger
}

type repository interface {
	BatchInsert(ctx context.Context, events []policy.DecisionEvent) error
}

func NewSink(repo repository, logger *slog.Logger) *Sink {
	return &Sink{
		ch:     make(chan policy.DecisionEvent, 500),
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "audit")),
	}
}

// OnDecision implements policy.DecisionObserver.
func (s *Sink) OnDecision(_ context.Context, e policy.DecisionEvent) {
	select {
	case s.ch <- e:
	default:
		s.logger.Error("audit buffer full, event dropped")
	}
}

// Run processes the channel until ctx is cancelled, flushing on interval or batch fill.
// Flush interval: 2s. Batch size: 50 events.
// On context cancellation Run drains the channel and performs a final flush before returning.
func (s *Sink) Run(ctx context.Context) error {
	const (
		batchSize     = 50
		flushInterval = 2 * time.Second
	)

	buffer := make([]policy.DecisionEvent, 0, batchSize)
	flush := func(ctx context.Context) {
		if len(buffer) == 0 {
			return
		}
		events := make([]policy.DecisionEvent, len(buffer))
		copy(events, buffer)
		buffer = buffer[:0]

		s.logger.DebugContext(ctx, "flushing decision events", slog.Int(logging.AttrKeyCount, len(events)))
		if err := s.repo.BatchInsert(ctx, events); err != nil {
			// Best-effort logging; audit failures must not crash the app.
			s.logger.ErrorContext(ctx, "failed to flush audit events", slog.Any(logging.AttrKeyError, err))
		}
	}

	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Drain the channel before final flush.
			s.logger.InfoContext(ctx, "stopping sink")
			drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for {
				select {
				case e := <-s.ch:
					buffer = append(buffer, e)
					if len(buffer) >= batchSize {
						flush(drainCtx)
					}
				default:
					flush(drainCtx)
					return nil
				}
			}
		case e := <-s.ch:
			buffer = append(buffer, e)
			if len(buffer) >= batchSize {
				flush(ctx)
			}
		case <-ticker.C:
			flush(ctx)
		}
	}
}
