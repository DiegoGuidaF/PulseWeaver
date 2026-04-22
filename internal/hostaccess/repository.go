package hostaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) BulkCreateKnownHosts(ctx context.Context, fqdns []string) ([]KnownHost, error) {
	hosts := make([]KnownHost, 0, len(fqdns))
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		const query = `INSERT INTO known_hosts (fqdn) VALUES (?) RETURNING *`
		for _, fqdn := range fqdns {
			host := new(KnownHost)
			if err := r.db.GetContext(ctx, host, query, strings.ToLower(fqdn)); err != nil {
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

func (r *Repository) CreateHostGroupWithMembers(ctx context.Context, name string, description *string, icon *string, hostIDs []KnownHostID) (HostGroupID, error) {
	var groupID HostGroupID
	err := r.db.WithinTx(ctx, func(ctx context.Context) error {
		const insertGroup = `INSERT INTO host_groups (name, description, icon) VALUES (?, ?, ?) RETURNING id`
		if err := r.db.GetContext(ctx, &groupID, insertGroup, name, description, icon); err != nil {
			if isUniqueViolation(err) {
				return ErrHostGroupConflict
			}
			return fmt.Errorf("create host group: %w", err)
		}
		return r.setHostGroupMembers(ctx, groupID, hostIDs)
	})
	return groupID, err
}

func (r *Repository) UpdateHostGroupWithMembers(ctx context.Context, id HostGroupID, name string, description *string, icon *string, hostIDs []KnownHostID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		const updateMeta = `
			UPDATE host_groups SET name = ?, description = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		res, err := r.db.ExecContext(ctx, updateMeta, name, description, icon, id)
		if err != nil {
			if isUniqueViolation(err) {
				return ErrHostGroupConflict
			}
			return fmt.Errorf("update host group: %w", err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return ErrHostGroupNotFound
		}
		return r.setHostGroupMembers(ctx, id, hostIDs)
	})
}

func (r *Repository) UpdateHostGroupMetadata(ctx context.Context, id HostGroupID, name string, description *string, icon *string) error {
	const query = `
		UPDATE host_groups SET name = ?, description = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query, name, description, icon, id)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrHostGroupConflict
		}
		return fmt.Errorf("update host group metadata: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrHostGroupNotFound
	}
	return nil
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

func (r *Repository) setHostGroupMembers(ctx context.Context, groupID HostGroupID, hostIDs []KnownHostID) error {
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
}

// ── User grants ───────────────────────────────────────────────────────────────

func (r *Repository) SetFullUserGrants(ctx context.Context, userID auth.UserID, bypass *bool, hostIDs []KnownHostID, groupIDs []HostGroupID) error {
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
		if bypass != nil {
			const upsert = `INSERT OR REPLACE INTO user_host_settings (user_id, bypass_host_allowlist) VALUES (?, ?)`
			if _, err := r.db.ExecContext(ctx, upsert, userID, *bypass); err != nil {
				if isFKViolation(err) {
					return auth.ErrUserNotFound
				}
				return fmt.Errorf("set user bypass allowlist: %w", err)
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

// ── User lifecycle ────────────────────────────────────────────────────────────

func (r *Repository) EnsureUserSettings(ctx context.Context, userID auth.UserID) error {
	const query = `INSERT OR IGNORE INTO user_host_settings (user_id, bypass_host_allowlist) VALUES (?, 0)`
	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("ensure user host settings: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUserData(ctx context.Context, userID auth.UserID) error {
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

func (r *Repository) GetAllUserHostAccess(ctx context.Context) ([]policy.UserHostAccess, error) {
	const query = `
		SELECT uhs.user_id, uhs.bypass_host_allowlist, kh.fqdn AS fqdn
		FROM user_host_settings uhs
		JOIN user_allowed_hosts uah ON uah.user_id = uhs.user_id
		JOIN known_hosts kh ON kh.id = uah.known_host_id

		UNION

		SELECT uhs.user_id, uhs.bypass_host_allowlist, kh.fqdn AS fqdn
		FROM user_host_settings uhs
		JOIN user_allowed_host_groups uahg ON uahg.user_id = uhs.user_id
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN known_hosts kh ON kh.id = hgm.known_host_id

		UNION

		SELECT uhs.user_id, uhs.bypass_host_allowlist, NULL AS fqdn
		FROM user_host_settings uhs
		WHERE uhs.bypass_host_allowlist = 1
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

// ── helpers ───────────────────────────────────────────────────────────────────

func isUniqueViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}

func isFKViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed")
}
