package queries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hostaccess"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type GroupRef struct {
	ID    ids.HostGroupID
	Name  string
	Color string
}

type HostWithGroups struct {
	ID        ids.KnownHostID `db:"id"`
	FQDN      string          `db:"fqdn"`
	CreatedAt time.Time       `db:"created_at"`
	UserCount int             `db:"user_count"`
	Groups    []GroupRef
}

func (r *Repository) GetAllHostsWithGroups(ctx context.Context) ([]HostWithGroups, error) {
	type row struct {
		ID         ids.KnownHostID  `db:"id"`
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
		FROM known_hosts kh
		LEFT JOIN host_group_members hgm ON hgm.known_host_id = kh.id
		LEFT JOIN host_groups hg ON hg.id = hgm.host_group_id
		ORDER BY kh.fqdn, hg.name
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get known hosts with stats: %w", err)
	}

	seen := make(map[ids.KnownHostID]int)
	var hosts []HostWithGroups
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(hosts)
			seen[rw.ID] = idx
			hosts = append(hosts, HostWithGroups{
				ID:        rw.ID,
				FQDN:      rw.FQDN,
				CreatedAt: rw.CreatedAt,
				Groups:    []GroupRef{},
			})
		}
		if rw.GroupID != nil && rw.GroupName != nil {
			hosts[idx].Groups = append(hosts[idx].Groups, GroupRef{
				ID:    *rw.GroupID,
				Name:  *rw.GroupName,
				Color: *rw.GroupColor,
			})
		}
	}
	if hosts == nil {
		hosts = []HostWithGroups{}
	}
	return hosts, nil
}

type HostGroupDetails struct {
	ID          ids.HostGroupID
	Name        string
	Color       string
	Icon        string
	Description *string
	CreatedAt   time.Time
	Hosts       []KnownHostRef
	MemberIDs   []ids.KnownHostID
}

func (r *Repository) GetHostGroupsDetails(ctx context.Context) ([]HostGroupDetails, error) {
	type row struct {
		ID          ids.HostGroupID  `db:"id"`
		Name        string           `db:"name"`
		Color       string           `db:"color"`
		Icon        string           `db:"icon"`
		Description *string          `db:"description"`
		CreatedAt   time.Time        `db:"created_at"`
		HostID      *ids.KnownHostID `db:"known_host_id"`
		HostFQDN    *string          `db:"host_fqdn"`
		HostIcon    *string          `db:"host_icon"`
	}
	const query = `
		SELECT hg.id, hg.name, hg.color, hg.description, hg.icon, hg.created_at,
		       hgm.known_host_id, kh.fqdn AS host_fqdn, kh.icon AS host_icon
		FROM host_groups hg
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		LEFT JOIN known_hosts kh ON kh.id = hgm.known_host_id
		ORDER BY hg.name, kh.fqdn
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get host groups with members: %w", err)
	}

	seen := make(map[ids.HostGroupID]int)
	var groups []HostGroupDetails
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(groups)
			seen[rw.ID] = idx
			groups = append(groups, HostGroupDetails{
				ID:          rw.ID,
				Name:        rw.Name,
				Color:       rw.Color,
				Description: rw.Description,
				Icon:        rw.Icon,
				CreatedAt:   rw.CreatedAt,
				Hosts:       []KnownHostRef{},
			})
		}
		if rw.HostID != nil && rw.HostFQDN != nil {
			groups[idx].Hosts = append(groups[idx].Hosts, KnownHostRef{
				ID:   *rw.HostID,
				FQDN: *rw.HostFQDN,
			})
		}
	}
	if groups == nil {
		groups = []HostGroupDetails{}
	}
	return groups, nil
}

type HostSuggestion struct {
	FQDN        string          `db:"fqdn"`
	FirstSeen   database.DBTime `db:"first_seen"`
	AllowedHits int             `db:"allowed_hits"`
	DeniedHits  int             `db:"denied_hits"`
}

type HostSuggestionsPage struct {
	Suggestions []HostSuggestion
	Ignored     []hostaccess.IgnoredHostSuggestion
}

func (r *Repository) GetHostSuggestionsPage(ctx context.Context) (HostSuggestionsPage, error) {
	const suggestionsQuery = `
		SELECT
			LOWER(al.target_host) AS fqdn,
			MIN(al.created_at)    AS first_seen,
			SUM(CASE WHEN al.outcome = 1 THEN 1 ELSE 0 END) AS allowed_hits,
			SUM(CASE WHEN al.outcome = 0 THEN 1 ELSE 0 END) AS denied_hits
		FROM access_log al
		WHERE al.target_host IS NOT NULL
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM known_hosts)
		  AND LOWER(al.target_host) NOT IN (SELECT fqdn FROM ignored_host_suggestions)
		GROUP BY LOWER(al.target_host)
		ORDER BY denied_hits DESC, allowed_hits DESC
	`
	var suggestions []HostSuggestion
	if err := r.db.SelectContext(ctx, &suggestions, suggestionsQuery); err != nil {
		return HostSuggestionsPage{}, fmt.Errorf("get host suggestions: %w", err)
	}

	// Filter out entries that aren't valid FQDNs (e.g. bare IPs, host:port, garbage).
	// This ensures any suggestion can be directly added as a known host.
	valid := make([]HostSuggestion, 0, len(suggestions))
	for _, s := range suggestions {
		if hostaccess.ValidateFQDN(s.FQDN) == nil {
			valid = append(valid, s)
		}
	}
	suggestions = valid

	const ignoredQuery = `
		SELECT id, fqdn, created_at
		FROM ignored_host_suggestions
		ORDER BY fqdn
	`
	var ignored []hostaccess.IgnoredHostSuggestion
	if err := r.db.SelectContext(ctx, &ignored, ignoredQuery); err != nil {
		return HostSuggestionsPage{}, fmt.Errorf("get ignored suggestions: %w", err)
	}
	if ignored == nil {
		ignored = []hostaccess.IgnoredHostSuggestion{}
	}

	return HostSuggestionsPage{Suggestions: suggestions, Ignored: ignored}, nil
}

// ── User access table (list view) ─────────────────────────────────────────────

type UserAccessRow struct {
	ID              ids.UserID
	Username        string
	DisplayName     string
	Role            auth.Role
	BypassHostCheck bool
	DeviceCount     int
	HostCount       int
	LiveIPCount     int
	Groups          []GroupRef
}

func (r *Repository) ListUserAccessRows(ctx context.Context) ([]UserAccessRow, error) {
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
			uhs.bypass_host_allowlist as bypass_host_check,
			CASE WHEN COALESCE(uhs.bypass_host_allowlist, 0) = 1 THEN
				(SELECT COUNT(*) FROM known_hosts)
			ELSE
				(
					SELECT COUNT(DISTINCT kh.id)
					FROM known_hosts kh
					WHERE kh.id IN (
						SELECT hgm.known_host_id FROM host_group_members hgm
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

	grantsByUser := make(map[ids.UserID][]GroupRef)
	for _, gr := range grantRows {
		grantsByUser[gr.UserID] = append(grantsByUser[gr.UserID], GroupRef{ID: gr.GroupID, Name: gr.GroupName})
	}

	rows := make([]UserAccessRow, len(userRows))
	for i, ur := range userRows {
		groups := grantsByUser[ur.ID]
		if groups == nil {
			groups = []GroupRef{}
		}
		rows[i] = UserAccessRow{
			ID:              ur.ID,
			Username:        ur.UserName,
			DisplayName:     ur.DisplayName,
			Role:            ur.Role,
			BypassHostCheck: ur.BypassHostCheck,
			DeviceCount:     ur.DeviceCount,
			HostCount:       ur.HostCount,
			LiveIPCount:     ur.LiveIPCount,
			Groups:          groups,
		}
	}
	return rows, nil
}

// ── User access editor (drawer view) ─────────────────────────────────────────

type UserSummary struct {
	ID          ids.UserID
	DisplayName string
	Username    string
	Email       string
	Role        auth.Role
}

type UserAccessGroupOption struct {
	ID              ids.HostGroupID
	Name            string
	Description     *string
	Icon            string
	Color           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Granted         bool
	Hosts           []KnownHostRef
	NetworkPolicies []NetworkPolicyRef
}

type KnownHostRef struct {
	ID   ids.KnownHostID
	FQDN string
}

type NetworkPolicyRef struct {
	ID   ids.NetworkPolicyID
	Name string
	CIDR string
}

type UserAccessDetails struct {
	User            UserSummary
	BypassHostCheck bool
	GroupOptions    []UserAccessGroupOption
}

func (r *Repository) GetUserAccessDetail(ctx context.Context, userID ids.UserID) (UserAccessDetails, error) {
	// Q1: user base info + allow_all_hosts
	type userRow struct {
		ID            ids.UserID `db:"id"`
		DisplayName   string     `db:"display_name"`
		Email         string     `db:"email"`
		Role          auth.Role  `db:"role"`
		AllowAllHosts bool       `db:"allow_all_hosts"`
	}
	const userQuery = `
		SELECT u.id, u.display_name, u.email, u.role,
		       COALESCE(uhs.bypass_host_allowlist, 0) AS allow_all_hosts
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.id = ? AND u.deleted_at IS NULL
	`
	var ur userRow
	if err := r.db.GetContext(ctx, &ur, userQuery, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserAccessDetails{}, auth.ErrUserNotFound
		}
		return UserAccessDetails{}, fmt.Errorf("get user access editor user: %w", err)
	}

	// Q2: all groups with selected flag and their member hosts
	type groupRow struct {
		GroupID          ids.HostGroupID  `db:"group_id"`
		GroupName        string           `db:"group_name"`
		GroupIcon        string           `db:"group_icon"`
		GroupColor       string           `db:"group_color"`
		GroupDescription *string          `db:"group_description"`
		GroupCreatedAt   time.Time        `db:"group_created_at"`
		GroupUpdatedAt   time.Time        `db:"group_updated_at"`
		Granted          bool             `db:"granted"`
		HostID           *ids.KnownHostID `db:"host_id"`
		HostFQDN         *string          `db:"host_fqdn"`
		HostIcon         *string          `db:"host_icon"`
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
			kh.id   AS host_id,
			kh.fqdn AS host_fqdn,
			kh.icon AS host_icon
		FROM host_groups hg
		LEFT JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hg.id AND uahg.user_id = ?
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		LEFT JOIN known_hosts kh ON kh.id = hgm.known_host_id
		ORDER BY hg.name, kh.fqdn
	`
	var groupRows []groupRow
	if err := r.db.SelectContext(ctx, &groupRows, groupQuery, userID); err != nil {
		return UserAccessDetails{}, fmt.Errorf("get user access editor groups: %w", err)
	}

	seenGroups := make(map[ids.HostGroupID]int)
	groupOptions := []UserAccessGroupOption{}
	for _, gr := range groupRows {
		idx, exists := seenGroups[gr.GroupID]
		if !exists {
			idx = len(groupOptions)
			seenGroups[gr.GroupID] = idx
			groupOptions = append(groupOptions, UserAccessGroupOption{
				ID:          gr.GroupID,
				Name:        gr.GroupName,
				Icon:        gr.GroupIcon,
				Color:       gr.GroupColor,
				Description: gr.GroupDescription,
				CreatedAt:   gr.GroupCreatedAt,
				UpdatedAt:   gr.GroupUpdatedAt,
				Granted:     gr.Granted,
				Hosts:       []KnownHostRef{},
			})
		}
		if gr.HostID != nil && gr.HostFQDN != nil {
			groupOptions[idx].Hosts = append(groupOptions[idx].Hosts, KnownHostRef{
				ID:   *gr.HostID,
				FQDN: *gr.HostFQDN,
			})
		}
	}

	return UserAccessDetails{
		User: UserSummary{
			ID:          ur.ID,
			DisplayName: ur.DisplayName,
			Email:       ur.Email,
			Role:        ur.Role,
		},
		BypassHostCheck: ur.AllowAllHosts,
		GroupOptions:    groupOptions,
	}, nil
}
