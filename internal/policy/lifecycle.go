package policy

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// triggerRefresh sends a non-blocking signal to refresh the full cache.
func (s *Service) triggerRefresh() {
	select {
	case s.refreshSignal <- struct{}{}:
	default:
	}
}

// OnAddressEvent implements device.AddressObserver.
// AddressRefreshed is ignored — the IP set is unchanged on a simple refresh.
func (s *Service) OnAddressEvent(_ context.Context, e device.AddressEvent) {
	if e.Type == device.EventTypeAddressRefreshed {
		return
	}
	s.triggerRefresh()
}

// OnHostAccessChanged implements hosts.Observer and useraccess.Observer.
func (s *Service) OnHostAccessChanged(_ context.Context) {
	s.triggerRefresh()
}

// OnNetworkPolicyChanged implements networkpolicies.PolicyChangeObserver.
func (s *Service) OnNetworkPolicyChanged(_ context.Context) {
	s.triggerRefresh()
}

// RunListener rebuilds the full cache on each change signal, and also on a
// periodic timer as a staleness backstop. Runs until ctx is cancelled.
//
// The periodic reconcile guards against the two ways the event path can leave
// the cache stale: a change signal that was never delivered, and a rebuild that
// failed (refreshCache swaps only on success, so a failed refresh keeps the old,
// more-permissive snapshot). For an authz cache a stale snapshot is a stale
// allow, so an unconditional rebuild bounds that exposure regardless of events.
//
// TODO: partial refreshes — an address change only needs to rebuild ipSet;
// a network policy change only needs to rebuild networkPolicies. Separating
// them would require two signals again and is not worth it without data on
// relative change frequency or refresh cost at scale.
func (s *Service) RunListener(ctx context.Context) error {
	interval := s.reconcileInterval
	if interval <= 0 {
		interval = defaultReconcileInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.refreshSignal:
			if err := s.refreshCache(ctx); err != nil {
				s.logger.ErrorContext(ctx, "policy cache refresh failed", slog.Any(logging.AttrKeyError, err))
			}
		case <-ticker.C:
			if err := s.refreshCache(ctx); err != nil {
				s.logger.ErrorContext(ctx, "policy cache periodic reconcile failed", slog.Any(logging.AttrKeyError, err))
			}
		case <-ctx.Done():
			return nil
		}
	}
}
