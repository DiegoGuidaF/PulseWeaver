package policy

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/netip"
	"strings"
	"sync"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

// EnabledIPsProvider is the cross-domain interface the policy service consumes.
// Implemented by device.Service.
type EnabledIPsProvider interface {
	GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
}

// Service maintains an in-memory cache of enabled IPs for fast forward-auth lookups.
type Service struct {
	provider            EnabledIPsProvider
	apiSecretHash       [32]byte
	trustedProxy        netip.Addr
	mu                  sync.RWMutex
	ipSet               map[string]struct{}
	addressChangeSignal chan struct{} // buffered cap 1
	logger              *slog.Logger
}

func NewService(provider EnabledIPsProvider, secret string, logger *slog.Logger, trustedProxy netip.Addr) (*Service, error) {
	componentLogger := logger.With(slog.String(logging.AttrKeyComponent, "policy"))
	if strings.TrimSpace(secret) == "" {
		return nil, ErrSecretNotConfigured
	}

	return &Service{
		provider:            provider,
		apiSecretHash:       sha256.Sum256([]byte(secret)),
		trustedProxy:        trustedProxy,
		ipSet:               make(map[string]struct{}),
		addressChangeSignal: make(chan struct{}, 1),
		logger:              componentLogger,
	}, nil
}

// Initialize populates the cache on startup. Called once from app.go.
func (s *Service) Initialize(ctx context.Context) error {
	return s.refreshCache(ctx)
}

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

// RunListener processes address change signals and refreshes the cache immediately.
// Runs until ctx is cancelled.
func (s *Service) RunListener(ctx context.Context) error {
	for {
		select {
		case <-s.addressChangeSignal:
			if err := s.refreshCache(ctx); err != nil {
				s.logger.ErrorContext(ctx, "policy cache refresh failed", slog.Any(logging.AttrKeyError, err))
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// VerifyAccess validates bearer token and verifies that the IP is enabled.
func (s *Service) VerifyAccess(ctx context.Context, token, clientIP string) error {
	// Compare fixed-size 32-byte slices and avoid
	// leaking length information through early returns.
	tokenHash := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(tokenHash[:], s.apiSecretHash[:]) != 1 {
		s.logger.WarnContext(ctx, "policy: invalid bearer token")
		return ErrInvalidBearerToken
	}

	if !s.ContainsIP(clientIP) {
		return ErrIPNotEnabled
	}

	return nil
}

// ContainsIP reports whether ip is currently in the enabled set. Thread-safe.
func (s *Service) ContainsIP(ip string) bool {
	if s.trustedProxy.IsValid() {
		addr, err := netip.ParseAddr(ip)
		if err == nil && s.trustedProxy.Compare(addr) == 0 {
			s.logger.Warn("rejected trusted proxy IP authorization", slog.String(AttrKeyRequestIP, ip))
			return false
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.ipSet[ip]
	return ok
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

	s.logger.DebugContext(ctx, "policy IP cache refreshed", slog.Int("ip_count", len(ips)))
	return nil
}
