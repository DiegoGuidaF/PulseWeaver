package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
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

	seen := make(map[ids.HostID]int)
	hosts := []httpapi.Host{}
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(hosts)
			seen[rw.ID] = idx
			hosts = append(hosts, httpapi.Host{
				Id:        rw.ID.Int64(),
				Fqdn:      rw.FQDN,
				CreatedAt: httpapi.UTCTime(rw.CreatedAt),
				Groups:    []httpapi.GroupSummary{},
			})
		}
		if rw.GroupID != nil && rw.GroupName != nil {
			hosts[idx].Groups = append(hosts[idx].Groups, httpapi.GroupSummary{
				Id:    (*rw.GroupID).Int64(),
				Name:  *rw.GroupName,
				Color: *rw.GroupColor,
			})
		}
	}
	return httpapi.HostListResponse{Hosts: hosts}, nil
}

func (r *Repository) GetHostGroupsDetails(ctx context.Context) (httpapi.GroupListResponse, error) {
	// Q1: groups with their member hosts
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

	// Q2: users granted access to each group
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

	usersByGroup := make(map[ids.HostGroupID][]httpapi.UserSummary)
	for _, ur := range userRows {
		usersByGroup[ur.GroupID] = append(usersByGroup[ur.GroupID], httpapi.UserSummary{
			Id:          ur.UserID.Int64(),
			Username:    ur.Username,
			DisplayName: ur.DisplayName,
		})
	}

	// Q3: network policies assigned to each group
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

	policiesByGroup := make(map[ids.HostGroupID][]httpapi.NetworkPolicyRef)
	for _, pr := range groupPolicyRows {
		policiesByGroup[pr.GroupID] = append(policiesByGroup[pr.GroupID], httpapi.NetworkPolicyRef{
			Id:   pr.PolicyID.Int64(),
			Name: pr.PolicyName,
			Cidr: pr.PolicyCIDR,
		})
	}

	seen := make(map[ids.HostGroupID]int)
	groups := []httpapi.GroupDetailWithUsers{}
	for _, rw := range groupRows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(groups)
			seen[rw.ID] = idx
			users := usersByGroup[rw.ID]
			if users == nil {
				users = []httpapi.UserSummary{}
			}
			policies := policiesByGroup[rw.ID]
			if policies == nil {
				policies = []httpapi.NetworkPolicyRef{}
			}
			groups = append(groups, httpapi.GroupDetailWithUsers{
				Id:              rw.ID.Int64(),
				Name:            rw.Name,
				Color:           rw.Color,
				Description:     rw.Description,
				Icon:            rw.Icon,
				CreatedAt:       httpapi.UTCTime(rw.CreatedAt),
				UpdatedAt:       httpapi.UTCTime(rw.UpdatedAt),
				Hosts:           []httpapi.HostSummary{},
				Users:           &users,
				NetworkPolicies: policies,
			})
		}
		if rw.HostID != nil && rw.HostFQDN != nil {
			groups[idx].Hosts = append(groups[idx].Hosts, httpapi.HostSummary{
				Id:   (*rw.HostID).Int64(),
				Fqdn: *rw.HostFQDN,
			})
		}
	}
	return httpapi.GroupListResponse{Groups: groups}, nil
}

func (r *Repository) GetHostSuggestionsPage(ctx context.Context) (httpapi.HostSuggestionsPage, error) {
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
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM hosts)
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM ignored_host_suggestions)
		GROUP BY LOWER(al.target_host)
		ORDER BY denied_hits DESC, allowed_hits DESC
	`
	var rawSuggestions []suggestionRow
	if err := r.db.SelectContext(ctx, &rawSuggestions, suggestionsQuery); err != nil {
		return httpapi.HostSuggestionsPage{}, fmt.Errorf("get host suggestions: %w", err)
	}

	suggestions := make([]httpapi.HostSuggestion, 0, len(rawSuggestions))
	for _, s := range rawSuggestions {
		if hosts.ValidateFQDN(s.FQDN) != nil {
			continue
		}
		suggestions = append(suggestions, httpapi.HostSuggestion{
			Fqdn:        s.FQDN,
			FirstSeen:   httpapi.UTCTime(s.FirstSeen.Time),
			AllowedHits: s.AllowedHits,
			DeniedHits:  s.DeniedHits,
		})
	}

	const ignoredQuery = `
		SELECT id, fqdn, created_at
		FROM ignored_host_suggestions
		ORDER BY fqdn
	`
	var rawIgnored []hosts.IgnoredHostSuggestion
	if err := r.db.SelectContext(ctx, &rawIgnored, ignoredQuery); err != nil {
		return httpapi.HostSuggestionsPage{}, fmt.Errorf("get ignored suggestions: %w", err)
	}

	ignored := make([]httpapi.IgnoredHostSuggestion, len(rawIgnored))
	for i, s := range rawIgnored {
		ignored[i] = httpapi.IgnoredHostSuggestion{
			Id:        s.ID,
			Fqdn:      s.FQDN,
			CreatedAt: httpapi.UTCTime(s.CreatedAt),
		}
	}

	return httpapi.HostSuggestionsPage{Suggestions: suggestions, Ignored: ignored}, nil
}

// ── User access table (list view) ─────────────────────────────────────────────

func (r *Repository) ListUserAccessRows(ctx context.Context) ([]httpapi.UserListItem, error) {
	type userRow struct {
		ID              ids.UserID `db:"id"`
		DisplayName     string     `db:"display_name"`
		UserName        string     `db:"username"`
		Email           string     `db:"email"`
		Role            auth.Role  `db:"role"`
		BypassHostCheck bool       `db:"bypass_host_check"`
		HostCount       int        `db:"host_count"`
		DeviceCount     int        `db:"device_count"`
		LiveIPCount     int        `db:"live_ip_count"`
	}
	const userQuery = `
		SELECT
			u.id,
			u.display_name,
			u.username,
			u.email,
			u.role,
			uhs.bypass_host_check as bypass_host_check,
			CASE WHEN COALESCE(uhs.bypass_host_check, 0) = 1 THEN
				(SELECT COUNT(*) FROM hosts)
			ELSE
				(
					SELECT COUNT(DISTINCT h.id)
					FROM hosts h
					WHERE h.id IN (
						SELECT hgm.host_id FROM host_group_members hgm
						JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hgm.host_group_id
						WHERE uahg.user_id = u.id
					)
				)
			END as host_count,
			(
				SELECT count(d.id) FROM devices d
				WHERE d.owner_id = u.id
				AND d.deleted_at is NULL
			) as device_count,
			(
				SELECT count(a.id) FROM addresses a
				JOIN devices d on a.device_id = d.id
				WHERE d.owner_id = u.id
				and a.device_id = d.id
			) as live_ip_count
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.display_name
	`
	var userRows []userRow
	if err := r.db.SelectContext(ctx, &userRows, userQuery); err != nil {
		return nil, fmt.Errorf("list user access rows: %w", err)
	}

	type grantRow struct {
		UserID    ids.UserID      `db:"user_id"`
		GroupID   ids.HostGroupID `db:"group_id"`
		GroupName string          `db:"group_name"`
	}
	const grantQuery = `
		SELECT uahg.user_id, hg.id AS group_id, hg.name AS group_name
		FROM user_allowed_host_groups uahg
		JOIN host_groups hg ON hg.id = uahg.host_group_id
		ORDER BY uahg.user_id, hg.name
	`
	var grantRows []grantRow
	if err := r.db.SelectContext(ctx, &grantRows, grantQuery); err != nil {
		return nil, fmt.Errorf("list user access rows groups: %w", err)
	}

	grantsByUser := make(map[ids.UserID][]httpapi.GroupRef)
	for _, gr := range grantRows {
		grantsByUser[gr.UserID] = append(grantsByUser[gr.UserID], httpapi.GroupRef{
			Id:   gr.GroupID.Int64(),
			Name: gr.GroupName,
		})
	}

	rows := make([]httpapi.UserListItem, len(userRows))
	for i, ur := range userRows {
		groups := grantsByUser[ur.ID]
		if groups == nil {
			groups = []httpapi.GroupRef{}
		}
		rows[i] = httpapi.UserListItem{
			Id:              ur.ID.Int64(),
			Username:        ur.UserName,
			DisplayName:     ur.DisplayName,
			Role:            httpapi.UserRole(ur.Role),
			BypassHostCheck: ur.BypassHostCheck,
			DeviceCount:     ur.DeviceCount,
			HostCount:       ur.HostCount,
			LiveIpCount:     ur.LiveIPCount,
			Groups:          groups,
		}
	}
	return rows, nil
}

// ── User access editor (drawer view) ─────────────────────────────────────────

func (r *Repository) GetUserAccessDetail(ctx context.Context, userID ids.UserID) (httpapi.UserAccessDetail, error) {
	// Q1: user base info + bypass flag
	type userRow struct {
		ID              ids.UserID `db:"id"`
		DisplayName     string     `db:"display_name"`
		Username        string     `db:"username"`
		Email           string     `db:"email"`
		Role            auth.Role  `db:"role"`
		BypassHostCheck bool       `db:"bypass_host_check"`
	}
	const userQuery = `
		SELECT u.id, u.display_name, u.username, u.email, u.role,
		       COALESCE(uhs.bypass_host_check, 0) AS bypass_host_check
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.id = ? AND u.deleted_at IS NULL
	`
	var ur userRow
	if err := r.db.GetContext(ctx, &ur, userQuery, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return httpapi.UserAccessDetail{}, auth.ErrUserNotFound
		}
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail user: %w", err)
	}

	// Q2: all groups with grant flag and their member hosts
	type groupRow struct {
		GroupID          ids.HostGroupID `db:"group_id"`
		GroupName        string          `db:"group_name"`
		GroupIcon        string          `db:"group_icon"`
		GroupColor       string          `db:"group_color"`
		GroupDescription *string         `db:"group_description"`
		GroupCreatedAt   time.Time       `db:"group_created_at"`
		GroupUpdatedAt   time.Time       `db:"group_updated_at"`
		Granted          bool            `db:"granted"`
		HostID           *ids.HostID     `db:"host_id"`
		HostFQDN         *string         `db:"host_fqdn"`
	}
	const groupQuery = `
		SELECT
			hg.id   AS group_id,
			hg.name AS group_name,
			hg.icon AS group_icon,
			hg.color AS group_color,
			hg.description AS group_description,
			hg.created_at AS group_created_at,
			hg.updated_at AS group_updated_at,
			CASE WHEN uahg.user_id IS NOT NULL THEN 1 ELSE 0 END AS granted,
			h.id   AS host_id,
			h.fqdn AS host_fqdn
		FROM host_groups hg
		LEFT JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hg.id AND uahg.user_id = ?
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		LEFT JOIN hosts h ON h.id = hgm.host_id
		ORDER BY hg.name, h.fqdn
	`
	var groupRows []groupRow
	if err := r.db.SelectContext(ctx, &groupRows, groupQuery, userID); err != nil {
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail groups: %w", err)
	}

	// Q3: network policies assigned to each group
	type userDetailPolicyRow struct {
		GroupID    ids.HostGroupID     `db:"host_group_id"`
		PolicyID   ids.NetworkPolicyID `db:"policy_id"`
		PolicyName string              `db:"policy_name"`
		PolicyCIDR string              `db:"policy_cidr"`
	}
	const userDetailPoliciesQuery = `
		SELECT nphg.host_group_id, np.id AS policy_id, np.name AS policy_name, np.cidr AS policy_cidr
		FROM network_policy_allowed_host_groups nphg
		JOIN network_policies np ON np.id = nphg.policy_id
		ORDER BY nphg.host_group_id, np.name
	`
	var userDetailPolicyRows []userDetailPolicyRow
	if err := r.db.SelectContext(ctx, &userDetailPolicyRows, userDetailPoliciesQuery); err != nil {
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail network policies: %w", err)
	}

	policiesByGroupForUser := make(map[ids.HostGroupID][]httpapi.NetworkPolicyRef)
	for _, pr := range userDetailPolicyRows {
		policiesByGroupForUser[pr.GroupID] = append(policiesByGroupForUser[pr.GroupID], httpapi.NetworkPolicyRef{
			Id:   pr.PolicyID.Int64(),
			Name: pr.PolicyName,
			Cidr: pr.PolicyCIDR,
		})
	}

	seenGroups := make(map[ids.HostGroupID]int)
	groups := []httpapi.SubjectGroupDetail{}
	for _, gr := range groupRows {
		idx, exists := seenGroups[gr.GroupID]
		if !exists {
			idx = len(groups)
			seenGroups[gr.GroupID] = idx
			policies := policiesByGroupForUser[gr.GroupID]
			if policies == nil {
				policies = []httpapi.NetworkPolicyRef{}
			}
			groups = append(groups, httpapi.SubjectGroupDetail{
				Id:              gr.GroupID.Int64(),
				Name:            gr.GroupName,
				Icon:            gr.GroupIcon,
				Color:           gr.GroupColor,
				Description:     gr.GroupDescription,
				CreatedAt:       httpapi.UTCTime(gr.GroupCreatedAt),
				UpdatedAt:       httpapi.UTCTime(gr.GroupUpdatedAt),
				Granted:         gr.Granted,
				Hosts:           []httpapi.HostSummary{},
				NetworkPolicies: policies,
			})
		}
		if gr.HostID != nil && gr.HostFQDN != nil {
			groups[idx].Hosts = append(groups[idx].Hosts, httpapi.HostSummary{
				Id:   (*gr.HostID).Int64(),
				Fqdn: *gr.HostFQDN,
			})
		}
	}

	// Q3: devices owned by the user
	deviceViews, err := r.GetDevicesByUser(ctx, userID)
	if err != nil {
		return httpapi.UserAccessDetail{}, fmt.Errorf("get user access detail devices: %w", err)
	}
	devices := make([]httpapi.DeviceListItem, len(deviceViews))
	for i := range deviceViews {
		devices[i] = httpapi.DeviceListItem{
			Id:           deviceViews[i].ID.Int64(),
			Name:         deviceViews[i].Name,
			ApiKeyPrefix: deviceViews[i].KeyPrefix,
			Icon:         deviceViews[i].Icon,
			LiveIpCount:  deviceViews[i].AddressCount,
		}
	}

	email := openapi_types.Email(ur.Email)
	return httpapi.UserAccessDetail{
		Id:              ur.ID.Int64(),
		Username:        ur.Username,
		DisplayName:     ur.DisplayName,
		BypassHostCheck: ur.BypassHostCheck,
		Role:            httpapi.UserRole(ur.Role),
		Email:           &email,
		Groups:          groups,
		Devices:         devices,
	}, nil
}
