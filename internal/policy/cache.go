package policy

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/netip"
	"slices"
	"sort"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
)

// networkPolicyCacheEntry holds a parsed CIDR prefix and its access config
// for fast in-loop CIDR containment checks.
type networkPolicyCacheEntry struct {
	PolicyID        int64
	PolicyName      string
	Prefix          netip.Prefix
	BypassHostCheck bool
	AllowedHosts    map[string]struct{}
}

// refreshCache queries all providers and atomically replaces both in-memory
// caches (IP set and network policies).
func (s *Service) refreshCache(ctx context.Context) error {
	start := time.Now()

	//TODO: ipEntries is a flat list (one row per device/address/user combination); consider
	// returning entries pre-grouped by IP so buildIPSet doesn't need the accumulator pattern.
	ipEntries, err := s.ipProvider.GetEnabledIPEntries(ctx)
	if err != nil {
		return fmt.Errorf("loading IP entries: %w", err)
	}

	var hostAccess []UserHostAccess
	if s.hostProvider != nil {
		hostAccess, err = s.hostProvider.GetAllUserHostAccess(ctx)
		if err != nil {
			return fmt.Errorf("loading host access grants: %w", err)
		}
	}

	newIPSet := buildIPSet(ipEntries, hostAccess)

	var networkPolicyEntries []networkpolicies.CacheEntry
	if s.networkPoliciesProvider != nil {
		networkPolicyEntries, err = s.networkPoliciesProvider.GetEnabledCacheEntries(ctx)
		if err != nil {
			return fmt.Errorf("loading network policy entries: %w", err)
		}
	}

	newNetworkPolicies := buildNetworkPolicyCache(ctx, networkPolicyEntries, s.logger)

	s.mu.Lock()
	s.ipSet = newIPSet
	s.networkPolicies = newNetworkPolicies
	s.lastRefreshedAt = time.Now().UTC()
	s.lastRefreshDurationMs = time.Since(start).Milliseconds()
	s.mu.Unlock()

	s.logger.DebugContext(ctx, "policy IP cache refreshed",
		slog.Int(logging.AttrKeyIPCount, len(newIPSet)),
		slog.Int("network_policy_count", len(newNetworkPolicies)))
	return nil
}

// buildIPSet joins IP entries with host-access grants and applies deny-wins
// intersection for IPs shared by multiple restricted users. Pure function;
// safe to call without holding any lock.
func buildIPSet(ipEntries []device.IPEntry, hostAccess []UserHostAccess) map[string]ipSetEntry {
	accessByUser := make(map[ids.UserID]UserHostAccess, len(hostAccess))
	hostSetByUser := make(map[ids.UserID]map[string]struct{}, len(hostAccess))
	for _, ua := range hostAccess {
		accessByUser[ua.UserID] = ua

		hosts := make(map[string]struct{}, len(ua.AllowedHosts))
		for _, h := range ua.AllowedHosts {
			hosts[h] = struct{}{}
		}
		hostSetByUser[ua.UserID] = hosts
	}

	type accumulator struct {
		contributors      []ContributorAccess
		allBypass         bool
		hasRestrictedUser bool
		allowedHosts      map[string]struct{}
		initialHostsLen   int // size of first restricted user's host set; used to detect whether intersection shrank it
	}

	byIP := make(map[string]*accumulator, len(ipEntries))

	for _, e := range ipEntries {
		acc := byIP[e.IP]
		if acc == nil {
			acc = &accumulator{allBypass: true}
			byIP[e.IP] = acc
		}

		ua := accessByUser[e.UserID]

		acc.contributors = append(acc.contributors, ContributorAccess{
			DeviceID:         e.DeviceID,
			AddressID:        e.AddressID,
			UserID:           e.UserID,
			UserBypass:       ua.BypassAllowlist,
			UserAllowedHosts: sortedKeys(hostSetByUser[e.UserID]),
		})

		acc.allBypass = acc.allBypass && ua.BypassAllowlist

		if ua.BypassAllowlist {
			continue
		}

		userHosts := hostSetByUser[e.UserID]
		if !acc.hasRestrictedUser {
			acc.allowedHosts = maps.Clone(userHosts)
			acc.initialHostsLen = len(acc.allowedHosts)
			acc.hasRestrictedUser = true
			continue
		}

		intersectHostSets(acc.allowedHosts, userHosts)
	}

	newIPSet := make(map[string]ipSetEntry, len(byIP))
	for ip, acc := range byIP {
		newIPSet[ip] = ipSetEntry{
			Contributors:        acc.contributors,
			BypassAllowlist:     acc.allBypass,
			AllowedHosts:        acc.allowedHosts,
			IntersectionApplied: acc.hasRestrictedUser && len(acc.allowedHosts) < acc.initialHostsLen,
		}
	}
	return newIPSet
}

// buildNetworkPolicyCache parses raw CIDR entries into a cache sorted most-specific-first,
// skipping entries whose CIDRs cannot be parsed.
func buildNetworkPolicyCache(ctx context.Context, entries []networkpolicies.CacheEntry, logger *slog.Logger) []networkPolicyCacheEntry {
	result := make([]networkPolicyCacheEntry, 0, len(entries))
	for _, e := range entries {
		prefix, err := netip.ParsePrefix(e.CIDR)
		if err != nil {
			logger.WarnContext(ctx, "skipping network policy with invalid CIDR",
				slog.String("cidr", e.CIDR),
				slog.Int64("policy_id", e.PolicyID.Int64()))
			continue
		}
		// Normalize the host bits regardless of what the provider stored.
		prefix = prefix.Masked()

		allowedHosts := make(map[string]struct{}, len(e.AllowedHostFQDNs))
		for _, fqdn := range e.AllowedHostFQDNs {
			allowedHosts[fqdn] = struct{}{}
		}

		result = append(result, networkPolicyCacheEntry{
			PolicyID:        e.PolicyID.Int64(),
			PolicyName:      e.PolicyName,
			Prefix:          prefix,
			BypassHostCheck: e.BypassHostCheck,
			AllowedHosts:    allowedHosts,
		})
	}

	slices.SortFunc(result, byMostSpecificFirst)

	return result
}

// sortedKeys returns a sorted slice of the map's keys; nil map returns empty slice.
func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// byMostSpecificFirst orders network policy entries longest-prefix-first so that
// more specific CIDRs are evaluated before broader ones (longest-prefix-match semantics).
func byMostSpecificFirst(a, b networkPolicyCacheEntry) int {
	return b.Prefix.Bits() - a.Prefix.Bits()
}

// intersectHostSets removes elements from dst that are not present in src.
func intersectHostSets(dst, src map[string]struct{}) {
	for h := range dst {
		if _, ok := src[h]; !ok {
			delete(dst, h)
		}
	}
}
