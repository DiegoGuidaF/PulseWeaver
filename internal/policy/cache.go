package policy

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// refreshCache queries enabled IPs and host access grants, then atomically
// replaces the in-memory set with deny-wins intersection for shared IPs.
func (s *Service) refreshCache(ctx context.Context) error {
	start := time.Now()

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
	for _, ua := range hostAccess {
		accessByUser[ua.UserID] = ua

		hosts := make(map[string]struct{}, len(ua.AllowedHosts))
		for _, h := range ua.AllowedHosts {
			hosts[h] = struct{}{}
		}
		hostSetByUser[ua.UserID] = hosts
	}

	type accumulator struct {
		contributors        []ContributorAccess
		allBypass           bool
		hasRestrictedUser   bool
		allowedHosts        map[string]struct{}
		firstRestrictedSize int // size of first restricted user's host set; for IntersectionApplied
	}

	byIP := make(map[string]*accumulator, len(ipEntries))

	for _, e := range ipEntries {
		acc := byIP[e.IP]
		if acc == nil {
			acc = &accumulator{allBypass: true}
			byIP[e.IP] = acc
		}

		ua := accessByUser[e.UserID]

		userAllowedHosts := sortedKeys(hostSetByUser[e.UserID])
		acc.contributors = append(acc.contributors, ContributorAccess{
			DeviceID:         e.DeviceID,
			AddressID:        e.AddressID,
			UserID:           e.UserID,
			UserBypass:       ua.BypassAllowlist,
			UserAllowedHosts: userAllowedHosts,
		})

		acc.allBypass = acc.allBypass && ua.BypassAllowlist

		if ua.BypassAllowlist {
			continue // bypass users are intersection-neutral
		}

		userHosts := hostSetByUser[e.UserID]
		if !acc.hasRestrictedUser {
			acc.allowedHosts = cloneHostSet(userHosts)
			acc.firstRestrictedSize = len(acc.allowedHosts)
			acc.hasRestrictedUser = true
			continue
		}

		intersectHostSets(acc.allowedHosts, userHosts)
	}

	newSet := make(map[string]ipSetEntry, len(byIP))
	for ip, acc := range byIP {
		newSet[ip] = ipSetEntry{
			Contributors:        acc.contributors,
			BypassAllowlist:     acc.allBypass,
			AllowedHosts:        acc.allowedHosts,
			IntersectionApplied: acc.hasRestrictedUser && len(acc.allowedHosts) < acc.firstRestrictedSize,
		}
	}

	s.mu.Lock()
	s.ipSet = newSet
	s.lastRefreshedAt = time.Now().UTC()
	s.lastRefreshDurationMs = time.Since(start).Milliseconds()
	s.mu.Unlock()

	s.logger.DebugContext(ctx, "policy IP cache refreshed", slog.Int(logging.AttrKeyIPCount, len(newSet)))
	return nil
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

// intersectHostSets removes elements from dst that are not present in src.
func intersectHostSets(dst, src map[string]struct{}) {
	for h := range dst {
		if _, ok := src[h]; !ok {
			delete(dst, h)
		}
	}
}
