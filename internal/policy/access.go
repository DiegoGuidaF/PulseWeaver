package policy

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
)

// Decide evaluates whether ip can access host against the live cache.
// It does not notify observers and does not perform bearer-token verification.
// Safe for concurrent use.
//
// Matching order:
//  1. Exact IP match against device address set (device owner's host policy applies).
//  2. CIDR containment against enabled network policies (most-specific first).
//  3. Deny if neither matches.
func (s *Service) Decide(ctx context.Context, ip netip.Addr, host string) DecisionResult {
	// Fail closed on an invalid address (e.g. an unparseable IP at the boundary).
	if !ip.IsValid() {
		return DecisionResult{DenyReason: new(DenyReasonIPNotRegistered)}
	}
	// Defensive: callers pass canonical addresses, but unmapping here guarantees a
	// 4-in-6 address is evaluated identically to its plain IPv4 form (idempotent).
	ip = ip.Unmap()

	entry, ok := s.lookupIP(ctx, ip)
	if ok {
		contributors := toIPContributors(entry.Contributors)
		if entry.BypassAllowlist {
			return DecisionResult{Allowed: true, MatchSource: MatchSourceDevice, Contributors: contributors}
		}
		h := strings.ToLower(host)
		if _, ok := entry.AllowedHosts[h]; !ok {
			return DecisionResult{DenyReason: new(DenyReasonHostNotAllowed), MatchSource: MatchSourceDevice, Contributors: contributors}
		}
		return DecisionResult{Allowed: true, MatchSource: MatchSourceDevice, Contributors: contributors}
	}

	// CIDR fallback: check network policies in most-specific-first order.
	// Reject the trusted proxy IP before evaluating any CIDR policy, mirroring
	// the lookupIP guard. Otherwise a network policy whose prefix covers the
	// proxy's own address (e.g. when X-Real-IP is absent/malformed and the
	// request falls back to the proxy IP) would grant blanket access.
	if s.trustedProxy.IsValid() && s.trustedProxy.Compare(ip) == 0 {
		s.logger.WarnContext(ctx, "rejected trusted proxy IP authorization", slog.String(AttrKeyRequestIP, ip.String()))
		return DecisionResult{DenyReason: new(DenyReasonIPNotRegistered)}
	}

	s.mu.RLock()
	policies := s.networkPolicies
	s.mu.RUnlock()

	h := strings.ToLower(host)
	for _, np := range policies {
		if np.Prefix.Contains(ip) {
			_, hostAllowed := np.AllowedHosts[h]
			if np.BypassHostCheck || hostAllowed {
				return DecisionResult{
					Allowed:           true,
					MatchSource:       MatchSourceNetworkPolicy,
					NetworkPolicyID:   &np.PolicyID,
					NetworkPolicyName: &np.PolicyName,
				}
			}
			return DecisionResult{DenyReason: new(DenyReasonHostNotAllowed)}
		}
	}

	return DecisionResult{DenyReason: new(DenyReasonIPNotRegistered)}
}

// VerifyAccess validates the bearer token and verifies that the IP is enabled, emitting a DecisionEvent.
func (s *Service) VerifyAccess(ctx context.Context, req *VerifyRequest) error {
	s.logger.DebugContext(ctx, "Verify access for ip")
	start := time.Now()

	tokenHash := sha256.Sum256([]byte(req.Token))
	if subtle.ConstantTimeCompare(tokenHash[:], s.apiSecretHash[:]) != 1 {
		s.logger.WarnContext(ctx, "policy: invalid bearer token")
		// No geo enrichment for an unauthenticated request: the token check gates
		// the lookup so a bad-token flood can't drive a geo resolution per request.
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, new(DenyReasonInvalidToken), nil, req, geoip.Result{}, time.Since(start).Microseconds()))
		return ErrInvalidBearerToken
	}

	// A nil resolver is a valid configuration (enrichment disabled), so guard the
	// call rather than assuming a non-nil interface — see GeoIPResolver's contract.
	var geo geoip.Result
	if s.geoResolver != nil {
		geo = s.geoResolver.Resolve(req.ClientIP.String())
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
		s.notifyDecisionObservers(ctx, NewDecisionEvent(false, result.DenyReason, &result, req, geo, time.Since(start).Microseconds()))
		return ErrHostNotAllowed
	}

	s.logger.DebugContext(ctx, "IP is enabled")
	s.notifyDecisionObservers(ctx, NewDecisionEvent(true, nil, &result, req, geo, time.Since(start).Microseconds()))

	return nil
}

// lookupIP returns the ipSetEntry for ip if it is currently in the enabled set.
// It rejects the trusted proxy IP regardless of registration status. Thread-safe.
func (s *Service) lookupIP(ctx context.Context, ip netip.Addr) (ipSetEntry, bool) {
	if s.trustedProxy.IsValid() && s.trustedProxy.Compare(ip) == 0 {
		s.logger.WarnContext(ctx, "rejected trusted proxy IP authorization", slog.String(AttrKeyRequestIP, ip.String()))
		return ipSetEntry{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.ipSet[ip]
	s.logger.DebugContext(ctx, "found IP", slog.String(AttrKeyRequestIP, ip.String()))
	return entry, ok
}

// toIPContributors projects ContributorAccess down to IPContributor for observer notification.
func toIPContributors(cs []ContributorAccess) []IPContributor {
	if len(cs) == 0 {
		return nil
	}
	out := make([]IPContributor, len(cs))
	for i, c := range cs {
		out[i] = IPContributor{DeviceID: c.DeviceID, AddressID: c.AddressID, UserID: c.UserID}
	}
	return out
}
