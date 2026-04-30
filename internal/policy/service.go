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
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// EnabledIPsProvider is the cross-domain interface the policy service consumes.
// Implemented by device.Service.
type EnabledIPsProvider interface {
	GetEnabledIPEntries(ctx context.Context) ([]device.IPEntry, error)
}

// HostAccessProvider is the cross-domain interface for host-level access grants.
// Implemented by hostaccess.Service.
type HostAccessProvider interface {
	GetAllUserHostAccess(ctx context.Context) ([]UserHostAccess, error)
}

// Service maintains an in-memory cache of enabled IPs for fast forward-auth lookups.
type Service struct {
	ipProvider            EnabledIPsProvider
	hostProvider          HostAccessProvider
	geoResolver           GeoIPResolver
	apiSecretHash         [32]byte
	trustedProxy          netip.Addr
	mu                    sync.RWMutex
	ipSet                 map[string]ipSetEntry
	lastRefreshedAt       time.Time
	lastRefreshDurationMs int64
	addressChangeSignal   chan struct{} // buffered cap 1
	hostAccessSignal      chan struct{} // buffered cap 1
	logger                *slog.Logger
	observers             []DecisionObserver
}

func NewService(
	ipProvider EnabledIPsProvider,
	hostProvider HostAccessProvider,
	geoResolver GeoIPResolver,
	secret string,
	logger *slog.Logger,
	trustedProxy netip.Addr,
) (*Service, error) {
	componentLogger := logger.With(slog.String(logging.AttrKeyComponent, "policy"))
	if strings.TrimSpace(secret) == "" {
		return nil, ErrSecretNotConfigured
	}
	return &Service{
		ipProvider:          ipProvider,
		hostProvider:        hostProvider,
		geoResolver:         geoResolver,
		apiSecretHash:       sha256.Sum256([]byte(secret)),
		trustedProxy:        trustedProxy,
		ipSet:               make(map[string]ipSetEntry),
		addressChangeSignal: make(chan struct{}, 1),
		hostAccessSignal:    make(chan struct{}, 1),
		logger:              componentLogger,
	}, nil
}

// Initialize populates the cache on startup. Called once from app.go.
func (s *Service) Initialize(ctx context.Context) error {
	return s.refreshCache(ctx)
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
