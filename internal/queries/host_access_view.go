package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
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
	Description *string
	Icon        *string
	CreatedAt   time.Time
	Hosts       []hostaccess.KnownHostRef
}

func (r *Repository) GetHostGroupsWithMembers(ctx context.Context) ([]HostGroupWithMembers, error) {
	type row struct {
		ID          hostaccess.HostGroupID  `db:"id"`
		Name        string                  `db:"name"`
		Description *string                 `db:"description"`
		Icon        *string                 `db:"icon"`
		CreatedAt   time.Time               `db:"created_at"`
		HostID      *hostaccess.KnownHostID `db:"known_host_id"`
		HostFQDN    *string                 `db:"host_fqdn"`
		HostIcon    *string                 `db:"host_icon"`
	}
	const query = `
		SELECT hg.id, hg.name, hg.description, hg.icon, hg.created_at,
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
				Description: rw.Description,
				Icon:        rw.Icon,
				CreatedAt:   rw.CreatedAt,
				Hosts:       []hostaccess.KnownHostRef{},
			})
		}
		if rw.HostID != nil && rw.HostFQDN != nil {
			groups[idx].Hosts = append(groups[idx].Hosts, hostaccess.KnownHostRef{
				ID:   *rw.HostID,
				FQDN: *rw.HostFQDN,
				Icon: rw.HostIcon,
			})
		}
	}
	if groups == nil {
		groups = []HostGroupWithMembers{}
	}
	return groups, nil
}

// ── Host suggestions page ─────────────────────────────────────────────────────

type HostSuggestion struct {
	FQDN        string    `db:"fqdn"`
	FirstSeen   time.Time `db:"first_seen"`
	AllowedHits int       `db:"allowed_hits"`
	DeniedHits  int       `db:"denied_hits"`
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
	if suggestions == nil {
		suggestions = []HostSuggestion{}
	}

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

// ── Users host access summary ─────────────────────────────────────────────────

type UserHostAccessSummary struct {
	ID              auth.UserID
	DisplayName     string
	Email           string
	Role            auth.Role
	Bypass          bool
	DirectHostCount int
	Groups          []GroupRef
}

func (r *Repository) GetUsersHostAccess(ctx context.Context) ([]UserHostAccessSummary, error) {
	type row struct {
		ID              auth.UserID             `db:"id"`
		DisplayName     string                  `db:"display_name"`
		Email           string                  `db:"email"`
		Role            auth.Role               `db:"role"`
		Bypass          bool                    `db:"bypass"`
		DirectHostCount int                     `db:"direct_host_count"`
		GroupID         *hostaccess.HostGroupID `db:"group_id"`
		GroupName       *string                 `db:"group_name"`
	}
	const query = `
		SELECT
			u.id,
			u.display_name,
			u.email,
			u.role,
			COALESCE(uhs.bypass_host_allowlist, 0) AS bypass,
			(SELECT COUNT(*) FROM user_allowed_hosts WHERE user_id = u.id) AS direct_host_count,
			hg.id   AS group_id,
			hg.name AS group_name
		FROM users u
		LEFT JOIN user_host_settings uhs      ON uhs.user_id      = u.id
		LEFT JOIN user_allowed_host_groups uahg ON uahg.user_id   = u.id
		LEFT JOIN host_groups hg              ON hg.id             = uahg.host_group_id
		WHERE u.deleted_at IS NULL
		ORDER BY u.display_name, hg.name
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get users host access: %w", err)
	}

	seen := make(map[auth.UserID]int)
	var users []UserHostAccessSummary
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(users)
			seen[rw.ID] = idx
			users = append(users, UserHostAccessSummary{
				ID:              rw.ID,
				DisplayName:     rw.DisplayName,
				Email:           rw.Email,
				Role:            rw.Role,
				Bypass:          rw.Bypass,
				DirectHostCount: rw.DirectHostCount,
				Groups:          []GroupRef{},
			})
		}
		if rw.GroupID != nil && rw.GroupName != nil {
			users[idx].Groups = append(users[idx].Groups, GroupRef{
				ID:   *rw.GroupID,
				Name: *rw.GroupName,
			})
		}
	}
	if users == nil {
		users = []UserHostAccessSummary{}
	}
	return users, nil
}

// ── User host details ─────────────────────────────────────────────────────────

type UserHostDetailsGroup struct {
	ID      hostaccess.HostGroupID
	Name    string
	Icon    *string
	Granted bool
	Hosts   []hostaccess.KnownHostRef
}

type UserHostDetailsHost struct {
	ID              hostaccess.KnownHostID
	FQDN            string
	Icon            *string
	DirectlyGranted bool
	ViaGroup        *GroupRef
}

type UserHostDetails struct {
	ID          auth.UserID
	DisplayName string
	Email       string
	Role        auth.Role
	Bypass      bool
	Groups      []UserHostDetailsGroup
	Hosts       []UserHostDetailsHost
}

func (r *Repository) GetUserHostDetails(ctx context.Context, userID auth.UserID) (UserHostDetails, error) {
	// Q1: user base info
	type userRow struct {
		ID          auth.UserID `db:"id"`
		DisplayName string      `db:"display_name"`
		Email       string      `db:"email"`
		Role        auth.Role   `db:"role"`
		Bypass      bool        `db:"bypass"`
	}
	const userQuery = `
		SELECT u.id, u.display_name, u.email, u.role,
		       COALESCE(uhs.bypass_host_allowlist, 0) AS bypass
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.id = ? AND u.deleted_at IS NULL
	`
	var ur userRow
	if err := r.db.GetContext(ctx, &ur, userQuery, userID); err != nil {
		return UserHostDetails{}, auth.ErrUserNotFound
	}

	// Q2: all groups with granted flag and their member hosts
	type groupRow struct {
		GroupID   hostaccess.HostGroupID  `db:"group_id"`
		GroupName string                  `db:"group_name"`
		GroupIcon *string                 `db:"group_icon"`
		Granted   bool                    `db:"granted"`
		HostID    *hostaccess.KnownHostID `db:"host_id"`
		HostFQDN  *string                 `db:"host_fqdn"`
		HostIcon  *string                 `db:"host_icon"`
	}
	const groupQuery = `
		SELECT
			hg.id   AS group_id,
			hg.name AS group_name,
			hg.icon AS group_icon,
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
		return UserHostDetails{}, fmt.Errorf("get user host details groups: %w", err)
	}

	seenGroups := make(map[hostaccess.HostGroupID]int)
	var groups []UserHostDetailsGroup
	for _, gr := range groupRows {
		idx, exists := seenGroups[gr.GroupID]
		if !exists {
			idx = len(groups)
			seenGroups[gr.GroupID] = idx
			groups = append(groups, UserHostDetailsGroup{
				ID:      gr.GroupID,
				Name:    gr.GroupName,
				Icon:    gr.GroupIcon,
				Granted: gr.Granted,
				Hosts:   []hostaccess.KnownHostRef{},
			})
		}
		if gr.HostID != nil && gr.HostFQDN != nil {
			groups[idx].Hosts = append(groups[idx].Hosts, hostaccess.KnownHostRef{
				ID:   *gr.HostID,
				FQDN: *gr.HostFQDN,
				Icon: gr.HostIcon,
			})
		}
	}
	if groups == nil {
		groups = []UserHostDetailsGroup{}
	}

	// Q3: all known hosts with grant status and first granted-group (alphabetically)
	type hostRow struct {
		HostID          hostaccess.KnownHostID  `db:"host_id"`
		HostFQDN        string                  `db:"host_fqdn"`
		HostIcon        *string                 `db:"host_icon"`
		DirectlyGranted bool                    `db:"directly_granted"`
		ViaGroupID      *hostaccess.HostGroupID `db:"via_group_id"`
		ViaGroupName    *string                 `db:"via_group_name"`
	}
	const hostQuery = `
		SELECT
			kh.id   AS host_id,
			kh.fqdn AS host_fqdn,
			kh.icon AS host_icon,
			CASE WHEN uah.user_id IS NOT NULL THEN 1 ELSE 0 END AS directly_granted,
			hg.id   AS via_group_id,
			hg.name AS via_group_name
		FROM known_hosts kh
		LEFT JOIN user_allowed_hosts uah ON uah.known_host_id = kh.id AND uah.user_id = ?
		LEFT JOIN host_group_members hgm ON hgm.known_host_id = kh.id
		LEFT JOIN user_allowed_host_groups uahg ON uahg.host_group_id = hgm.host_group_id AND uahg.user_id = ?
		LEFT JOIN host_groups hg ON hg.id = hgm.host_group_id
		ORDER BY kh.fqdn, hg.name
	`
	var hostRows []hostRow
	if err := r.db.SelectContext(ctx, &hostRows, hostQuery, userID, userID); err != nil {
		return UserHostDetails{}, fmt.Errorf("get user host details hosts: %w", err)
	}

	seenHosts := make(map[hostaccess.KnownHostID]int)
	var hosts []UserHostDetailsHost
	for _, hr := range hostRows {
		idx, exists := seenHosts[hr.HostID]
		if !exists {
			idx = len(hosts)
			seenHosts[hr.HostID] = idx
			hosts = append(hosts, UserHostDetailsHost{
				ID:              hr.HostID,
				FQDN:            hr.HostFQDN,
				Icon:            hr.HostIcon,
				DirectlyGranted: hr.DirectlyGranted,
				ViaGroup:        nil,
			})
		}
		// via_group_id is non-null only when the host is covered by a granted group.
		// Rows are ordered by hg.name so the first non-null encountered is alphabetically first.
		if hr.ViaGroupID != nil && hr.ViaGroupName != nil && hosts[idx].ViaGroup == nil {
			hosts[idx].ViaGroup = &GroupRef{ID: *hr.ViaGroupID, Name: *hr.ViaGroupName}
		}
	}
	if hosts == nil {
		hosts = []UserHostDetailsHost{}
	}

	return UserHostDetails{
		ID:          ur.ID,
		DisplayName: ur.DisplayName,
		Email:       ur.Email,
		Role:        ur.Role,
		Bypass:      ur.Bypass,
		Groups:      groups,
		Hosts:       hosts,
	}, nil
}
