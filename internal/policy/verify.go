package policy

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/netip"
	"strings"
	"time"
)

// Decide evaluates whether ip can access host against the live cache.
// It does not notify observers and does not perform bearer-token verification.
// Safe for concurrent use.
func (s *Service) Decide(ctx context.Context, ip, host string) DecisionResult {
	entry, ok := s.lookupIP(ctx, ip)
	if !ok {
		return DecisionResult{DenyReason: new(DenyReasonIPNotRegistered)}
	}
	contributors := toIPContributors(entry.Contributors)
	if entry.BypassAllowlist {
		return DecisionResult{Allowed: true, Contributors: contributors}
	}
	h := strings.ToLower(host)
	if _, ok := entry.AllowedHosts[h]; !ok {
		return DecisionResult{DenyReason: new(DenyReasonHostNotAllowed), Contributors: contributors}
	}
	return DecisionResult{Allowed: true, Contributors: contributors}
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

	host := ""
	if req.TargetHost != nil {
		host = *req.TargetHost
	}

	result := s.Decide(ctx, req.ClientIP, host)

	if !result.Allowed {
		if result.DenyReason != nil && *result.DenyReason == DenyReasonIPNotRegistered {
			s.logger.DebugContext(ctx, "IP not enabled")
			s.notifyDecisionObservers(ctx, NewDecisionEvent(false, result.DenyReason, nil, req, geo, time.Since(start).Microseconds()))
			return ErrIPNotEnabled
		}
		s.logger.DebugContext(ctx, "host not in allowlist", slog.String("host", strings.ToLower(host)))
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, result.DenyReason, result.Contributors, req, geo, time.Since(start).Microseconds()))
		return ErrHostNotAllowed
	}

	s.logger.DebugContext(ctx, "IP is enabled")
	s.notifyDecisionObservers(ctx, NewDecisionEvent(true, nil, result.Contributors, req, geo, time.Since(start).Microseconds()))

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
