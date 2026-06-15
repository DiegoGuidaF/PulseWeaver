package queries

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

// reducePosture folds a policy user map audit into the dashboard posture counts.
// It is a pure function over the audit DTO plus the live suggestion count, so the
// posture answer cannot drift from the audit page: both come from the same
// BuildPolicyUserMap result.
func reducePosture(audit httpapi.PolicyUserMapAudit, pendingSuggestionCount int) httpapi.DashboardPosture {
	var users httpapi.DashboardPostureUsers
	for _, u := range audit.Users {
		switch u.Status {
		case httpapi.Bypass:
			users.Bypass++
		case httpapi.LiveWithAccess:
			users.LiveWithAccess++
		case httpapi.LiveNoHostAccess:
			users.LiveNoHostAccess++
		case httpapi.NoLiveIps:
			users.NoLiveIps++
		case httpapi.NoAccess:
			users.NoAccess++
		}
	}

	bypassHostCheck := 0
	for _, np := range audit.NetworkPolicies {
		if np.BypassHostCheck {
			bypassHostCheck++
		}
	}

	return httpapi.DashboardPosture{
		RefreshedAt: audit.RefreshedAt,
		Users:       users,
		NetworkPolicies: httpapi.DashboardPostureNetworkPolicies{
			Enabled:         audit.TotalNetworkPolicyCount,
			BypassHostCheck: bypassHostCheck,
		},
		SharedIpCount:          audit.SharedIpCount,
		KnownHostCount:         audit.TotalHostCount,
		PendingSuggestionCount: pendingSuggestionCount,
	}
}

// BuildDashboardPosture reduces the policy cache snapshot to the dashboard
// posture counts and bundles the live pending-suggestion count. The cache-derived
// counts reuse BuildPolicyUserMap rather than re-aggregating, so they stay
// consistent with the policy audit page.
func (r *Repository) BuildDashboardPosture(
	ctx context.Context,
	reader PolicyMapReader,
	npProvider AuditNetworkPoliciesProvider,
) (httpapi.DashboardPosture, error) {
	// Posture is count-only; geo enrichment is not needed here.
	audit, err := r.BuildPolicyUserMap(ctx, reader, npProvider, nil)
	if err != nil {
		return httpapi.DashboardPosture{}, fmt.Errorf("build policy user map: %w", err)
	}

	pending, err := r.countPendingHostSuggestions(ctx)
	if err != nil {
		return httpapi.DashboardPosture{}, fmt.Errorf("count pending host suggestions: %w", err)
	}

	return reducePosture(audit, pending), nil
}

// countPendingHostSuggestions counts unknown hosts receiving real traffic that are
// neither registered nor ignored. It mirrors GetHostSuggestionsPage's pending set —
// same filters, same FQDN validation — so the dashboard count matches the list the
// suggestions page renders.
func (r *Repository) countPendingHostSuggestions(ctx context.Context) (int, error) {
	const query = `
		SELECT DISTINCT LOWER(al.target_host) AS fqdn
		FROM access_log al
		WHERE al.target_host IS NOT NULL
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM hosts)
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM ignored_host_suggestions)
	`
	var fqdns []string
	if err := r.db.SelectContext(ctx, &fqdns, query); err != nil {
		return 0, fmt.Errorf("select pending host suggestions: %w", err)
	}

	count := 0
	for _, fqdn := range fqdns {
		if hosts.ValidateFQDN(fqdn) == nil {
			count++
		}
	}
	return count, nil
}
