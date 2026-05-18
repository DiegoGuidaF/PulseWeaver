package hosts

import (
	"context"
	"fmt"
	"strings"

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

// ── Hosts ─────────────────────────────────────────────────────────────────────

func (r *Repository) ListHosts(ctx context.Context) ([]Host, error) {
	const query = `SELECT id, fqdn, updated_at, created_at FROM hosts ORDER BY id`
	var hosts []Host
	if err := r.db.SelectContext(ctx, &hosts, query); err != nil {
		return nil, fmt.Errorf("list hosts: %w", err)
	}
	return hosts, nil
}

func (r *Repository) CreateHost(ctx context.Context, draft HostDraft) (ids.HostID, error) {
	var hostID ids.HostID
	const query = `INSERT INTO hosts (fqdn) VALUES (?) RETURNING id`
	if err := r.db.GetContext(ctx, &hostID, query, draft.FQDN); err != nil {
		if isUniqueViolation(err) {
			return hostID, ErrHostConflict
		}
		return hostID, fmt.Errorf("create host %q: %w", draft.FQDN, err)
	}
	return hostID, nil
}

func (r *Repository) DeleteHost(ctx context.Context, id ids.HostID) error {
	const query = `DELETE FROM hosts WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete host: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrHostNotFound
	}
	return nil
}

func (r *Repository) ListHostsByIDs(ctx context.Context, hostIDs []ids.HostID) ([]Host, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`
		SELECT id, fqdn, updated_at, created_at
		FROM hosts
		WHERE id IN (?)
	`, hostIDs)
	if err != nil {
		return nil, fmt.Errorf("build list hosts query: %w", err)
	}
	query = r.db.Rebind(query)
	var hosts []Host
	if err := r.db.SelectContext(ctx, &hosts, query, args...); err != nil {
		return nil, fmt.Errorf("list hosts by ids: %w", err)
	}
	return hosts, nil
}

// SetHostGroupMembership replaces the full set of groups that hostID belongs to.
func (r *Repository) SetHostGroupMembership(ctx context.Context, hostID ids.HostID, groupIDs []ids.HostGroupID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.ExecContext(ctx,
			`DELETE FROM host_group_members WHERE host_id = ?`, hostID,
		); err != nil {
			return fmt.Errorf("clear host group memberships for host %d: %w", hostID, err)
		}
		for _, groupID := range groupIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO host_group_members (host_group_id, host_id) VALUES (?, ?)`,
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
// members still appears in the result.
func (r *Repository) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	var groups []HostGroup
	const groupsQuery = `SELECT id, name, description, icon, color FROM host_groups`
	if err := r.db.SelectContext(ctx, &groups, groupsQuery); err != nil {
		return nil, fmt.Errorf("list host groups: %w", err)
	}

	type memberRow struct {
		GroupID ids.HostGroupID `db:"host_group_id"`
		HostID  ids.HostID      `db:"host_id"`
	}
	var rows []memberRow
	const membersQuery = `SELECT host_group_id, host_id FROM host_group_members`
	if err := r.db.SelectContext(ctx, &rows, membersQuery); err != nil {
		return nil, fmt.Errorf("list host group members: %w", err)
	}

	hostIDsByGroup := make(map[ids.HostGroupID][]ids.HostID, len(groups))
	for _, row := range rows {
		hostIDsByGroup[row.GroupID] = append(hostIDsByGroup[row.GroupID], row.HostID)
	}
	for i := range groups {
		groups[i].HostIDs = hostIDsByGroup[groups[i].ID]
	}
	return groups, nil
}

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

func (r *Repository) UpdateHostGroup(ctx context.Context, group HostGroup) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		const query = `
			UPDATE host_groups
			SET name = ?, description = ?, icon = ?, color = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		res, err := r.db.ExecContext(ctx, query,
			group.Name, group.Description, group.Icon, group.Color, group.ID,
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

func (r *Repository) replaceHostGroupMembers(ctx context.Context, groupID ids.HostGroupID, hostIDs []ids.HostID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := r.db.ExecContext(ctx, `DELETE FROM host_group_members WHERE host_group_id = ?`, groupID); err != nil {
			return fmt.Errorf("clear host group members: %w", err)
		}
		for _, hostID := range hostIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO host_group_members (host_group_id, host_id) VALUES (?, ?)`,
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

// ── helpers ───────────────────────────────────────────────────────────────────

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func isFKViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed")
}
