package hostaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// ── Known hosts ───────────────────────────────────────────────────────────────
func (r *Repository) GetKnownHost(ctx context.Context, id KnownHostID) (KnownHost, error) {
	const query = `SELECT id, fqdn, icon, created_at FROM known_hosts WHERE id = ?`
	host := new(KnownHost)
	if err := r.db.GetContext(ctx, host, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return KnownHost{}, ErrKnownHostNotFound
		}
		return KnownHost{}, fmt.Errorf("get known host: %w", err)
	}
	return *host, nil
}

func (r *Repository) ListKnownHosts(ctx context.Context) ([]KnownHost, error) {
	const query = `SELECT id, fqdn, icon, created_at FROM known_hosts ORDER BY fqdn`
	var hosts []KnownHost
	if err := r.db.SelectContext(ctx, &hosts, query); err != nil {
		return nil, fmt.Errorf("list known hosts: %w", err)
	}
	if hosts == nil {
		hosts = []KnownHost{}
	}
	return hosts, nil
}

func (r *Repository) CreateKnownHost(ctx context.Context, fqdn string, icon *string) (KnownHost, error) {
	const query = `INSERT INTO known_hosts (fqdn, icon) VALUES (?, ?) RETURNING id, fqdn, icon, created_at`
	host := new(KnownHost)
	if err := r.db.GetContext(ctx, host, query, strings.ToLower(fqdn), icon); err != nil {
		if isUniqueViolation(err) {
			return KnownHost{}, ErrKnownHostConflict
		}
		return KnownHost{}, fmt.Errorf("create known host: %w", err)
	}
	return *host, nil
}

func (r *Repository) UpdateKnownHost(ctx context.Context, id KnownHostID, icon *string) (KnownHost, error) {
	const query = `UPDATE known_hosts SET icon = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? RETURNING *`
	host := new(KnownHost)
	if err := r.db.GetContext(ctx, host, query, icon, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return KnownHost{}, ErrKnownHostNotFound
		}
		return KnownHost{}, fmt.Errorf("update known host: %w", err)
	}
	return *host, nil
}

func (r *Repository) BulkCreateKnownHosts(ctx context.Context, fqdns []string) ([]KnownHost, error) {
	hosts := make([]KnownHost, 0, len(fqdns))
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		const query = `INSERT INTO known_hosts (fqdn) VALUES (?) RETURNING *`
		for _, fqdn := range fqdns {
			host := new(KnownHost)
			if err := r.db.GetContext(ctx, host, query, strings.ToLower(fqdn)); err != nil {
				//TODO: Should we fail or just ignore this? It is a conflict, but the end result is the right one
				if isUniqueViolation(err) {
					return ErrKnownHostConflict
				}
				return fmt.Errorf("bulk create known host %q: %w", fqdn, err)
			}
			hosts = append(hosts, *host)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func (r *Repository) DeleteKnownHost(ctx context.Context, id KnownHostID) error {
	const query = `DELETE FROM known_hosts WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete known host: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrKnownHostNotFound
	}
	return nil
}

// ── Host groups ───────────────────────────────────────────────────────────────

func (r *Repository) GetHostGroup(ctx context.Context, id HostGroupID) (HostGroup, error) {
	const query = `SELECT id, name, description, icon, created_at FROM host_groups WHERE id = ?`
	group := new(HostGroup)
	if err := r.db.GetContext(ctx, group, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return HostGroup{}, ErrHostGroupNotFound
		}
		return HostGroup{}, fmt.Errorf("get host group: %w", err)
	}
	return *group, nil
}

func (r *Repository) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	const query = `SELECT id, name, description, icon, created_at FROM host_groups ORDER BY name`
	var groups []HostGroup
	if err := r.db.SelectContext(ctx, &groups, query); err != nil {
		return nil, fmt.Errorf("list host groups: %w", err)
	}
	if groups == nil {
		groups = []HostGroup{}
	}
	return groups, nil
}

func (r *Repository) CreateHostGroup(ctx context.Context, name string, description *string, icon *string) (HostGroup, error) {
	const query = `INSERT INTO host_groups (name, description, icon) VALUES (?, ?, ?) RETURNING *`
	group := new(HostGroup)
	if err := r.db.GetContext(ctx, group, query, name, description, icon); err != nil {
		if isUniqueViolation(err) {
			return HostGroup{}, ErrHostGroupConflict
		}
		return HostGroup{}, fmt.Errorf("create host group: %w", err)
	}
	return *group, nil
}

func (r *Repository) DeleteHostGroup(ctx context.Context, id HostGroupID) error {
	const query = `DELETE FROM host_groups WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete host group: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrHostGroupNotFound
	}
	return nil
}

type HostGroupWithMembers struct {
	HostGroup
	HostIDs []KnownHostID
}

func (r *Repository) ListHostGroupsWithMembers(ctx context.Context) ([]HostGroupWithMembers, error) {
	type row struct {
		ID          HostGroupID  `db:"id"`
		Name        string       `db:"name"`
		Description *string      `db:"description"`
		Icon        *string      `db:"icon"`
		CreatedAt   time.Time    `db:"created_at"`
		UpdatedAt   time.Time    `db:"updated_at"`
		HostID      *KnownHostID `db:"known_host_id"`
	}
	const query = `
		SELECT hg.id, hg.name, hg.description, hg.icon, hg.created_at, hgm.known_host_id, hg.updated_at
		FROM host_groups hg
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = hg.id
		ORDER BY hg.name, hgm.known_host_id
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("list host groups with members: %w", err)
	}

	seen := make(map[HostGroupID]int)
	var groups []HostGroupWithMembers
	for _, rw := range rows {
		idx, exists := seen[rw.ID]
		if !exists {
			idx = len(groups)
			seen[rw.ID] = idx
			groups = append(groups, HostGroupWithMembers{
				HostGroup: HostGroup{
					ID:          rw.ID,
					Name:        rw.Name,
					Description: rw.Description,
					Icon:        rw.Icon,
					CreatedAt:   rw.CreatedAt,
				},
				HostIDs: []KnownHostID{},
			})
		}
		if rw.HostID != nil {
			groups[idx].HostIDs = append(groups[idx].HostIDs, *rw.HostID)
		}
	}
	return groups, nil
}

func (r *Repository) UpdateHostGroup(ctx context.Context, id HostGroupID, name string, description *string, icon *string) (HostGroup, error) {
	const query = `
		UPDATE host_groups SET name = ?, description = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		RETURNING *
	`
	group := new(HostGroup)
	if err := r.db.GetContext(ctx, group, query, name, description, icon, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return HostGroup{}, ErrHostGroupNotFound
		}
		if isUniqueViolation(err) {
			return HostGroup{}, ErrHostGroupConflict
		}
		return HostGroup{}, fmt.Errorf("update host group: %w", err)
	}
	return *group, nil
}

// ── Host group members ────────────────────────────────────────────────────────

func (r *Repository) SetHostGroupMembers(ctx context.Context, groupID HostGroupID, hostIDs []KnownHostID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM host_group_members WHERE host_group_id = ?`, groupID); err != nil {
			return fmt.Errorf("clear host group members: %w", err)
		}
		for _, hostID := range hostIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO host_group_members (host_group_id, known_host_id) VALUES (?, ?)`,
				groupID, hostID,
			); err != nil {
				if isFKViolation(err) {
					return ErrReferenceNotFound
				}
				return fmt.Errorf("insert host group member: %w", err)
			}
		}
		return nil
	})
}

func (r *Repository) AddHostToGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error {
	const query = `INSERT INTO host_group_members (host_group_id, known_host_id) VALUES (?, ?)`
	if _, err := r.db.ExecContext(ctx, query, groupID, hostID); err != nil {
		if isUniqueViolation(err) {
			return ErrGrantConflict
		}
		if isFKViolation(err) {
			return ErrReferenceNotFound
		}
		return fmt.Errorf("add host to group: %w", err)
	}
	return nil
}

func (r *Repository) RemoveHostFromGroup(ctx context.Context, groupID HostGroupID, hostID KnownHostID) error {
	const query = `DELETE FROM host_group_members WHERE host_group_id = ? AND known_host_id = ?`
	res, err := r.db.ExecContext(ctx, query, groupID, hostID)
	if err != nil {
		return fmt.Errorf("remove host from group: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrGrantNotFound
	}
	return nil
}

func (r *Repository) ListHostGroupMembers(ctx context.Context, groupID HostGroupID) ([]KnownHost, error) {
	const query = `
		SELECT kh.id, kh.fqdn, kh.icon, kh.created_at
		FROM host_group_members hgm
		JOIN known_hosts kh ON kh.id = hgm.known_host_id
		WHERE hgm.host_group_id = ?
		ORDER BY kh.fqdn
	`
	var hosts []KnownHost
	if err := r.db.SelectContext(ctx, &hosts, query, groupID); err != nil {
		return nil, fmt.Errorf("list host group members: %w", err)
	}
	if hosts == nil {
		hosts = []KnownHost{}
	}
	return hosts, nil
}

// ── User grants ───────────────────────────────────────────────────────────────

func (r *Repository) GrantUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error {
	const query = `INSERT INTO user_allowed_hosts (user_id, known_host_id) VALUES (?, ?)`
	if _, err := r.db.ExecContext(ctx, query, userID, hostID); err != nil {
		if isUniqueViolation(err) {
			return ErrGrantConflict
		}
		if isFKViolation(err) {
			return ErrReferenceNotFound
		}
		return fmt.Errorf("grant user host: %w", err)
	}
	return nil
}

func (r *Repository) RevokeUserHost(ctx context.Context, userID auth.UserID, hostID KnownHostID) error {
	const query = `DELETE FROM user_allowed_hosts WHERE user_id = ? AND known_host_id = ?`
	res, err := r.db.ExecContext(ctx, query, userID, hostID)
	if err != nil {
		return fmt.Errorf("revoke user host: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrGrantNotFound
	}
	return nil
}

func (r *Repository) GrantUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error {
	const query = `INSERT INTO user_allowed_host_groups (user_id, host_group_id) VALUES (?, ?)`
	if _, err := r.db.ExecContext(ctx, query, userID, groupID); err != nil {
		if isUniqueViolation(err) {
			return ErrGrantConflict
		}
		if isFKViolation(err) {
			return ErrReferenceNotFound
		}
		return fmt.Errorf("grant user host group: %w", err)
	}
	return nil
}

func (r *Repository) RevokeUserHostGroup(ctx context.Context, userID auth.UserID, groupID HostGroupID) error {
	const query = `DELETE FROM user_allowed_host_groups WHERE user_id = ? AND host_group_id = ?`
	res, err := r.db.ExecContext(ctx, query, userID, groupID)
	if err != nil {
		return fmt.Errorf("revoke user host group: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrGrantNotFound
	}
	return nil
}

func (r *Repository) ListUserGrants(ctx context.Context, userID auth.UserID) (hosts []KnownHost, groups []HostGroup, err error) {
	const hostsQuery = `
		SELECT kh.id, kh.fqdn, kh.icon, kh.created_at
		FROM user_allowed_hosts uah
		JOIN known_hosts kh ON kh.id = uah.known_host_id
		WHERE uah.user_id = ?
		ORDER BY kh.fqdn
	`
	const groupsQuery = `
		SELECT hg.id, hg.name, hg.description, hg.icon, hg.created_at
		FROM user_allowed_host_groups uahg
		JOIN host_groups hg ON hg.id = uahg.host_group_id
		WHERE uahg.user_id = ?
		ORDER BY hg.name
	`
	if err = r.db.SelectContext(ctx, &hosts, hostsQuery, userID); err != nil {
		return nil, nil, fmt.Errorf("list user host grants: %w", err)
	}
	if hosts == nil {
		hosts = []KnownHost{}
	}
	if err = r.db.SelectContext(ctx, &groups, groupsQuery, userID); err != nil {
		return nil, nil, fmt.Errorf("list user group grants: %w", err)
	}
	if groups == nil {
		groups = []HostGroup{}
	}
	return hosts, groups, nil
}

// ── Ignored suggestions ───────────────────────────────────────────────────────

func (r *Repository) AddIgnoredSuggestion(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	const query = `INSERT INTO ignored_host_suggestions (fqdn) VALUES (?) RETURNING id, fqdn, created_at`
	s := new(IgnoredHostSuggestion)
	if err := r.db.GetContext(ctx, s, query, strings.ToLower(fqdn)); err != nil {
		if isUniqueViolation(err) {
			return IgnoredHostSuggestion{}, ErrSuggestionConflict
		}
		return IgnoredHostSuggestion{}, fmt.Errorf("add ignored suggestion: %w", err)
	}
	return *s, nil
}

func (r *Repository) FindIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) (IgnoredHostSuggestion, error) {
	const query = `SELECT id, fqdn, created_at FROM ignored_host_suggestions WHERE fqdn = ?`
	s := new(IgnoredHostSuggestion)
	if err := r.db.GetContext(ctx, s, query, strings.ToLower(fqdn)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return IgnoredHostSuggestion{}, ErrSuggestionNotFound
		}
		return IgnoredHostSuggestion{}, fmt.Errorf("find ignored suggestion by fqdn: %w", err)
	}
	return *s, nil
}

func (r *Repository) RemoveIgnoredSuggestion(ctx context.Context, id int64) error {
	const query = `DELETE FROM ignored_host_suggestions WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("remove ignored suggestion: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrSuggestionNotFound
	}
	return nil
}

func (r *Repository) ListIgnoredSuggestions(ctx context.Context) ([]IgnoredHostSuggestion, error) {
	const query = `SELECT id, fqdn, created_at FROM ignored_host_suggestions ORDER BY fqdn`
	var suggestions []IgnoredHostSuggestion
	if err := r.db.SelectContext(ctx, &suggestions, query); err != nil {
		return nil, fmt.Errorf("list ignored suggestions: %w", err)
	}
	if suggestions == nil {
		suggestions = []IgnoredHostSuggestion{}
	}
	return suggestions, nil
}

// ── Policy cache feed ─────────────────────────────────────────────────────────

// GetAllUserHostAccess returns a row for every user that has either bypass access
// or at least one allowed host grant. Users with neither are not included; the
// policy layer treats them as deny-all (zero value accumulator).
// TODO: This method should go to the query package since it is cross domain (users table)
func (r *Repository) GetAllUserHostAccess(ctx context.Context) ([]policy.UserHostAccess, error) {
	// Each row represents one (user, fqdn) pair, or one (user, NULL) for bypass-only users.
	// UNION removes duplicates that arise when a host is granted both directly and via group.
	const query = `
		SELECT u.id AS user_id, u.bypass_host_allowlist, kh.fqdn AS fqdn
		FROM users u
		JOIN user_allowed_hosts uah ON uah.user_id = u.id
		JOIN known_hosts kh ON kh.id = uah.known_host_id
		WHERE u.deleted_at IS NULL

		UNION

		SELECT u.id AS user_id, u.bypass_host_allowlist, kh.fqdn AS fqdn
		FROM users u
		JOIN user_allowed_host_groups uahg ON uahg.user_id = u.id
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN known_hosts kh ON kh.id = hgm.known_host_id
		WHERE u.deleted_at IS NULL

		UNION

		SELECT u.id AS user_id, u.bypass_host_allowlist, NULL AS fqdn
		FROM users u
		WHERE u.bypass_host_allowlist = 1 AND u.deleted_at IS NULL
	`

	type row struct {
		UserID          auth.UserID `db:"user_id"`
		BypassAllowlist bool        `db:"bypass_host_allowlist"`
		FQDN            *string     `db:"fqdn"`
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get all user host access: %w", err)
	}

	byUser := make(map[auth.UserID]*policy.UserHostAccess, len(rows))
	for _, row := range rows {
		acc, exists := byUser[row.UserID]
		if !exists {
			acc = &policy.UserHostAccess{
				UserID:          row.UserID,
				BypassAllowlist: row.BypassAllowlist,
			}
			byUser[row.UserID] = acc
		}
		if row.FQDN != nil {
			acc.AllowedHosts = append(acc.AllowedHosts, *row.FQDN)
		}
	}

	result := make([]policy.UserHostAccess, 0, len(byUser))
	for _, acc := range byUser {
		result = append(result, *acc)
	}
	return result, nil
}

func (r *Repository) GetUserBypassAllowlist(ctx context.Context, userID auth.UserID) (bool, error) {
	const query = `SELECT bypass_host_allowlist FROM users WHERE id = ? AND deleted_at IS NULL`
	var bypass bool
	if err := r.db.GetContext(ctx, &bypass, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, auth.ErrUserNotFound
		}
		return false, fmt.Errorf("get user bypass allowlist: %w", err)
	}
	return bypass, nil
}

func (r *Repository) SetUserBypassAllowlist(ctx context.Context, userID auth.UserID, bypass bool) error {
	const query = `UPDATE users SET bypass_host_allowlist = ? WHERE id = ? AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, bypass, userID)
	if err != nil {
		return fmt.Errorf("set user bypass allowlist: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return auth.ErrUserNotFound
	}
	return nil
}

func (r *Repository) SetUserGrants(ctx context.Context, userID auth.UserID, hostIDs []KnownHostID, groupIDs []HostGroupID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM user_allowed_hosts WHERE user_id = ?`, userID); err != nil {
			return fmt.Errorf("clear user host grants: %w", err)
		}
		if _, err := r.db.ExecContext(ctx, `DELETE FROM user_allowed_host_groups WHERE user_id = ?`, userID); err != nil {
			return fmt.Errorf("clear user group grants: %w", err)
		}
		for _, hostID := range hostIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO user_allowed_hosts (user_id, known_host_id) VALUES (?, ?)`,
				userID, hostID,
			); err != nil {
				if isFKViolation(err) {
					return ErrReferenceNotFound
				}
				return fmt.Errorf("insert user host grant: %w", err)
			}
		}
		for _, groupID := range groupIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO user_allowed_host_groups (user_id, host_group_id) VALUES (?, ?)`,
				userID, groupID,
			); err != nil {
				if isFKViolation(err) {
					return ErrReferenceNotFound
				}
				return fmt.Errorf("insert user group grant: %w", err)
			}
		}
		return nil
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func isFKViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed")
}
