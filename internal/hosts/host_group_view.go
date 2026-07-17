package hosts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// GetHostGroupsList returns the light host-groups list for the group list page:
// display fields plus a member-host count, without the per-group users/hosts/
// network-policy detail (see GetHostGroupDetail for that).
func (r *Repository) GetHostGroupsList(ctx context.Context) (httpapi.GroupListResponse, error) {
	type row struct {
		ID        ids.HostGroupID `db:"id"`
		Name      string          `db:"name"`
		Color     string          `db:"color"`
		Icon      string          `db:"icon"`
		HostCount int             `db:"host_count"`
	}
	const query = `
		SELECT hg.id, hg.name, hg.color, hg.icon,
		       COUNT(hgm.host_id) AS host_count
		FROM host_groups hg
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		GROUP BY hg.id
		ORDER BY hg.name
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return httpapi.GroupListResponse{}, fmt.Errorf("list host groups: %w", err)
	}

	groups := make([]httpapi.GroupListItem, len(rows))
	for i, rw := range rows {
		groups[i] = httpapi.GroupListItem{
			Id:        rw.ID.Int64(),
			Name:      rw.Name,
			Color:     rw.Color,
			Icon:      rw.Icon,
			HostCount: rw.HostCount,
		}
	}
	return httpapi.GroupListResponse{Groups: groups}, nil
}

// GetHostGroupDetail returns the full detail for one host group: its member
// hosts, the users granted access to it, and the network policies that grant
// access through it. Returns ErrHostGroupNotFound if id does not exist.
func (r *Repository) GetHostGroupDetail(ctx context.Context, id ids.HostGroupID) (*httpapi.GroupDetailWithUsers, error) {
	type groupRow struct {
		ID          ids.HostGroupID `db:"id"`
		Name        string          `db:"name"`
		Color       string          `db:"color"`
		Icon        string          `db:"icon"`
		Description *string         `db:"description"`
	}
	var g groupRow
	if err := r.db.GetContext(ctx, &g,
		`SELECT id, name, color, icon, description FROM host_groups WHERE id = ?`, id,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrHostGroupNotFound
		}
		return nil, fmt.Errorf("get host group: %w", err)
	}

	type hostRow struct {
		ID   ids.HostID `db:"id"`
		FQDN string     `db:"fqdn"`
	}
	var hostRows []hostRow
	const hostsQuery = `
		SELECT h.id, h.fqdn
		FROM host_group_members hgm
		JOIN hosts h ON h.id = hgm.host_id
		WHERE hgm.host_group_id = ?
		ORDER BY h.fqdn
	`
	if err := r.db.SelectContext(ctx, &hostRows, hostsQuery, id); err != nil {
		return nil, fmt.Errorf("list host group hosts: %w", err)
	}
	hostSummaries := make([]httpapi.HostSummary, len(hostRows))
	for i, hr := range hostRows {
		hostSummaries[i] = httpapi.HostSummary{Id: hr.ID.Int64(), Fqdn: hr.FQDN}
	}

	type userRow struct {
		UserID      ids.UserID `db:"user_id"`
		Username    string     `db:"username"`
		DisplayName string     `db:"display_name"`
	}
	var userRows []userRow
	const usersQuery = `
		SELECT u.id AS user_id, u.username, u.display_name
		FROM user_allowed_host_groups uahg
		JOIN users u ON u.id = uahg.user_id
		WHERE uahg.host_group_id = ? AND u.deleted_at IS NULL
		ORDER BY u.display_name
	`
	if err := r.db.SelectContext(ctx, &userRows, usersQuery, id); err != nil {
		return nil, fmt.Errorf("list host group users: %w", err)
	}
	users := make([]httpapi.UserSummary, len(userRows))
	for i, ur := range userRows {
		users[i] = httpapi.UserSummary{Id: ur.UserID.Int64(), Username: ur.Username, DisplayName: ur.DisplayName}
	}

	type policyRow struct {
		PolicyID   ids.NetworkPolicyID `db:"policy_id"`
		PolicyName string              `db:"policy_name"`
		PolicyCIDR string              `db:"policy_cidr"`
	}
	var policyRows []policyRow
	const policiesQuery = `
		SELECT np.id AS policy_id, np.name AS policy_name, np.cidr AS policy_cidr
		FROM network_policy_allowed_host_groups nphg
		JOIN network_policies np ON np.id = nphg.policy_id
		WHERE nphg.host_group_id = ?
		ORDER BY np.name
	`
	if err := r.db.SelectContext(ctx, &policyRows, policiesQuery, id); err != nil {
		return nil, fmt.Errorf("list host group network policies: %w", err)
	}
	policies := make([]httpapi.NetworkPolicyRef, len(policyRows))
	for i, pr := range policyRows {
		policies[i] = httpapi.NetworkPolicyRef{Id: pr.PolicyID.Int64(), Name: pr.PolicyName, Cidr: pr.PolicyCIDR}
	}

	return &httpapi.GroupDetailWithUsers{
		Id:              g.ID.Int64(),
		Name:            g.Name,
		Color:           g.Color,
		Icon:            g.Icon,
		Description:     g.Description,
		Hosts:           hostSummaries,
		Users:           users,
		NetworkPolicies: policies,
	}, nil
}
