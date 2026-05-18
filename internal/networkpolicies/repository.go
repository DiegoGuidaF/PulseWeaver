package networkpolicies

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// Repository owns all DB access for network policies.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

//// dbNetworkPolicy is the raw DB row for network_policies.
//type dbNetworkPolicy struct {
//	ID              ids.NetworkPolicyID `db:"id"`
//	Name            string              `db:"name"`
//	CIDR            string              `db:"cidr"`
//	Description     *string             `db:"description"`
//	Enabled         bool                `db:"enabled"`
//	BypassHostCheck bool                `db:"bypass_host_check"`
//	CreatedAt       time.Time           `db:"created_at"`
//	UpdatedAt       time.Time           `db:"updated_at"`
//}
//
//func (r dbNetworkPolicy) toPolicy() NetworkPolicy {
//	return NetworkPolicy{
//		ID:              r.ID,
//		Name:            r.Name,
//		CIDR:            r.CIDR,
//		Description:     r.Description,
//		Enabled:         r.Enabled,
//		BypassHostCheck: r.BypassHostCheck,
//		CreatedAt:       r.CreatedAt,
//		UpdatedAt:       r.UpdatedAt,
//	}
//}

func (r *Repository) CreatePolicy(ctx context.Context, p NetworkPolicy) (NetworkPolicy, error) {
	const query = `
		INSERT INTO network_policies (name, cidr, description, enabled, bypass_host_check)
		VALUES (?, ?, ?, ?, ?)
		RETURNING *
	`
	var row NetworkPolicy
	err := r.db.GetContext(ctx, &row, query,
		p.Name, p.CIDR, p.Description, p.Enabled, p.BypassHostCheck,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return NetworkPolicy{}, ErrCIDRConflict
		}
		return NetworkPolicy{}, fmt.Errorf("create network policy: %w", err)
	}
	return row, nil
}

func (r *Repository) GetPolicy(ctx context.Context, id ids.NetworkPolicyID) (NetworkPolicy, error) {
	const query = `
		SELECT *
		FROM network_policies
		WHERE id = ?
	`
	var row NetworkPolicy
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NetworkPolicy{}, ErrNotFound
		}
		return NetworkPolicy{}, fmt.Errorf("get network policy: %w", err)
	}
	return row, nil
}

func (r *Repository) UpdatePolicy(ctx context.Context, p NetworkPolicy) (NetworkPolicy, error) {
	const query = `
		UPDATE network_policies
		SET name = ?, cidr = ?, description = ?, enabled = ?, bypass_host_check = ?,
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?
		RETURNING id, name, cidr, description, enabled, bypass_host_check, created_at, updated_at
	`
	var row NetworkPolicy
	err := r.db.GetContext(ctx, &row, query,
		p.Name, p.CIDR, p.Description, p.Enabled, p.BypassHostCheck, p.ID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NetworkPolicy{}, ErrNotFound
		}
		if isUniqueConstraint(err) {
			return NetworkPolicy{}, ErrCIDRConflict
		}
		return NetworkPolicy{}, fmt.Errorf("update network policy: %w", err)
	}
	return row, nil
}

func (r *Repository) DeletePolicy(ctx context.Context, id ids.NetworkPolicyID) error {
	const query = `DELETE FROM network_policies WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete network policy: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetHostAccess atomically replaces the M2M rows and bypass_host_check flag for this policy.
func (r *Repository) SetHostAccess(
	ctx context.Context,
	id ids.NetworkPolicyID,
	bypassHostCheck bool,
	groupIDs []ids.HostGroupID,
) error {
	return r.db.WithinTx(ctx, func(ctx context.Context) error {
		var exists bool
		if err := r.db.GetContext(ctx, &exists,
			`SELECT EXISTS(SELECT 1 FROM network_policies WHERE id = ?)`, id); err != nil {
			return fmt.Errorf("check policy existence: %w", err)
		}
		if !exists {
			return ErrNotFound
		}

		if _, err := r.db.ExecContext(ctx,
			`DELETE FROM network_policy_allowed_host_groups WHERE policy_id = ?`, id); err != nil {
			return fmt.Errorf("delete host groups: %w", err)
		}

		for _, gid := range groupIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO network_policy_allowed_host_groups (policy_id, host_group_id) VALUES (?, ?)`,
				id, gid); err != nil {
				return fmt.Errorf("insert host group: %w", err)
			}
		}

		if _, err := r.db.ExecContext(ctx,
			`UPDATE network_policies SET bypass_host_check = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
			bypassHostCheck, id); err != nil {
			return fmt.Errorf("update host access: %w", err)
		}
		return nil
	})
}

// GetEnabledCacheEntries returns all enabled policies with their allowed host FQDNs,
// used to populate the policy cache.
//
// NOTE: this query intentionally crosses into the hosts domain tables
// (hosts, host_group_members), which violates the cross-domain-queries pattern
// (docs/patterns/backend/cross-domain-queries.md). It is kept here because this
// data is tightly coupled to the policy cache lifecycle, which lives in this package.
// If this query grows or is reused outside of cache population, move it to internal/queries/.
func (r *Repository) GetEnabledCacheEntries(ctx context.Context) ([]CacheEntry, error) {
	type policyRow struct {
		PolicyID        ids.NetworkPolicyID `db:"policy_id"`
		PolicyName      string              `db:"policy_name"`
		CIDR            string              `db:"cidr"`
		BypassHostCheck bool                `db:"bypass_host_check"`
		FQDN            *string             `db:"fqdn"`
	}

	// Returns one row per (policy, fqdn); fqdn is NULL when no host groups assigned.
	const query = `
		SELECT np.id AS policy_id, np.name AS policy_name, np.cidr, np.bypass_host_check,
		       h.fqdn
		FROM network_policies np
		LEFT JOIN network_policy_allowed_host_groups npahg ON npahg.policy_id = np.id
		LEFT JOIN host_group_members hgm ON hgm.host_group_id = npahg.host_group_id
		LEFT JOIN hosts h ON h.id = hgm.host_id
		WHERE np.enabled = 1
		ORDER BY np.id, h.fqdn
	`
	var rows []policyRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get enabled cache entries: %w", err)
	}

	seen := make(map[ids.NetworkPolicyID]int)
	var result []CacheEntry
	for _, row := range rows {
		idx, ok := seen[row.PolicyID]
		if !ok {
			idx = len(result)
			seen[row.PolicyID] = idx
			result = append(result, CacheEntry{
				PolicyID:        row.PolicyID,
				PolicyName:      row.PolicyName,
				CIDR:            row.CIDR,
				BypassHostCheck: row.BypassHostCheck,
			})
		}
		if row.FQDN != nil {
			result[idx].AllowedHostFQDNs = append(result[idx].AllowedHostFQDNs, *row.FQDN)
		}
	}
	return result, nil
}

// isUniqueConstraint returns true for SQLite UNIQUE constraint violations.
func isUniqueConstraint(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
