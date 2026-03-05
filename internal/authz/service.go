package authz

import (
	"context"
	"log/slog"
	"sync"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

// EnabledIPsProvider is the cross-domain interface the authz service consumes.
// Implemented by device.Service.
type EnabledIPsProvider interface {
	GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
}

// Service maintains an in-memory cache of enabled IPs for fast forward-auth lookups.
type Service struct {
	provider            EnabledIPsProvider
	secret              string // AUTHZ_API_SECRET; empty = fail-closed
	mu                  sync.RWMutex
	ipSet               map[string]struct{}
	addressChangeSignal chan struct{} // buffered cap 1
	logger              *slog.Logger
}

func NewService(provider EnabledIPsProvider, secret string, logger *slog.Logger) *Service {
	return &Service{
		provider:            provider,
		secret:              secret,
		ipSet:               make(map[string]struct{}),
		addressChangeSignal: make(chan struct{}, 1),
		logger:              logger.With(slog.String(logging.AttrKeyComponent, "authz")),
	}
}

// Initialize populates the cache on startup. Called once from app.go.
func (s *Service) Initialize(ctx context.Context) error {
	return s.refreshCache(ctx)
}

// OnAddressEvent implements device.AddressObserver.
// Non-blocking signal; context is intentionally discarded.
func (s *Service) OnAddressEvent(_ context.Context, _ device.AddressEvent) {
	select {
	case s.addressChangeSignal <- struct{}{}:
	default:
	}
}

// RunListener processes address change signals and refreshes the cache immediately.
// Runs until ctx is cancelled.
func (s *Service) RunListener(ctx context.Context) error {
	for {
		select {
		case <-s.addressChangeSignal:
			if err := s.refreshCache(ctx); err != nil {
				s.logger.ErrorContext(ctx, "authz cache refresh failed", slog.Any(logging.AttrKeyError, err))
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// ContainsIP reports whether ip is currently in the enabled set. Thread-safe.
func (s *Service) ContainsIP(ip string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.ipSet[ip]
	return ok
}

// Secret returns the configured internal secret. Used by the handler.
func (s *Service) Secret() string {
	return s.secret
}

// refreshCache queries enabled IPs and atomically replaces the in-memory set.
func (s *Service) refreshCache(ctx context.Context) error {
	ips, err := s.provider.GetEnabledUniqueIPs(ctx)
	if err != nil {
		return err
	}

	newSet := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		newSet[ip] = struct{}{}
	}

	s.mu.Lock()
	s.ipSet = newSet
	s.mu.Unlock()

	s.logger.DebugContext(ctx, "authz IP cache refreshed", slog.Int("ip_count", len(ips)))
	return nil
}
