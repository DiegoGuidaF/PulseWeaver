package policy

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// EnabledIPsProvider is the cross-domain interface the policy service consumes.
// Implemented by device.Service.
type EnabledIPsProvider interface {
	GetEnabledIPEntries(ctx context.Context) ([]device.IPEntry, error)
}

// GeoIPResolver resolves an IP to geographic and ASN data.
// Implementations must be safe for concurrent use and fail-open.
// A nil GeoIPResolver is valid — the service skips enrichment.
type GeoIPResolver interface {
	Resolve(ip string) geoip.Result
}

type ipSetEntry struct {
	DeviceID  device.DeviceID
	AddressID device.AddressID
}

// Service maintains an in-memory cache of enabled IPs for fast forward-auth lookups.
type Service struct {
	provider            EnabledIPsProvider
	geoResolver         GeoIPResolver
	apiSecretHash       [32]byte
	trustedProxy        netip.Addr
	mu                  sync.RWMutex
	ipSet               map[string]ipSetEntry
	addressChangeSignal chan struct{} // buffered cap 1
	logger              *slog.Logger
	observers           []DecisionObserver
}

func NewService(provider EnabledIPsProvider, geoResolver GeoIPResolver, secret string, logger *slog.Logger, trustedProxy netip.Addr) (*Service, error) {
	componentLogger := logger.With(slog.String(logging.AttrKeyComponent, "policy"))
	if strings.TrimSpace(secret) == "" {
		return nil, ErrSecretNotConfigured
	}
	return &Service{
		provider:            provider,
		geoResolver:         geoResolver,
		apiSecretHash:       sha256.Sum256([]byte(secret)),
		trustedProxy:        trustedProxy,
		ipSet:               make(map[string]ipSetEntry),
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

// VerifyAccess validates bearer token and verifies that the IP is enabled, emitting a DecisionEvent.
func (s *Service) VerifyAccess(ctx context.Context, req *VerifyRequest) error {
	s.logger.DebugContext(ctx, "Verify access for ip")
	start := time.Now()

	geo := s.geoResolver.Resolve(req.ClientIP)

	tokenHash := sha256.Sum256([]byte(req.Token))
	if subtle.ConstantTimeCompare(tokenHash[:], s.apiSecretHash[:]) != 1 {
		s.logger.WarnContext(ctx, "policy: invalid bearer token")
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, new(DenyReasonInvalidToken), nil, nil, req, geo, time.Since(start).Microseconds()))
		return ErrInvalidBearerToken
	}

	entry, ok := s.lookupIP(ctx, req.ClientIP)
	if !ok {
		s.logger.DebugContext(ctx, "IP not enabled")
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, new(DenyReasonIPNotRegistered), nil, nil, req, geo, time.Since(start).Microseconds()))
		return ErrIPNotEnabled
	}

	s.logger.DebugContext(ctx, "IP is enabled")
	s.notifyDecisionObservers(ctx, NewDecisionEvent(true, nil, &entry.DeviceID, &entry.AddressID, req, geo, time.Since(start).Microseconds()))

	return nil
}

// lookupIP returns the ipSetEntry for ip if it is currently in the enabled set.
// It rejects the trusted proxy IP regardless of registration status. Thread-safe.
func (s *Service) lookupIP(ctx context.Context, ip string) (ipSetEntry, bool) {
	if s.trustedProxy.IsValid() {
		addr, err := netip.ParseAddr(ip)
		if err == nil && s.trustedProxy.Compare(addr) == 0 {
			s.logger.WarnContext(ctx, "rejected trusted proxy IP authorization", slog.String(AttrKeyRequestIP, ip))
			return ipSetEntry{}, false
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.ipSet[ip]
	s.logger.DebugContext(ctx, "found IP", slog.String(AttrKeyRequestIP, ip))
	return entry, ok
}

// refreshCache queries enabled IPs and atomically replaces the in-memory set.
func (s *Service) refreshCache(ctx context.Context) error {
	entries, err := s.provider.GetEnabledIPEntries(ctx)
	if err != nil {
		return err
	}

	newSet := make(map[string]ipSetEntry, len(entries))
	for _, e := range entries {
		newSet[e.IP] = ipSetEntry{DeviceID: e.DeviceID, AddressID: e.AddressID}
	}

	s.mu.Lock()
	s.ipSet = newSet
	s.mu.Unlock()

	s.logger.DebugContext(ctx, "policy IP cache refreshed", slog.Int(logging.AttrKeyIPCount, len(entries)))
	return nil
}
