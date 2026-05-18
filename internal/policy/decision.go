package policy

import (
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// DecisionResult is returned by Decide. It carries enough information for
// VerifyAccess to notify observers and for the simulate handler to build its response.
type DecisionResult struct {
	Allowed           bool
	DenyReason        *DenyReason     // nil when Allowed
	Contributors      []IPContributor // nil when IP not in cache; populated for observer notification
	MatchSource       MatchSource
	NetworkPolicyID   *int64
	NetworkPolicyName *string
}

// UserHostAccess is the per-user projection consumed by refreshCache.
type UserHostAccess struct {
	UserID          ids.UserID
	BypassAllowlist bool
	AllowedHosts    []string // case-folded FQDNs
}

// GeoIPResolver resolves an IP to geographic and ASN data.
// Implementations must be safe for concurrent use and fail-open.
// A nil GeoIPResolver is valid — the service skips enrichment.
type GeoIPResolver interface {
	Resolve(ip string) geoip.Result
}

type ipSetEntry struct {
	Contributors        []ContributorAccess // all devices at this IP with pre-intersection state
	BypassAllowlist     bool
	AllowedHosts        map[string]struct{} // case-folded FQDNs; nil when all contributors bypass
	IntersectionApplied bool                // true when deny-wins trimmed at least one contributor's host set
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
