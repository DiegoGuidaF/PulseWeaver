package policy

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// OnAddressEvent implements device.AddressObserver.
// Non-blocking signal; context is intentionally discarded.
// AddressRefreshed is ignored (no cache refresh) since the IP set is unchanged.
func (s *Service) OnAddressEvent(_ context.Context, e device.AddressEvent) {
	if e.Type == device.EventTypeAddressRefreshed {
		return
	}
	select {
	case s.addressChangeSignal <- struct{}{}:
	default:
	}
}

// OnHostAccessChanged implements hostaccess.Observer.
func (s *Service) OnHostAccessChanged(_ context.Context) {
	select {
	case s.hostAccessSignal <- struct{}{}:
	default:
	}
}

// RunListener processes address and host-access change signals, refreshing the cache.
// Runs until ctx is cancelled.
func (s *Service) RunListener(ctx context.Context) error {
	for {
		select {
		case <-s.addressChangeSignal:
			if err := s.refreshCache(ctx); err != nil {
				s.logger.ErrorContext(ctx, "policy cache refresh failed", slog.Any(logging.AttrKeyError, err))
			}
		case <-s.hostAccessSignal:
			if err := s.refreshCache(ctx); err != nil {
				s.logger.ErrorContext(ctx, "policy cache refresh failed (host access)", slog.Any(logging.AttrKeyError, err))
			}
		case <-ctx.Done():
			return nil
		}
	}
}
