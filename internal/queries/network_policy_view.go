package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/jmoiron/sqlx"
)

// NetworkPolicySummaryView is the read model for the policy list page.
type NetworkPolicySummaryView struct {
	ID                 ids.NetworkPolicyID
	Name               string
	CIDR               string
	Description        *string
	Enabled            bool
	BypassHostCheck    bool
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

// PolicyHostRefView is a host reference used inside a group (id, fqdn).
type PolicyHostRefView struct {
	ID   int64
	FQDN string
}

// NetworkPolicyDetailView is the full detail read model for a single policy.
type NetworkPolicyDetailView struct {
	ID                 ids.NetworkPolicyID
	Name               string
	CIDR               string
	Description        *string
	Enabled            bool
	BypassHostCheck    bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	EffectiveHostCount int
	TotalHostCount     int
	HostGroups         []PolicyHostGroupView
}

// GetNetworkPolicySummaries returns all policies enriched with host count metadata.
func (r *Repository) GetNetworkPolicySummaries(ctx context.Context) ([]NetworkPolicySummaryView, error) {
	type policyRow struct {
		ID              ids.NetworkPolicyID `db:"id"`
		Name            string              `db:"name"`
		CIDR            string              `db:"cidr"`
		Description     *string             `db:"description"`
		Enabled         bool                `db:"enabled"`
		BypassHostCheck bool                `db:"bypass_host_check"`
		CreatedAt       time.Time           `db:"created_at"`
	}

	const listQuery = `
		SELECT id, name, cidr, description, enabled, bypass_host_check, created_at
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

	totalHostCount, err := r.totalHostsCount(ctx)
	if err != nil {
		return nil, err
	}

	policyIDs := make([]any, len(rows))
	for i, p := range rows {
		policyIDs[i] = p.ID
	}

	effectiveQuery, args, err := sqlx.In(`
		SELECT nphg.policy_id, COUNT(DISTINCT hgm.host_id) AS effective_host_count
		FROM network_policy_allowed_host_groups nphg
		JOIN host_group_members hgm ON hgm.host_group_id = nphg.host_group_id
		WHERE nphg.policy_id IN (?)
		GROUP BY nphg.policy_id
	`, policyIDs)
	if err != nil {
		return nil, fmt.Errorf("build effective count query: %w", err)
	}
	effectiveQuery = r.db.Rebind(effectiveQuery)

	type countRow struct {
		PolicyID           ids.NetworkPolicyID `db:"policy_id"`
		EffectiveHostCount int                 `db:"effective_host_count"`
	}
	var countRows []countRow
	if err := r.db.SelectContext(ctx, &countRows, effectiveQuery, args...); err != nil {
		return nil, fmt.Errorf("count effective hosts: %w", err)
	}

	countByID := make(map[ids.NetworkPolicyID]int, len(countRows))
	for _, cr := range countRows {
		countByID[cr.PolicyID] = cr.EffectiveHostCount
	}

	summaries := make([]NetworkPolicySummaryView, len(rows))
	for i, p := range rows {
		effective := countByID[p.ID]
		if p.BypassHostCheck {
			effective = totalHostCount
		}
		summaries[i] = NetworkPolicySummaryView{
			ID:                 p.ID,
			Name:               p.Name,
			CIDR:               p.CIDR,
			Description:        p.Description,
			Enabled:            p.Enabled,
			BypassHostCheck:    p.BypassHostCheck,
			CreatedAt:          p.CreatedAt,
			EffectiveHostCount: effective,
			TotalHostCount:     totalHostCount,
		}
	}
	return summaries, nil
}

// GetNetworkPolicyDetail returns the full detail view for one policy, including all host
// groups (with their full member lists) and all individual hosts annotated with assignment state.
func (r *Repository) GetNetworkPolicyDetail(ctx context.Context, id ids.NetworkPolicyID) (*NetworkPolicyDetailView, error) {
	type policyRow struct {
		ID              ids.NetworkPolicyID `db:"id"`
		Name            string              `db:"name"`
		CIDR            string              `db:"cidr"`
		Description     *string             `db:"description"`
		Enabled         bool                `db:"enabled"`
		BypassHostCheck bool                `db:"bypass_host_check"`
		CreatedAt       time.Time           `db:"created_at"`
		UpdatedAt       time.Time           `db:"updated_at"`
	}

	var p policyRow
	if err := r.db.GetContext(ctx, &p, `
		SELECT id, name, cidr, description, enabled, bypass_host_check, created_at, updated_at
		FROM network_policies WHERE id = ?`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, networkpolicies.ErrNotFound
		}
		return nil, fmt.Errorf("get network policy: %w", err)
	}

	totalHostCount, err := r.totalHostsCount(ctx)
	if err != nil {
		return nil, err
	}

	effectiveHostCount, err := r.effectiveHostCount(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.BypassHostCheck {
		effectiveHostCount = totalHostCount
	}

	groups, err := r.listGroupsForPolicy(ctx, id)
	if err != nil {
		return nil, err
	}

	return &NetworkPolicyDetailView{
		ID:                 p.ID,
		Name:               p.Name,
		CIDR:               p.CIDR,
		Description:        p.Description,
		Enabled:            p.Enabled,
		BypassHostCheck:    p.BypassHostCheck,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
		EffectiveHostCount: effectiveHostCount,
		TotalHostCount:     totalHostCount,
		HostGroups:         groups,
	}, nil
}

// ── private helpers ────────────────────────────────────────────────────────────

func (r *Repository) totalHostsCount(ctx context.Context) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM hosts`); err != nil {
		return 0, fmt.Errorf("count hosts: %w", err)
	}
	return count, nil
}

func (r *Repository) effectiveHostCount(ctx context.Context, id ids.NetworkPolicyID) (int, error) {
	const query = `
		SELECT COUNT(DISTINCT hgm.host_id)
		FROM network_policy_allowed_host_groups nphg
		JOIN host_group_members hgm ON hgm.host_group_id = nphg.host_group_id
		WHERE nphg.policy_id = ?
	`
	var count int
	if err := r.db.GetContext(ctx, &count, query, id); err != nil {
		return 0, fmt.Errorf("count effective hosts: %w", err)
	}
	return count, nil
}

func (r *Repository) listGroupsForPolicy(ctx context.Context, id ids.NetworkPolicyID) ([]PolicyHostGroupView, error) {
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
		SELECT hgm.host_group_id, h.id AS host_id, h.fqdn
		FROM host_group_members hgm
		JOIN hosts h ON h.id = hgm.host_id
		WHERE hgm.host_group_id IN (?)
		ORDER BY hgm.host_group_id, h.fqdn
	`, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("build group hosts query: %w", err)
	}
	hostQuery = r.db.Rebind(hostQuery)

	type hostRow struct {
		GroupID int64  `db:"host_group_id"`
		HostID  int64  `db:"host_id"`
		FQDN    string `db:"fqdn"`
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
