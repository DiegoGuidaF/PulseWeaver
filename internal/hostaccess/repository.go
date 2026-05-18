package hostaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// ── Known hosts ───────────────────────────────────────────────────────────────

// ListKnownHosts returns all known hosts ordered by ID.
func (r *Repository) ListKnownHosts(ctx context.Context) ([]KnownHost, error) {
	const query = `SELECT id, fqdn, icon, updated_at, created_at FROM known_hosts ORDER BY id`
	var hosts []KnownHost
	if err := r.db.SelectContext(ctx, &hosts, query); err != nil {
		return nil, fmt.Errorf("list known hosts: %w", err)
	}
	return hosts, nil
}

// CreateKnownHost inserts a single known host row. Translates unique-violation
// to ErrKnownHostConflict.
func (r *Repository) CreateKnownHost(ctx context.Context, draft KnownHostDraft) (ids.KnownHostID, error) {
	var knownHostID ids.KnownHostID

	const query = `INSERT INTO known_hosts (fqdn, icon) VALUES (?, ?) returning id`
	if err := r.db.GetContext(ctx, &knownHostID, query, draft.FQDN, draft.Icon); err != nil {
		if isUniqueViolation(err) {
			return knownHostID, ErrKnownHostConflict
		}
		return knownHostID, fmt.Errorf("create known host %q: %w", draft.FQDN, err)
	}
	return knownHostID, nil
}

func (r *Repository) UpdateKnownHost(ctx context.Context, id ids.KnownHostID, icon *string) (KnownHost, error) {
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

func (r *Repository) DeleteKnownHost(ctx context.Context, id ids.KnownHostID) error {
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

func (r *Repository) ListKnownHostsByIDs(ctx context.Context, ids []ids.KnownHostID) ([]KnownHost, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, fqdn, icon, updated_at, created_at
		FROM known_hosts
		WHERE id IN (?)
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("build list known hosts query: %w", err)
	}
	query = r.db.Rebind(query)

	var hosts []KnownHost
	if err := r.db.SelectContext(ctx, &hosts, query, args...); err != nil {
		return nil, fmt.Errorf("list known hosts by ids: %w", err)
	}
	return hosts, nil
}

// SetKnownHostGroupMembership replaces the full set of groups that hostID
// belongs to. It deletes all existing host_group_members rows for hostID then
// inserts one row per groupID. An unknown groupID is mapped to ErrReferenceNotFound.
func (r *Repository) SetKnownHostGroupMembership(ctx context.Context, hostID ids.KnownHostID, groupIDs []ids.HostGroupID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.ExecContext(ctx,
			`DELETE FROM host_group_members WHERE known_host_id = ?`, hostID,
		); err != nil {
			return fmt.Errorf("clear host group memberships for host %d: %w", hostID, err)
		}
		for _, groupID := range groupIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO host_group_members (host_group_id, known_host_id) VALUES (?, ?)`,
				groupID, hostID,
			); err != nil {
				if isFKViolation(err) {
					return ErrReferenceNotFound
				}
				return fmt.Errorf("insert host group membership (host=%d group=%d): %w", hostID, groupID, err)
			}
		}
		return nil
	})
}

// ── Host groups ───────────────────────────────────────────────────────────────

// ListHostGroups returns every host group with its member host IDs. Members
// are loaded with a second query and joined in memory so a group with no
// members still appears in the result (a single LEFT JOIN with GROUP BY is
// possible but harder to map back to typed slices via sqlx).
func (r *Repository) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	var groups []HostGroup
	const groupsQuery = `
		SELECT id, name, description, icon, color
		FROM host_groups
	`
	if err := r.db.SelectContext(ctx, &groups, groupsQuery); err != nil {
		return nil, fmt.Errorf("list host groups: %w", err)
	}

	type memberRow struct {
		GroupID ids.HostGroupID `db:"host_group_id"`
		HostID  ids.KnownHostID `db:"known_host_id"`
	}

	var rows []memberRow
	const membersQuery = `
		SELECT host_group_id, known_host_id
		FROM host_group_members
	`
	if err := r.db.SelectContext(ctx, &rows, membersQuery); err != nil {
		return nil, fmt.Errorf("list host group members: %w", err)
	}

	hostIDsByGroup := make(map[ids.HostGroupID][]ids.KnownHostID, len(groups))
	for _, row := range rows {
		hostIDsByGroup[row.GroupID] = append(hostIDsByGroup[row.GroupID], row.HostID)
	}
	for i := range groups {
		groups[i].HostIDs = hostIDsByGroup[groups[i].ID]
	}
	return groups, nil
}

// CreateHostGroup inserts a new host group together with its members in a
// single transaction. Intended to be called from inside a reconcile flow that
// already opened a transaction; in that case WithinTx reuses it.
//
// CAUTION: when several CreateHostGroup / UpdateHostGroup calls run in the
// same reconcile transaction, the host_groups.name unique index is checked at
// each statement. Two existing groups swapping names would therefore fail on
// the first UPDATE. The reconcile orchestrator deletes before updating before
// creating to side-step the common cases, but a true rename-swap still needs
// a two-phase rename-via-temp that is not implemented yet.
func (r *Repository) CreateHostGroup(ctx context.Context, draft HostGroupDraft) (ids.HostGroupID, error) {
	var groupID ids.HostGroupID
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		const insertGroup = `
			INSERT INTO host_groups (name, description, icon, color)
			VALUES (?, ?, ?, ?)
			RETURNING id
		`
		if err := r.db.GetContext(ctx, &groupID, insertGroup, draft.Name, draft.Description, draft.Icon, draft.Color); err != nil {
			if isUniqueViolation(err) {
				return ErrHostGroupConflict
			}
			return fmt.Errorf("create host group: %w", err)
		}
		return r.replaceHostGroupMembers(ctx, groupID, draft.HostIDs)
	})
	return groupID, err
}

// UpdateHostGroup replaces a group's metadata and its member set. See
// CreateHostGroup for the rename-swap caveat.
func (r *Repository) UpdateHostGroup(ctx context.Context, group HostGroup) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		const query = `
			UPDATE host_groups
			SET name = ?, description = ?, icon = ?, color = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		res, err := r.db.ExecContext(ctx, query,
			group.Name,
			group.Description,
			group.Icon,
			group.Color,
			group.ID,
		)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrHostGroupConflict
			}
			return fmt.Errorf("update host group: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("update host group rows affected: %w", err)
		}
		if rows == 0 {
			return ErrHostGroupNotFound
		}

		return r.replaceHostGroupMembers(ctx, group.ID, group.HostIDs)
	})
}

func (r *Repository) DeleteHostGroup(ctx context.Context, id ids.HostGroupID) error {
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

func (r *Repository) replaceHostGroupMembers(ctx context.Context, groupID ids.HostGroupID, hostIDs []ids.KnownHostID) error {
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

// ── User grants ───────────────────────────────────────────────────────────────

func (r *Repository) SetUserAccess(ctx context.Context, userID ids.UserID, bypassHostCheck bool, groupIDs []ids.HostGroupID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM user_allowed_host_groups WHERE user_id = ?`, userID); err != nil {
			return fmt.Errorf("clear user group grants: %w", err)
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
		const upsert = `INSERT OR REPLACE INTO user_host_settings (user_id, bypass_host_allowlist) VALUES (?, ?)`
		if _, err := r.db.ExecContext(ctx, upsert, userID, bypassHostCheck); err != nil {
			if isFKViolation(err) {
				return auth.ErrUserNotFound
			}
			return fmt.Errorf("set user bypass allowlist: %w", err)
		}
		return nil
	})
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

func (r *Repository) RemoveIgnoredSuggestionByFQDN(ctx context.Context, fqdn string) error {
	const query = `DELETE FROM ignored_host_suggestions WHERE fqdn = ?`
	res, err := r.db.ExecContext(ctx, query, strings.ToLower(fqdn))
	if err != nil {
		return fmt.Errorf("remove ignored suggestion: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrSuggestionNotFound
	}
	return nil
}

// ── User lifecycle ────────────────────────────────────────────────────────────

func (r *Repository) EnsureUserSettings(ctx context.Context, userID ids.UserID) error {
	const query = `INSERT OR IGNORE INTO user_host_settings (user_id, bypass_host_allowlist) VALUES (?, 0)`
	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("ensure user host settings: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUserData(ctx context.Context, userID ids.UserID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		for _, q := range []string{
			`DELETE FROM user_host_settings WHERE user_id = ?`,
			`DELETE FROM user_allowed_hosts WHERE user_id = ?`,
			`DELETE FROM user_allowed_host_groups WHERE user_id = ?`,
		} {
			if _, err := r.db.ExecContext(ctx, q, userID); err != nil {
				return fmt.Errorf("delete user host data: %w", err)
			}
		}
		return nil
	})
}

// ── Policy cache feed ─────────────────────────────────────────────────────────

func (r *Repository) GetAllUserHostSettings(ctx context.Context) ([]UserHostSetting, error) {
	const query = `SELECT user_id, bypass_host_allowlist FROM user_host_settings`
	var settings []UserHostSetting
	if err := r.db.SelectContext(ctx, &settings, query); err != nil {
		return nil, fmt.Errorf("get all user host settings: %w", err)
	}
	return settings, nil
}

func (r *Repository) GetAllUserDirectHostGrants(ctx context.Context) ([]UserHostGrant, error) {
	const query = `
		SELECT uah.user_id, kh.fqdn
		FROM user_allowed_hosts uah
		JOIN known_hosts kh ON kh.id = uah.known_host_id
	`
	var grants []UserHostGrant
	if err := r.db.SelectContext(ctx, &grants, query); err != nil {
		return nil, fmt.Errorf("get all user direct host grants: %w", err)
	}
	return grants, nil
}

func (r *Repository) GetAllUserGroupHostGrants(ctx context.Context) ([]UserHostGrant, error) {
	const query = `
		SELECT uahg.user_id, kh.fqdn
		FROM user_allowed_host_groups uahg
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN known_hosts kh ON kh.id = hgm.known_host_id
	`
	var grants []UserHostGrant
	if err := r.db.SelectContext(ctx, &grants, query); err != nil {
		return nil, fmt.Errorf("get all user group host grants: %w", err)
	}
	return grants, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func isFKViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed")
}
