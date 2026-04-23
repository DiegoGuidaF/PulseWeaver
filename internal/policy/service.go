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

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
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

// UserHostAccess is the per-user projection consumed by refreshCache.
type UserHostAccess struct {
	UserID          auth.UserID
	BypassAllowlist bool
	AllowedHosts    []string // case-folded FQDNs; pre-union of direct + group grants
}

// GeoIPResolver resolves an IP to geographic and ASN data.
// Implementations must be safe for concurrent use and fail-open.
// A nil GeoIPResolver is valid — the service skips enrichment.
type GeoIPResolver interface {
	Resolve(ip string) geoip.Result
}

type ipSetEntry struct {
	Contributors    []IPContributor // all devices at this IP; len > 1 = intersection applied
	BypassAllowlist bool
	AllowedHosts    map[string]struct{} // case-folded FQDNs; nil when all contributors bypass
}

// Service maintains an in-memory cache of enabled IPs for fast forward-auth lookups.
type Service struct {
	ipProvider          EnabledIPsProvider
	hostProvider        HostAccessProvider
	geoResolver         GeoIPResolver
	apiSecretHash       [32]byte
	trustedProxy        netip.Addr
	mu                  sync.RWMutex
	ipSet               map[string]ipSetEntry
	addressChangeSignal chan struct{} // buffered cap 1
	hostAccessSignal    chan struct{} // buffered cap 1
	logger              *slog.Logger
	observers           []DecisionObserver
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
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, new(DenyReasonInvalidToken), nil, req, geo, time.Since(start).Microseconds()))
		return ErrInvalidBearerToken
	}

	entry, ok := s.lookupIP(ctx, req.ClientIP)
	if !ok {
		s.logger.DebugContext(ctx, "IP not enabled")
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, new(DenyReasonIPNotRegistered), nil, req, geo, time.Since(start).Microseconds()))
		return ErrIPNotEnabled
	}

	if !entry.BypassAllowlist {
		host := ""
		if req.TargetHost != nil {
			host = strings.ToLower(*req.TargetHost)
		}
		if _, ok := entry.AllowedHosts[host]; !ok {
			s.logger.DebugContext(ctx, "host not in allowlist", slog.String("host", host))
			s.notifyDecisionObservers(ctx, NewDecisionEvent(false, new(DenyReasonHostNotAllowed), entry.Contributors, req, geo, time.Since(start).Microseconds()))
			return ErrHostNotAllowed
		}
	}

	s.logger.DebugContext(ctx, "IP is enabled")
	s.notifyDecisionObservers(ctx, NewDecisionEvent(true, nil, entry.Contributors, req, geo, time.Since(start).Microseconds()))

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

// refreshCache queries enabled IPs and host access grants, then atomically
// replaces the in-memory set with deny-wins intersection for shared IPs.
func (s *Service) refreshCache(ctx context.Context) error {
	//TODO: Log refresh time (start to finish)
	ipEntries, err := s.ipProvider.GetEnabledIPEntries(ctx)
	if err != nil {
		return err
	}

	var hostAccess []UserHostAccess
	if s.hostProvider != nil {
		hostAccess, err = s.hostProvider.GetAllUserHostAccess(ctx)
		if err != nil {
			return err
		}
	}

	accessByUser := make(map[auth.UserID]UserHostAccess, len(hostAccess))
	hostSetByUser := make(map[auth.UserID]map[string]struct{}, len(hostAccess))
	// Parse and build the list of hosts accessible per user
	for _, ua := range hostAccess {
		accessByUser[ua.UserID] = ua

		hosts := make(map[string]struct{}, len(ua.AllowedHosts))
		for _, h := range ua.AllowedHosts {
			hosts[h] = struct{}{}
		}
		hostSetByUser[ua.UserID] = hosts
	}

	type accumulator struct {
		contributors      []IPContributor
		allBypass         bool
		hasRestrictedUser bool
		allowedHosts      map[string]struct{}
	}

	byIP := make(map[string]*accumulator, len(ipEntries))

	for _, e := range ipEntries {
		acc := byIP[e.IP]
		if acc == nil {
			acc = &accumulator{allBypass: true}
			byIP[e.IP] = acc
		}

		acc.contributors = append(acc.contributors, IPContributor{
			DeviceID:  e.DeviceID,
			AddressID: e.AddressID,
			UserID:    e.UserID,
		})

		ua := accessByUser[e.UserID]
		acc.allBypass = acc.allBypass && ua.BypassAllowlist

		if ua.BypassAllowlist {
			continue // bypass users are intersection-neutral
		}

		userHosts := hostSetByUser[e.UserID]
		if !acc.hasRestrictedUser {
			acc.allowedHosts = cloneHostSet(userHosts)
			acc.hasRestrictedUser = true
			continue
		}

		intersectHostSets(acc.allowedHosts, userHosts)
	}

	newSet := make(map[string]ipSetEntry, len(byIP))
	for ip, acc := range byIP {
		newSet[ip] = ipSetEntry{
			Contributors:    acc.contributors,
			BypassAllowlist: acc.allBypass,
			AllowedHosts:    acc.allowedHosts,
		}
	}

	s.mu.Lock()
	s.ipSet = newSet
	s.mu.Unlock()

	s.logger.DebugContext(ctx, "policy IP cache refreshed", slog.Int(logging.AttrKeyIPCount, len(newSet)))
	return nil
}

func cloneHostSet(src map[string]struct{}) map[string]struct{} {
	if len(src) == 0 {
		return map[string]struct{}{}
	}
	dst := make(map[string]struct{}, len(src))
	for h := range src {
		dst[h] = struct{}{}
	}
	return dst
}

// intersectHostSets Removes elements from dsr that are not present in src
func intersectHostSets(dst, src map[string]struct{}) {
	for h := range dst {
		if _, ok := src[h]; !ok {
			delete(dst, h)
		}
	}
}
