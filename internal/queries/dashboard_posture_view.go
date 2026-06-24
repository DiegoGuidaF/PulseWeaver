package queries

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// EnabledIPProvider yields the enabled IP entries the policy cache is built from.
// Posture derives its live-IP counts from this same source so they reflect what
// the cache would materialize, without taking the cache lock.
type EnabledIPProvider interface {
	GetEnabledIPEntries(ctx context.Context) ([]device.IPEntry, error)
}

// postureUserRow carries one user's classification inputs. Live-IP presence is not
// here — it comes from the enabled IP entries, the same source the cache uses.
type postureUserRow struct {
	UserID    ids.UserID `db:"user_id"`
	Bypass    bool       `db:"bypass"`
	HasGrants bool       `db:"has_grants"`
}

// BuildDashboardPosture assembles the current-state posture counts entirely from
// the database — the same tables the policy cache is built from — so the counts
// reflect source-of-truth state without depending on the in-memory cache.
func (r *Repository) BuildDashboardPosture(ctx context.Context, ipProvider EnabledIPProvider) (httpapi.DashboardPosture, error) {
	ipEntries, err := ipProvider.GetEnabledIPEntries(ctx)
	if err != nil {
		return httpapi.DashboardPosture{}, fmt.Errorf("enabled ip entries: %w", err)
	}
	liveIPUsers, sharedIPCount := summarizeLiveIPs(ipEntries)

	userRows, err := r.postureUserRows(ctx)
	if err != nil {
		return httpapi.DashboardPosture{}, err
	}

	knownHostCount, err := r.distinctGrantedHostCount(ctx)
	if err != nil {
		return httpapi.DashboardPosture{}, err
	}

	npEnabled, npBypass, err := r.enabledNetworkPolicyCounts(ctx)
	if err != nil {
		return httpapi.DashboardPosture{}, err
	}

	pending, err := r.countPendingHostSuggestions(ctx)
	if err != nil {
		return httpapi.DashboardPosture{}, fmt.Errorf("count pending host suggestions: %w", err)
	}

	return httpapi.DashboardPosture{
		Users: foldUserStatuses(userRows, liveIPUsers),
		NetworkPolicies: httpapi.DashboardPostureNetworkPolicies{
			Enabled:         npEnabled,
			BypassHostCheck: npBypass,
		},
		SharedIpCount:          sharedIPCount,
		KnownHostCount:         knownHostCount,
		PendingSuggestionCount: pending,
	}, nil
}

// summarizeLiveIPs reduces the enabled IP entries to the set of users with at least
// one live IP and the count of IPs shared by two or more distinct users. IPs are
// canonicalized with Unmap so an IPv4-mapped IPv6 address and its plain twin collapse
// onto one key, and unparseable IPs are skipped — both matching how buildIPSet groups
// the same entries when the cache is built.
func summarizeLiveIPs(entries []device.IPEntry) (map[ids.UserID]struct{}, int) {
	liveUsers := make(map[ids.UserID]struct{})
	usersByIP := make(map[netip.Addr]map[ids.UserID]struct{})

	for _, e := range entries {
		addr, err := netip.ParseAddr(e.IP)
		if err != nil {
			continue
		}
		addr = addr.Unmap()

		liveUsers[e.UserID] = struct{}{}

		set := usersByIP[addr]
		if set == nil {
			set = make(map[ids.UserID]struct{})
			usersByIP[addr] = set
		}
		set[e.UserID] = struct{}{}
	}

	shared := 0
	for _, set := range usersByIP {
		if len(set) >= 2 {
			shared++
		}
	}
	return liveUsers, shared
}

// foldUserStatuses classifies each user via the shared deriveUserStatus and tallies
// the posture histogram. It reuses the exact classification the policy audit applies,
// so the two views cannot disagree on what a status means.
func foldUserStatuses(users []postureUserRow, liveIPUsers map[ids.UserID]struct{}) httpapi.DashboardPostureUsers {
	var out httpapi.DashboardPostureUsers
	for _, u := range users {
		var ipCount, allowedHostCount int
		if _, ok := liveIPUsers[u.UserID]; ok {
			ipCount = 1
		}
		if u.HasGrants {
			allowedHostCount = 1
		}
		switch deriveUserStatus(u.Bypass, ipCount, allowedHostCount) {
		case httpapi.Bypass:
			out.Bypass++
		case httpapi.LiveWithAccess:
			out.LiveWithAccess++
		case httpapi.LiveNoHostAccess:
			out.LiveNoHostAccess++
		case httpapi.NoLiveIps:
			out.NoLiveIps++
		case httpapi.NoAccess:
			out.NoAccess++
		}
	}
	return out
}

// postureUserRows returns every non-deleted user with their bypass flag and whether
// they hold any host grant. has_grants mirrors allowedHostCount > 0 in the audit:
// it is true when the user reaches at least one host through a group grant.
func (r *Repository) postureUserRows(ctx context.Context) ([]postureUserRow, error) {
	const query = `
		SELECT
			u.id AS user_id,
			COALESCE(uhs.bypass_host_check, 0) AS bypass,
			EXISTS (
				SELECT 1
				FROM user_allowed_host_groups uahg
				JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
				JOIN hosts h ON h.id = hgm.host_id
				WHERE uahg.user_id = u.id
			) AS has_grants
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
	`
	var rows []postureUserRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("posture user rows: %w", err)
	}
	return rows, nil
}

// distinctGrantedHostCount counts distinct hosts reachable through any non-deleted
// user's group grants — the union of all users' allowlists.
func (r *Repository) distinctGrantedHostCount(ctx context.Context) (int, error) {
	const query = `
		SELECT COUNT(DISTINCT h.fqdn)
		FROM user_allowed_host_groups uahg
		JOIN users u ON u.id = uahg.user_id AND u.deleted_at IS NULL
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN hosts h ON h.id = hgm.host_id
	`
	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, fmt.Errorf("distinct granted host count: %w", err)
	}
	return count, nil
}

// enabledNetworkPolicyCounts returns the number of enabled network policies and how
// many of those bypass the host check. The enabled predicate matches the cache
// provider (GetEnabledCacheEntries), so the counts agree with the policy audit.
func (r *Repository) enabledNetworkPolicyCounts(ctx context.Context) (enabled, bypass int, err error) {
	const query = `
		SELECT
			COUNT(*) AS enabled,
			COALESCE(SUM(bypass_host_check), 0) AS bypass
		FROM network_policies
		WHERE enabled = 1
	`
	var row struct {
		Enabled int `db:"enabled"`
		Bypass  int `db:"bypass"`
	}
	if err := r.db.GetContext(ctx, &row, query); err != nil {
		return 0, 0, fmt.Errorf("enabled network policy counts: %w", err)
	}
	return row.Enabled, row.Bypass, nil
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
		  AND al.created_at >= ?
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM hosts)
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM ignored_host_suggestions)
	`
	since := time.Now().UTC().Add(-hostSuggestionsWindow)
	var fqdns []string
	if err := r.db.SelectContext(ctx, &fqdns, query, since); err != nil {
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
