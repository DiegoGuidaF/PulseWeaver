package queries

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/collate"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

func (r *Repository) GetAllHostsWithGroups(ctx context.Context) (httpapi.HostListResponse, error) {
	type row struct {
		ID         ids.HostID       `db:"id"`
		FQDN       string           `db:"fqdn"`
		CreatedAt  time.Time        `db:"created_at"`
		GroupID    *ids.HostGroupID `db:"group_id"`
		GroupName  *string          `db:"group_name"`
		GroupColor *string          `db:"group_color"`
	}
	const query = `
		SELECT
			kh.id          AS id,
			kh.fqdn        AS fqdn,
			kh.created_at  AS created_at,
			hg.id   AS group_id,
			hg.name AS group_name,
			hg.color AS group_color
		FROM hosts kh
		LEFT JOIN host_group_members hgm ON hgm.host_id = kh.id
		LEFT JOIN host_groups hg ON hg.id = hgm.host_group_id
		ORDER BY kh.fqdn, hg.name
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return httpapi.HostListResponse{}, fmt.Errorf("get hosts with groups: %w", err)
	}

	hostsList := collate.Collapse(rows,
		func(rw row) ids.HostID { return rw.ID },
		func(rw row) httpapi.Host {
			return httpapi.Host{
				Id:        rw.ID.Int64(),
				Fqdn:      rw.FQDN,
				CreatedAt: httpapi.UTCTime(rw.CreatedAt),
				Groups:    []httpapi.GroupSummary{},
			}
		},
		func(rw row) (httpapi.GroupSummary, bool) {
			if rw.GroupID == nil || rw.GroupName == nil {
				return httpapi.GroupSummary{}, false
			}
			return httpapi.GroupSummary{
				Id:    (*rw.GroupID).Int64(),
				Name:  *rw.GroupName,
				Color: *rw.GroupColor,
			}, true
		},
		func(h *httpapi.Host, g httpapi.GroupSummary) { h.Groups = append(h.Groups, g) },
	)
	return httpapi.HostListResponse{Hosts: hostsList}, nil
}

func (r *Repository) GetHostGroupsDetails(ctx context.Context) (httpapi.GroupListResponse, error) {
	// Retrieve groups and their hosts
	type groupRow struct {
		ID          ids.HostGroupID `db:"id"`
		Name        string          `db:"name"`
		Color       string          `db:"color"`
		Icon        string          `db:"icon"`
		Description *string         `db:"description"`
		CreatedAt   time.Time       `db:"created_at"`
		UpdatedAt   time.Time       `db:"updated_at"`
		HostID      *ids.HostID     `db:"host_id"`
		HostFQDN    *string         `db:"host_fqdn"`
	}

	const groupQuery = `
		SELECT hg.id, hg.name, hg.color, hg.description, hg.icon, hg.created_at, hg.updated_at,
		       hgm.host_id, h.fqdn AS host_fqdn
		FROM host_groups hg
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		LEFT JOIN hosts h ON h.id = hgm.host_id
		ORDER BY hg.name, h.fqdn
	`
	var groupRows []groupRow
	if err := r.db.SelectContext(ctx, &groupRows, groupQuery); err != nil {
		return httpapi.GroupListResponse{}, fmt.Errorf("get host groups with members: %w", err)
	}

	// Retrieve users of each group
	type userRow struct {
		GroupID     ids.HostGroupID `db:"host_group_id"`
		UserID      ids.UserID      `db:"user_id"`
		Username    string          `db:"username"`
		DisplayName string          `db:"display_name"`
	}
	const usersQuery = `
		SELECT uahg.host_group_id, u.id AS user_id, u.username, u.display_name
		FROM user_allowed_host_groups uahg
		JOIN users u ON u.id = uahg.user_id
		WHERE u.deleted_at IS NULL
		ORDER BY uahg.host_group_id, u.display_name
	`
	var userRows []userRow
	if err := r.db.SelectContext(ctx, &userRows, usersQuery); err != nil {
		return httpapi.GroupListResponse{}, fmt.Errorf("get host group users: %w", err)
	}

	usersByGroup := collate.GroupByMap(userRows,
		func(ur userRow) ids.HostGroupID { return ur.GroupID },
		func(ur userRow) httpapi.UserSummary {
			return httpapi.UserSummary{
				Id:          ur.UserID.Int64(),
				Username:    ur.Username,
				DisplayName: ur.DisplayName,
			}
		},
	)

	// Retrieve network_policies of each group
	type groupPolicyRow struct {
		GroupID    ids.HostGroupID     `db:"host_group_id"`
		PolicyID   ids.NetworkPolicyID `db:"policy_id"`
		PolicyName string              `db:"policy_name"`
		PolicyCIDR string              `db:"policy_cidr"`
	}
	const groupPoliciesQuery = `
		SELECT nphg.host_group_id, np.id AS policy_id, np.name AS policy_name, np.cidr AS policy_cidr
		FROM network_policy_allowed_host_groups nphg
		JOIN network_policies np ON np.id = nphg.policy_id
		ORDER BY nphg.host_group_id, np.name
	`
	var groupPolicyRows []groupPolicyRow
	if err := r.db.SelectContext(ctx, &groupPolicyRows, groupPoliciesQuery); err != nil {
		return httpapi.GroupListResponse{}, fmt.Errorf("get host group network policies: %w", err)
	}

	policiesByGroup := collate.GroupByMap(groupPolicyRows,
		func(pr groupPolicyRow) ids.HostGroupID { return pr.GroupID },
		func(pr groupPolicyRow) httpapi.NetworkPolicyRef {
			return httpapi.NetworkPolicyRef{
				Id:   pr.PolicyID.Int64(),
				Name: pr.PolicyName,
				Cidr: pr.PolicyCIDR,
			}
		},
	)

	groups := collate.Collapse(groupRows,
		func(rw groupRow) ids.HostGroupID { return rw.ID },
		func(rw groupRow) httpapi.GroupDetailWithUsers {
			return httpapi.GroupDetailWithUsers{
				Id:              rw.ID.Int64(),
				Name:            rw.Name,
				Color:           rw.Color,
				Description:     rw.Description,
				Icon:            rw.Icon,
				CreatedAt:       httpapi.UTCTime(rw.CreatedAt),
				UpdatedAt:       httpapi.UTCTime(rw.UpdatedAt),
				Hosts:           []httpapi.HostSummary{},
				Users:           new(collate.OrEmpty(usersByGroup[rw.ID])),
				NetworkPolicies: collate.OrEmpty(policiesByGroup[rw.ID]),
			}
		},
		func(rw groupRow) (httpapi.HostSummary, bool) {
			if rw.HostID == nil || rw.HostFQDN == nil {
				return httpapi.HostSummary{}, false
			}
			return httpapi.HostSummary{
				Id:   (*rw.HostID).Int64(),
				Fqdn: *rw.HostFQDN,
			}, true
		},
		func(g *httpapi.GroupDetailWithUsers, h httpapi.HostSummary) { g.Hosts = append(g.Hosts, h) },
	)
	return httpapi.GroupListResponse{
		Groups: groups,
	}, nil
}

const hostSuggestionsWindow = 7 * 24 * time.Hour

func (r *Repository) GetHostSuggestionsPage(ctx context.Context) (httpapi.HostSuggestionsPage, error) {
	rawIgnored, err := r.ignoredHostSuggestions(ctx)
	if err != nil {
		return httpapi.HostSuggestionsPage{}, err
	}
	knownHosts, err := r.knownHostSet(ctx)
	if err != nil {
		return httpapi.HostSuggestionsPage{}, err
	}

	ignored := make([]httpapi.IgnoredHostSuggestion, len(rawIgnored))
	ignoredSet := make(map[string]bool, len(rawIgnored))
	for i, s := range rawIgnored {
		ignored[i] = httpapi.IgnoredHostSuggestion{
			Id:        s.ID,
			Fqdn:      s.FQDN,
			CreatedAt: httpapi.UTCTime(s.CreatedAt),
		}
		ignoredSet[hosts.NormaliseHost(s.FQDN)] = true
	}

	type suggestionRow struct {
		FQDN        string          `db:"fqdn"`
		FirstSeen   database.DBTime `db:"first_seen"`
		AllowedHits int             `db:"allowed_hits"`
		DeniedHits  int             `db:"denied_hits"`
	}
	const suggestionsQuery = `
		SELECT
			LOWER(al.target_host) AS fqdn,
			MIN(al.created_at)    AS first_seen,
			SUM(CASE WHEN al.outcome = 1 THEN 1 ELSE 0 END) AS allowed_hits,
			SUM(CASE WHEN al.outcome = 0 THEN 1 ELSE 0 END) AS denied_hits
		FROM access_log al
		WHERE al.target_host IS NOT NULL
		  AND al.created_at >= ?
		GROUP BY LOWER(al.target_host)
	`
	since := time.Now().UTC().Add(-hostSuggestionsWindow)
	var rawSuggestions []suggestionRow
	if err := r.db.SelectContext(ctx, &rawSuggestions, suggestionsQuery, since); err != nil {
		return httpapi.HostSuggestionsPage{}, fmt.Errorf("get host suggestions: %w", err)
	}

	// The policy engine matches a requested host after stripping its port
	// (hosts.NormaliseHost), so suggestions aggregate on the same normalised key: a
	// service observed as app.example.com:8443 surfaces as a suggestion for
	// app.example.com, merged with any bare-host hits, and disappears once
	// app.example.com is granted or ignored. Already-known and ignored hosts are
	// excluded here rather than in SQL because the exclusion must compare the
	// normalised form, not the raw (possibly port-suffixed) target_host.
	type aggregate struct {
		firstSeen   time.Time
		allowedHits int
		deniedHits  int
	}
	merged := make(map[string]*aggregate, len(rawSuggestions))
	for _, s := range rawSuggestions {
		fqdn := hosts.NormaliseHost(s.FQDN)
		if hosts.ValidateFQDN(fqdn) != nil || knownHosts[fqdn] || ignoredSet[fqdn] {
			continue
		}
		a := merged[fqdn]
		if a == nil {
			a = &aggregate{firstSeen: s.FirstSeen.Time}
			merged[fqdn] = a
		}
		if s.FirstSeen.Before(a.firstSeen) {
			a.firstSeen = s.FirstSeen.Time
		}
		a.allowedHits += s.AllowedHits
		a.deniedHits += s.DeniedHits
	}

	suggestions := make([]httpapi.HostSuggestion, 0, len(merged))
	for fqdn, a := range merged {
		suggestions = append(suggestions, httpapi.HostSuggestion{
			Fqdn:        fqdn,
			FirstSeen:   httpapi.UTCTime(a.firstSeen),
			AllowedHits: a.allowedHits,
			DeniedHits:  a.deniedHits,
		})
	}
	slices.SortFunc(suggestions, func(a, b httpapi.HostSuggestion) int {
		if d := b.DeniedHits - a.DeniedHits; d != 0 {
			return d
		}
		if d := b.AllowedHits - a.AllowedHits; d != 0 {
			return d
		}
		return strings.Compare(a.Fqdn, b.Fqdn)
	})

	return httpapi.HostSuggestionsPage{Suggestions: suggestions, Ignored: ignored}, nil
}

// ignoredHostSuggestions returns the operator's ignored-host list, ordered by FQDN.
func (r *Repository) ignoredHostSuggestions(ctx context.Context) ([]hosts.IgnoredHostSuggestion, error) {
	const query = `
		SELECT id, fqdn, created_at
		FROM ignored_host_suggestions
		ORDER BY fqdn
	`
	var rows []hosts.IgnoredHostSuggestion
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get ignored suggestions: %w", err)
	}
	return rows, nil
}

// knownHostSet returns the registered host FQDNs keyed by their normalised form,
// for excluding already-granted hosts from suggestions.
func (r *Repository) knownHostSet(ctx context.Context) (map[string]bool, error) {
	const query = `SELECT fqdn FROM hosts`
	var fqdns []string
	if err := r.db.SelectContext(ctx, &fqdns, query); err != nil {
		return nil, fmt.Errorf("get known hosts: %w", err)
	}
	set := make(map[string]bool, len(fqdns))
	for _, f := range fqdns {
		set[hosts.NormaliseHost(f)] = true
	}
	return set, nil
}
