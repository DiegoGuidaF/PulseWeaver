package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/jmoiron/sqlx"
)

// NetworkPolicySummaryView is the read model for the policy list page.
type NetworkPolicySummaryView struct {
	ID                 networkpolicies.NetworkPolicyID
	Name               string
	CIDR               string
	Description        *string
	Enabled            bool
	AllowAllHosts      bool
	CreatedAt          time.Time
	EffectiveHostCount int
	TotalHostCount     int
}

// PolicyHostGroupView is a host group annotated with its assignment state and full host list.
type PolicyHostGroupView struct {
	ID       int64
	Name     string
	Color    *string
	Icon     *string
	Hosts    []PolicyHostRefView
	Assigned bool
}

// PolicyHostRefView is a host reference used inside a group (id, fqdn, icon).
type PolicyHostRefView struct {
	ID   int64
	FQDN string
	Icon *string
}

// PolicyHostView is a known host annotated with its assignment state.
type PolicyHostView struct {
	ID       int64
	FQDN     string
	Icon     *string
	Assigned bool
	ViaGroup bool
}

// NetworkPolicyDetailView is the full detail read model for a single policy.
type NetworkPolicyDetailView struct {
	ID                 networkpolicies.NetworkPolicyID
	Name               string
	CIDR               string
	Description        *string
	Enabled            bool
	AllowAllHosts      bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	EffectiveHostCount int
	TotalHostCount     int
	HostGroups         []PolicyHostGroupView
	IndividualHosts    []PolicyHostView
}

// GetNetworkPolicySummaries returns all policies enriched with host count metadata.
func (r *Repository) GetNetworkPolicySummaries(ctx context.Context) ([]NetworkPolicySummaryView, error) {
	type policyRow struct {
		ID            networkpolicies.NetworkPolicyID `db:"id"`
		Name          string                          `db:"name"`
		CIDR          string                          `db:"cidr"`
		Description   *string                         `db:"description"`
		Enabled       bool                            `db:"enabled"`
		AllowAllHosts bool                            `db:"allow_all_hosts"`
		CreatedAt     time.Time                       `db:"created_at"`
	}

	const listQuery = `
		SELECT id, name, cidr, description, enabled, allow_all_hosts, created_at
		FROM network_policies
		ORDER BY created_at DESC
	`
	var rows []policyRow
	if err := r.db.SelectContext(ctx, &rows, listQuery); err != nil {
		return nil, fmt.Errorf("list network policies: %w", err)
	}
	if len(rows) == 0 {
		return []NetworkPolicySummaryView{}, nil
	}

	totalHostCount, err := r.totalKnownHostCount(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]any, len(rows))
	for i, p := range rows {
		ids[i] = p.ID
	}

	effectiveQuery, args, err := sqlx.In(`
		SELECT policy_id, COUNT(DISTINCT known_host_id) AS effective_host_count
		FROM (
			SELECT policy_id, known_host_id FROM network_policy_allowed_hosts WHERE policy_id IN (?)
			UNION
			SELECT nphg.policy_id, hgm.known_host_id
			FROM network_policy_allowed_host_groups nphg
			JOIN host_group_members hgm ON hgm.host_group_id = nphg.host_group_id
			WHERE nphg.policy_id IN (?)
		) combined
		GROUP BY policy_id
	`, ids, ids)
	if err != nil {
		return nil, fmt.Errorf("build effective count query: %w", err)
	}
	effectiveQuery = r.db.Rebind(effectiveQuery)

	type countRow struct {
		PolicyID           networkpolicies.NetworkPolicyID `db:"policy_id"`
		EffectiveHostCount int                             `db:"effective_host_count"`
	}
	var countRows []countRow
	if err := r.db.SelectContext(ctx, &countRows, effectiveQuery, args...); err != nil {
		return nil, fmt.Errorf("count effective hosts: %w", err)
	}

	countByID := make(map[networkpolicies.NetworkPolicyID]int, len(countRows))
	for _, cr := range countRows {
		countByID[cr.PolicyID] = cr.EffectiveHostCount
	}

	summaries := make([]NetworkPolicySummaryView, len(rows))
	for i, p := range rows {
		effective := countByID[p.ID]
		if p.AllowAllHosts {
			effective = totalHostCount
		}
		summaries[i] = NetworkPolicySummaryView{
			ID:                 p.ID,
			Name:               p.Name,
			CIDR:               p.CIDR,
			Description:        p.Description,
			Enabled:            p.Enabled,
			AllowAllHosts:      p.AllowAllHosts,
			CreatedAt:          p.CreatedAt,
			EffectiveHostCount: effective,
			TotalHostCount:     totalHostCount,
		}
	}
	return summaries, nil
}

// GetNetworkPolicyDetail returns the full detail view for one policy, including all host
// groups (with their full member lists) and all individual hosts annotated with assignment state.
func (r *Repository) GetNetworkPolicyDetail(ctx context.Context, id networkpolicies.NetworkPolicyID) (*NetworkPolicyDetailView, error) {
	type policyRow struct {
		ID            networkpolicies.NetworkPolicyID `db:"id"`
		Name          string                          `db:"name"`
		CIDR          string                          `db:"cidr"`
		Description   *string                         `db:"description"`
		Enabled       bool                            `db:"enabled"`
		AllowAllHosts bool                            `db:"allow_all_hosts"`
		CreatedAt     time.Time                       `db:"created_at"`
		UpdatedAt     time.Time                       `db:"updated_at"`
	}

	var p policyRow
	if err := r.db.GetContext(ctx, &p, `
		SELECT id, name, cidr, description, enabled, allow_all_hosts, created_at, updated_at
		FROM network_policies WHERE id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, networkpolicies.ErrNotFound
		}
		return nil, fmt.Errorf("get network policy: %w", err)
	}

	totalHostCount, err := r.totalKnownHostCount(ctx)
	if err != nil {
		return nil, err
	}

	effectiveHostCount, err := r.effectiveHostCount(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.AllowAllHosts {
		effectiveHostCount = totalHostCount
	}

	groups, err := r.listGroupsForPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	hosts, err := r.listHostsForPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	return &NetworkPolicyDetailView{
		ID:                 p.ID,
		Name:               p.Name,
		CIDR:               p.CIDR,
		Description:        p.Description,
		Enabled:            p.Enabled,
		AllowAllHosts:      p.AllowAllHosts,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		EffectiveHostCount: effectiveHostCount,
		TotalHostCount:     totalHostCount,
		HostGroups:         groups,
		IndividualHosts:    hosts,
	}, nil
}

// ── private helpers ────────────────────────────────────────────────────────────

func (r *Repository) totalKnownHostCount(ctx context.Context) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM known_hosts`); err != nil {
		return 0, fmt.Errorf("count known hosts: %w", err)
	}
	return count, nil
}

func (r *Repository) effectiveHostCount(ctx context.Context, id networkpolicies.NetworkPolicyID) (int, error) {
	const query = `
		SELECT COUNT(DISTINCT known_host_id)
		FROM (
			SELECT known_host_id FROM network_policy_allowed_hosts WHERE policy_id = ?
			UNION
			SELECT hgm.known_host_id
			FROM network_policy_allowed_host_groups nphg
			JOIN host_group_members hgm ON hgm.host_group_id = nphg.host_group_id
			WHERE nphg.policy_id = ?
		) combined
	`
	var count int
	if err := r.db.GetContext(ctx, &count, query, id, id); err != nil {
		return 0, fmt.Errorf("count effective hosts: %w", err)
	}
	return count, nil
}

func (r *Repository) listGroupsForPolicy(ctx context.Context, id networkpolicies.NetworkPolicyID) ([]PolicyHostGroupView, error) {
	type dbGroupRow struct {
		ID       int64   `db:"id"`
		Name     string  `db:"name"`
		Color    *string `db:"color"`
		Icon     *string `db:"icon"`
		Assigned bool    `db:"assigned"`
	}

	const groupQuery = `
		SELECT hg.id, hg.name, hg.color, hg.icon,
		       (npahg.policy_id IS NOT NULL) AS assigned
		FROM host_groups hg
		LEFT JOIN network_policy_allowed_host_groups npahg
		    ON npahg.host_group_id = hg.id AND npahg.policy_id = ?
		ORDER BY assigned DESC, hg.name ASC
	`
	var groupRows []dbGroupRow
	if err := r.db.SelectContext(ctx, &groupRows, groupQuery, id); err != nil {
		return nil, fmt.Errorf("list groups for policy: %w", err)
	}
	if len(groupRows) == 0 {
		return []PolicyHostGroupView{}, nil
	}

	groupIDs := make([]any, len(groupRows))
	for i, g := range groupRows {
		groupIDs[i] = g.ID
	}

	// Fetch full host list for all groups in one query.
	hostQuery, args, err := sqlx.In(`
		SELECT hgm.host_group_id, kh.id AS host_id, kh.fqdn, kh.icon
		FROM host_group_members hgm
		JOIN known_hosts kh ON kh.id = hgm.known_host_id
		WHERE hgm.host_group_id IN (?)
		ORDER BY hgm.host_group_id, kh.fqdn
	`, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("build group hosts query: %w", err)
	}
	hostQuery = r.db.Rebind(hostQuery)

	type hostRow struct {
		GroupID int64   `db:"host_group_id"`
		HostID  int64   `db:"host_id"`
		FQDN    string  `db:"fqdn"`
		Icon    *string `db:"icon"`
	}
	var hostRows []hostRow
	if err := r.db.SelectContext(ctx, &hostRows, hostQuery, args...); err != nil {
		return nil, fmt.Errorf("fetch group hosts: %w", err)
	}

	hostsByGroup := make(map[int64][]PolicyHostRefView, len(groupRows))
	for _, h := range hostRows {
		hostsByGroup[h.GroupID] = append(hostsByGroup[h.GroupID], PolicyHostRefView{
			ID:   h.HostID,
			FQDN: h.FQDN,
			Icon: h.Icon,
		})
	}

	groups := make([]PolicyHostGroupView, len(groupRows))
	for i, g := range groupRows {
		hosts := hostsByGroup[g.ID]
		if hosts == nil {
			hosts = []PolicyHostRefView{}
		}
		groups[i] = PolicyHostGroupView{
			ID:       g.ID,
			Name:     g.Name,
			Color:    g.Color,
			Icon:     g.Icon,
			Hosts:    hosts,
			Assigned: g.Assigned,
		}
	}
	return groups, nil
}

func (r *Repository) listHostsForPolicy(ctx context.Context, id networkpolicies.NetworkPolicyID) ([]PolicyHostView, error) {
	const query = `
		SELECT
			kh.id,
			kh.fqdn,
			kh.icon,
			(npah.policy_id IS NOT NULL) AS assigned,
			EXISTS(
				SELECT 1 FROM host_group_members hgm
				JOIN network_policy_allowed_host_groups npahg
				    ON npahg.host_group_id = hgm.host_group_id
				WHERE hgm.known_host_id = kh.id AND npahg.policy_id = ?
			) AS via_group
		FROM known_hosts kh
		LEFT JOIN network_policy_allowed_hosts npah
		    ON npah.known_host_id = kh.id AND npah.policy_id = ?
		ORDER BY assigned DESC, via_group DESC, kh.fqdn ASC
	`
	type dbHostRow struct {
		ID       int64   `db:"id"`
		FQDN     string  `db:"fqdn"`
		Icon     *string `db:"icon"`
		Assigned bool    `db:"assigned"`
		ViaGroup bool    `db:"via_group"`
	}
	var rows []dbHostRow
	if err := r.db.SelectContext(ctx, &rows, query, id, id); err != nil {
		return nil, fmt.Errorf("list hosts for policy: %w", err)
	}

	hosts := make([]PolicyHostView, len(rows))
	for i, row := range rows {
		hosts[i] = PolicyHostView(row)
	}
	return hosts, nil
}
