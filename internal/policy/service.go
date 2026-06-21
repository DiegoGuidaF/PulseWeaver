package policy

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
)

// NetworkPoliciesProvider is the interface the policy cache consumes to load
// enabled CIDR ranges. Implemented by networkpolicies.Repository.
type NetworkPoliciesProvider interface {
	GetEnabledCacheEntries(ctx context.Context) ([]networkpolicies.CacheEntry, error)
}

// GeoIPResolver resolves an IP to geographic and ASN data.
// Implementations must be safe for concurrent use and fail-open.
// A nil GeoIPResolver is valid — the service skips enrichment.
type GeoIPResolver interface {
	Resolve(ip string) geoip.Result
}

// EnabledIPsProvider is the cross-domain interface the policy service consumes.
// Implemented by device.Service.
type EnabledIPsProvider interface {
	GetEnabledIPEntries(ctx context.Context) ([]device.IPEntry, error)
}

// HostAccessProvider is the cross-domain interface for host-level access grants.
// Implemented by useraccess.Service.
type HostAccessProvider interface {
	GetAllUserHostAccess(ctx context.Context) ([]UserHostAccess, error)
}

// defaultReconcileInterval is how often RunListener rebuilds the cache
// unconditionally, independent of change events. It is a staleness backstop, not
// a tuning knob: the event path already propagates real changes in ~ms, so this
// only bounds the worst case when an event is dropped or a rebuild failed. Kept
// internal deliberately — the only sensible values cluster tightly around this.
const defaultReconcileInterval = 60 * time.Second

// Service maintains an in-memory cache of enabled IPs for fast forward-auth lookups.
type Service struct {
	ipProvider              EnabledIPsProvider
	hostProvider            HostAccessProvider
	geoResolver             GeoIPResolver
	networkPoliciesProvider NetworkPoliciesProvider
	apiSecretHash           [32]byte
	trustedProxy            netip.Addr
	reconcileInterval       time.Duration // periodic full-rebuild backstop; defaults to defaultReconcileInterval
	mu                      sync.RWMutex
	ipSet                   map[netip.Addr]ipSetEntry
	networkPolicies         []networkPolicyCacheEntry
	lastRefreshedAt         time.Time
	lastRefreshDurationMs   int64
	refreshSignal           chan struct{} // buffered cap 1
	logger                  *slog.Logger
	observers               []DecisionObserver
}

func NewService(
	ipProvider EnabledIPsProvider,
	hostProvider HostAccessProvider,
	geoResolver GeoIPResolver,
	networkPoliciesProvider NetworkPoliciesProvider,
	secret string,
	logger *slog.Logger,
	trustedProxy netip.Addr,
) (*Service, error) {
	componentLogger := logger.With(slog.String(logging.AttrKeyComponent, "policy"))
	if strings.TrimSpace(secret) == "" {
		return nil, ErrSecretNotConfigured
	}
	return &Service{
		ipProvider:              ipProvider,
		hostProvider:            hostProvider,
		geoResolver:             geoResolver,
		networkPoliciesProvider: networkPoliciesProvider,
		apiSecretHash:           sha256.Sum256([]byte(secret)),
		trustedProxy:            trustedProxy,
		reconcileInterval:       defaultReconcileInterval,
		ipSet:                   make(map[netip.Addr]ipSetEntry),
		refreshSignal:           make(chan struct{}, 1),
		logger:                  componentLogger,
	}, nil
}

// Initialize populates the cache on startup. Called once from app.go.
func (s *Service) Initialize(ctx context.Context) error {
	return s.refreshCache(ctx)
}

// LastRefreshedAt returns the time of the most recent successful cache refresh.
func (s *Service) LastRefreshedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRefreshedAt
}

func (s *Service) AddDecisionObserver(o DecisionObserver) {
	if o == nil {
		return
	}
	s.observers = append(s.observers, o)
}

func (s *Service) notifyDecisionObservers(ctx context.Context, event DecisionEvent) {
	for _, o := range s.observers {
		o.OnDecision(ctx, event)
	}
}
