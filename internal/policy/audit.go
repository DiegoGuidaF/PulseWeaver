package policy

import (
	"time"
)

// PolicyMapEntry is the exported snapshot of a single ipSetEntry.
type PolicyMapEntry struct {
	IP                  string
	BypassAllowlist     bool
	AllowedHosts        []string // sorted lexicographically
	IntersectionApplied bool
	Contributors        []ContributorAccess
}

// PolicyMapSnapshot is a consistent point-in-time copy of the full cache.
type PolicyMapSnapshot struct {
	Entries               []PolicyMapEntry
	LastRefreshedAt       time.Time
	LastRefreshDurationMs int64
}

// GetPolicyMap returns a deep snapshot of the current IP set under RLock.
// Callers receive their own copy and cannot mutate the live cache.
func (s *Service) GetPolicyMap() PolicyMapSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]PolicyMapEntry, 0, len(s.ipSet))
	for ip, e := range s.ipSet {
		entries = append(entries, toPolicyMapEntry(ip, e))
	}

	return PolicyMapSnapshot{
		Entries:               entries,
		LastRefreshedAt:       s.lastRefreshedAt,
		LastRefreshDurationMs: s.lastRefreshDurationMs,
	}
}

// toPolicyMapEntry converts an internal ipSetEntry to its exported form.
func toPolicyMapEntry(ip string, e ipSetEntry) PolicyMapEntry {
	hosts := sortedKeys(e.AllowedHosts)

	contributors := make([]ContributorAccess, len(e.Contributors))
	copy(contributors, e.Contributors)

	return PolicyMapEntry{
		IP:                  ip,
		BypassAllowlist:     e.BypassAllowlist,
		AllowedHosts:        hosts,
		IntersectionApplied: e.IntersectionApplied,
		Contributors:        contributors,
	}
}
