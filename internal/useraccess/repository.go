package useraccess

import (
	"context"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

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
		const upsert = `INSERT OR REPLACE INTO user_host_settings (user_id, bypass_host_check) VALUES (?, ?)`
		if _, err := r.db.ExecContext(ctx, upsert, userID, bypassHostCheck); err != nil {
			if isFKViolation(err) {
				return auth.ErrUserNotFound
			}
			return fmt.Errorf("set user bypass host check: %w", err)
		}
		return nil
	})
}

func (r *Repository) EnsureUserSettings(ctx context.Context, userID ids.UserID) error {
	const query = `INSERT OR IGNORE INTO user_host_settings (user_id, bypass_host_check) VALUES (?, 0)`
	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("ensure user host settings: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUserData(ctx context.Context, userID ids.UserID) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		for _, q := range []string{
			`DELETE FROM user_host_settings WHERE user_id = ?`,
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
	const query = `SELECT user_id, bypass_host_check FROM user_host_settings`
	var settings []UserHostSetting
	if err := r.db.SelectContext(ctx, &settings, query); err != nil {
		return nil, fmt.Errorf("get all user host settings: %w", err)
	}
	return settings, nil
}

// GetAllUserHostGrants resolves every (user, fqdn) pair reachable through
// user_allowed_host_groups → host_group_members → hosts.
//
// NOTE: this query intentionally crosses into the hosts domain tables
// (hosts, host_group_members), which violates the cross-domain-queries pattern
// (docs/patterns/backend/cross-domain-queries.md). It is kept here because this
// data is tightly coupled to the policy cache lifecycle. If this query grows or
// is reused outside of cache population, move it to internal/queries/.
func (r *Repository) GetAllUserHostGrants(ctx context.Context) ([]UserHostGrant, error) {
	const query = `
		SELECT uahg.user_id, h.fqdn
		FROM user_allowed_host_groups uahg
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN hosts h ON h.id = hgm.host_id
	`
	var grants []UserHostGrant
	if err := r.db.SelectContext(ctx, &grants, query); err != nil {
		return nil, fmt.Errorf("get all user group host grants: %w", err)
	}
	return grants, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func isFKViolation(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "foreign key constraint failed")
}
