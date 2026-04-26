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
)

// ── GroupRef ──────────────────────────────────────────────────────────────────

type GroupRef struct {
	ID   hostaccess.HostGroupID
	Name string
}

// ── Known hosts with stats ────────────────────────────────────────────────────

type KnownHostStats struct {
	ID        hostaccess.KnownHostID `db:"id"`
	FQDN      string                 `db:"fqdn"`
	Icon      *string                `db:"icon"`
	CreatedAt time.Time              `db:"created_at"`
	UserCount int                    `db:"user_count"`
	Groups    []GroupRef
}

func (r *Repository) GetKnownHostsWithStats(ctx context.Context) ([]KnownHostStats, error) {
	type row struct {
		ID        hostaccess.KnownHostID  `db:"id"`
		FQDN      string                  `db:"fqdn"`
		Icon      *string                 `db:"icon"`
		CreatedAt time.Time               `db:"created_at"`
		UserCount int                     `db:"user_count"`
		GroupID   *hostaccess.HostGroupID `db:"group_id"`
		GroupName *string                 `db:"group_name"`
	}
	const query = `
		SELECT
			kh.id          AS id,
			kh.fqdn        AS fqdn,
			kh.icon        AS icon,
			kh.created_at  AS created_at,
			(
				SELECT COUNT(DISTINCT user_id) FROM (
					SELECT uah.user_id FROM user_allowed_hosts uah WHERE uah.known_host_id = kh.id
					UNION
					SELECT uahg.user_id FROM user_allowed_host_groups uahg
					JOIN host_group_members hgm2 ON hgm2.host_group_id = uahg.host_group_id
					WHERE hgm2.known_host_id = kh.id
				)
			) AS user_count,
			hg.id   AS group_id,
			hg.name AS group_name
		FROM known_hosts kh
		LEFT JOIN host_group_members hgm ON hgm.known_host_id = kh.id
		LEFT JOIN host_groups hg ON hg.id = hgm.host_group_id
		ORDER BY kh.fqdn, hg.name
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get known hosts with stats: %w", err)
	}

	seen := make(map[hostaccess.KnownHostID]int)
	var hosts []KnownHostStats
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(hosts)
			seen[rw.ID] = idx
			hosts = append(hosts, KnownHostStats{
				ID:        rw.ID,
				FQDN:      rw.FQDN,
				Icon:      rw.Icon,
				CreatedAt: rw.CreatedAt,
				UserCount: rw.UserCount,
				Groups:    []GroupRef{},
			})
		}
		if rw.GroupID != nil && rw.GroupName != nil {
			hosts[idx].Groups = append(hosts[idx].Groups, GroupRef{
				ID:   *rw.GroupID,
				Name: *rw.GroupName,
			})
		}
	}
	if hosts == nil {
		hosts = []KnownHostStats{}
	}
	return hosts, nil
}

// ── Host groups with members ──────────────────────────────────────────────────

type HostGroupWithMembers struct {
	ID          hostaccess.HostGroupID
	Name        string
	Color       *string
	Description *string
	Icon        *string
	CreatedAt   time.Time
	Hosts       []hostaccess.KnownHostRef
	MemberIDs   []hostaccess.KnownHostID
}

func (r *Repository) GetHostGroupsWithMembers(ctx context.Context) ([]HostGroupWithMembers, error) {
	type row struct {
		ID          hostaccess.HostGroupID  `db:"id"`
		Name        string                  `db:"name"`
		Color       *string                 `db:"color"`
		Description *string                 `db:"description"`
		Icon        *string                 `db:"icon"`
		CreatedAt   time.Time               `db:"created_at"`
		HostID      *hostaccess.KnownHostID `db:"known_host_id"`
		HostFQDN    *string                 `db:"host_fqdn"`
		HostIcon    *string                 `db:"host_icon"`
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

	seen := make(map[hostaccess.HostGroupID]int)
	var groups []HostGroupWithMembers
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(groups)
			seen[rw.ID] = idx
			groups = append(groups, HostGroupWithMembers{
				ID:          rw.ID,
				Name:        rw.Name,
				Color:       rw.Color,
				Description: rw.Description,
				Icon:        rw.Icon,
				CreatedAt:   rw.CreatedAt,
				Hosts:       []hostaccess.KnownHostRef{},
				MemberIDs:   []hostaccess.KnownHostID{},
			})
		}
		if rw.HostID != nil && rw.HostFQDN != nil {
			groups[idx].Hosts = append(groups[idx].Hosts, hostaccess.KnownHostRef{
				ID:   *rw.HostID,
				FQDN: *rw.HostFQDN,
				Icon: rw.HostIcon,
			})
			groups[idx].MemberIDs = append(groups[idx].MemberIDs, *rw.HostID)
		}
	}
	if groups == nil {
		groups = []HostGroupWithMembers{}
	}
	return groups, nil
}

// ── Host suggestions page ─────────────────────────────────────────────────────

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
	ID                 auth.UserID
	DisplayName        string
	Email              string
	Role               auth.Role
	AllowAllHosts      bool
	EffectiveHostCount int
	GrantedGroups      []GroupRef
}

func (r *Repository) ListUserAccessRows(ctx context.Context) ([]UserAccessRow, error) {
	type userRow struct {
		ID                 auth.UserID `db:"id"`
		DisplayName        string      `db:"display_name"`
		Email              string      `db:"email"`
		Role               auth.Role   `db:"role"`
		AllowAllHosts      bool        `db:"allow_all_hosts"`
		EffectiveHostCount int         `db:"effective_host_count"`
	}
	const userQuery = `
		SELECT
			u.id,
			u.display_name,
			u.email,
			u.role,
			COALESCE(uhs.bypass_host_allowlist, 0) AS allow_all_hosts,
			CASE
				WHEN COALESCE(uhs.bypass_host_allowlist, 0) = 1 THEN (SELECT COUNT(*) FROM known_hosts)
				ELSE (
					SELECT COUNT(DISTINCT kh.id)
					FROM known_hosts kh
					WHERE kh.id IN (
						SELECT uah.known_host_id FROM user_allowed_hosts uah WHERE uah.user_id = u.id
						UNION
						SELECT hgm.known_host_id FROM host_group_members hgm
						JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hgm.host_group_id
						WHERE uahg.user_id = u.id
					)
				)
			END AS effective_host_count
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
		UserID    auth.UserID            `db:"user_id"`
		GroupID   hostaccess.HostGroupID `db:"group_id"`
		GroupName string                 `db:"group_name"`
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

	grantsByUser := make(map[auth.UserID][]GroupRef)
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
			ID:                 ur.ID,
			DisplayName:        ur.DisplayName,
			Email:              ur.Email,
			Role:               ur.Role,
			AllowAllHosts:      ur.AllowAllHosts,
			EffectiveHostCount: ur.EffectiveHostCount,
			GrantedGroups:      groups,
		}
	}
	return rows, nil
}

// ── User access editor (drawer view) ─────────────────────────────────────────

type UserSummary struct {
	ID          auth.UserID
	DisplayName string
	Email       string
	Role        auth.Role
}

type UserAccessSummary struct {
	TotalHosts     int
	EffectiveHosts int
	DirectHosts    int
	// GroupOnlyHosts counts hosts reachable exclusively via a granted group (not also directly).
	// Useful for "you would lose access to N hosts if all groups were revoked."
	GroupOnlyHosts int
}

// HostAccessKind is the computed access state for a single host in the drawer.
type HostAccessKind string

const (
	HostAccessNone         HostAccessKind = "none"
	HostAccessDirect       HostAccessKind = "direct"
	HostAccessViaGroup     HostAccessKind = "via_group"
	HostAccessDirectAndVia HostAccessKind = "direct_and_via_group"
	HostAccessAllowAll     HostAccessKind = "allow_all"
)

type UserAccessGroupOption struct {
	ID       hostaccess.HostGroupID
	Name     string
	Icon     *string
	Selected bool
	Hosts    []hostaccess.KnownHostRef
}

type UserAccessHostOption struct {
	ID             hostaccess.KnownHostID
	FQDN           string
	Icon           *string
	DirectSelected bool
	Effective      bool
	GrantingGroups []GroupRef
	AccessKind     HostAccessKind
}

type UserAccessEditor struct {
	User          UserSummary
	AllowAllHosts bool
	Summary       UserAccessSummary
	GroupOptions  []UserAccessGroupOption
	HostOptions   []UserAccessHostOption
}

func (r *Repository) GetUserAccessEditor(ctx context.Context, userID auth.UserID) (UserAccessEditor, error) {
	// Q1: user base info + allow_all_hosts
	type userRow struct {
		ID            auth.UserID `db:"id"`
		DisplayName   string      `db:"display_name"`
		Email         string      `db:"email"`
		Role          auth.Role   `db:"role"`
		AllowAllHosts bool        `db:"allow_all_hosts"`
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
			return UserAccessEditor{}, auth.ErrUserNotFound
		}
		return UserAccessEditor{}, fmt.Errorf("get user access editor user: %w", err)
	}

	// Q2: all groups with selected flag and their member hosts
	type groupRow struct {
		GroupID   hostaccess.HostGroupID  `db:"group_id"`
		GroupName string                  `db:"group_name"`
		GroupIcon *string                 `db:"group_icon"`
		Selected  bool                    `db:"selected"`
		HostID    *hostaccess.KnownHostID `db:"host_id"`
		HostFQDN  *string                 `db:"host_fqdn"`
		HostIcon  *string                 `db:"host_icon"`
	}
	const groupQuery = `
		SELECT
			hg.id   AS group_id,
			hg.name AS group_name,
			hg.icon AS group_icon,
			CASE WHEN uahg.user_id IS NOT NULL THEN 1 ELSE 0 END AS selected,
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
		return UserAccessEditor{}, fmt.Errorf("get user access editor groups: %w", err)
	}

	seenGroups := make(map[hostaccess.HostGroupID]int)
	groupOptions := []UserAccessGroupOption{}
	for _, gr := range groupRows {
		idx, exists := seenGroups[gr.GroupID]
		if !exists {
			idx = len(groupOptions)
			seenGroups[gr.GroupID] = idx
			groupOptions = append(groupOptions, UserAccessGroupOption{
				ID:       gr.GroupID,
				Name:     gr.GroupName,
				Icon:     gr.GroupIcon,
				Selected: gr.Selected,
				Hosts:    []hostaccess.KnownHostRef{},
			})
		}
		if gr.HostID != nil && gr.HostFQDN != nil {
			groupOptions[idx].Hosts = append(groupOptions[idx].Hosts, hostaccess.KnownHostRef{
				ID:   *gr.HostID,
				FQDN: *gr.HostFQDN,
				Icon: gr.HostIcon,
			})
		}
	}

	// Q3: all known hosts
	type hostRow struct {
		ID   hostaccess.KnownHostID `db:"id"`
		FQDN string                 `db:"fqdn"`
		Icon *string                `db:"icon"`
	}
	const hostsQuery = `SELECT id, fqdn, icon FROM known_hosts ORDER BY fqdn`
	var hostRows []hostRow
	if err := r.db.SelectContext(ctx, &hostRows, hostsQuery); err != nil {
		return UserAccessEditor{}, fmt.Errorf("get user access editor hosts: %w", err)
	}

	// Q4: direct host grants for this user
	const directGrantQuery = `SELECT known_host_id FROM user_allowed_hosts WHERE user_id = ?`
	var directGrantIDs []hostaccess.KnownHostID
	if err := r.db.SelectContext(ctx, &directGrantIDs, directGrantQuery, userID); err != nil {
		return UserAccessEditor{}, fmt.Errorf("get user access editor direct grants: %w", err)
	}
	directGrantSet := make(map[hostaccess.KnownHostID]struct{}, len(directGrantIDs))
	for _, id := range directGrantIDs {
		directGrantSet[id] = struct{}{}
	}

	// Q5: all groups granting each host to this user (no order dependency)
	type grantingGroupRow struct {
		KnownHostID hostaccess.KnownHostID `db:"known_host_id"`
		GroupID     hostaccess.HostGroupID `db:"group_id"`
		GroupName   string                 `db:"group_name"`
	}
	const grantingGroupQuery = `
		SELECT hgm.known_host_id, hg.id AS group_id, hg.name AS group_name
		FROM host_group_members hgm
		JOIN host_groups hg ON hg.id = hgm.host_group_id
		JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hgm.host_group_id AND uahg.user_id = ?
		ORDER BY hgm.known_host_id, hg.name
	`
	var grantingGroupRows []grantingGroupRow
	if err := r.db.SelectContext(ctx, &grantingGroupRows, grantingGroupQuery, userID); err != nil {
		return UserAccessEditor{}, fmt.Errorf("get user access editor granting groups: %w", err)
	}
	grantingGroupsByHost := make(map[hostaccess.KnownHostID][]GroupRef)
	for _, gg := range grantingGroupRows {
		grantingGroupsByHost[gg.KnownHostID] = append(grantingGroupsByHost[gg.KnownHostID], GroupRef{ID: gg.GroupID, Name: gg.GroupName})
	}

	// Assemble host options and compute summary counts.
	hostOptions := make([]UserAccessHostOption, len(hostRows))
	var directHosts, groupOnlyHosts, effectiveHosts int
	for i, hr := range hostRows {
		_, directSelected := directGrantSet[hr.ID]
		grantingGroups := grantingGroupsByHost[hr.ID]
		if grantingGroups == nil {
			grantingGroups = []GroupRef{}
		}
		effective := ur.AllowAllHosts || directSelected || len(grantingGroups) > 0
		accessKind := hostAccessKind(ur.AllowAllHosts, directSelected, grantingGroups)
		hostOptions[i] = UserAccessHostOption{
			ID:             hr.ID,
			FQDN:           hr.FQDN,
			Icon:           hr.Icon,
			DirectSelected: directSelected,
			Effective:      effective,
			GrantingGroups: grantingGroups,
			AccessKind:     accessKind,
		}
		if directSelected {
			directHosts++
		}
		if len(grantingGroups) > 0 && !directSelected {
			groupOnlyHosts++
		}
		if effective {
			effectiveHosts++
		}
	}

	return UserAccessEditor{
		User: UserSummary{
			ID:          ur.ID,
			DisplayName: ur.DisplayName,
			Email:       ur.Email,
			Role:        ur.Role,
		},
		AllowAllHosts: ur.AllowAllHosts,
		Summary: UserAccessSummary{
			TotalHosts:     len(hostRows),
			EffectiveHosts: effectiveHosts,
			DirectHosts:    directHosts,
			GroupOnlyHosts: groupOnlyHosts,
		},
		GroupOptions: groupOptions,
		HostOptions:  hostOptions,
	}, nil
}

func hostAccessKind(allowAll, directSelected bool, grantingGroups []GroupRef) HostAccessKind {
	if allowAll {
		return HostAccessAllowAll
	}
	if directSelected && len(grantingGroups) > 0 {
		return HostAccessDirectAndVia
	}
	if directSelected {
		return HostAccessDirect
	}
	if len(grantingGroups) > 0 {
		return HostAccessViaGroup
	}
	return HostAccessNone
}
