package networkpolicies

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// Repository owns all DB access for network policies.
type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

// dbNetworkPolicy is the raw DB row for network_policies.
type dbNetworkPolicy struct {
	ID            NetworkPolicyID `db:"id"`
	Name          string          `db:"name"`
	CIDR          string          `db:"cidr"`
	Description   *string         `db:"description"`
	Enabled       bool            `db:"enabled"`
	AllowAllHosts bool            `db:"allow_all_hosts"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}

func (r dbNetworkPolicy) toPolicy() NetworkPolicy {
	return NetworkPolicy{
		ID:            r.ID,
		Name:          r.Name,
		CIDR:          r.CIDR,
		Description:   r.Description,
		Enabled:       r.Enabled,
		AllowAllHosts: r.AllowAllHosts,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

func (r *Repository) CreatePolicy(ctx context.Context, p NetworkPolicy) (NetworkPolicy, error) {
	const query = `
		INSERT INTO network_policies (name, cidr, description, enabled, allow_all_hosts)
		VALUES (?, ?, ?, ?, ?)
		RETURNING *
	`
	var row dbNetworkPolicy
	err := r.db.GetContext(ctx, &row, query,
		p.Name, p.CIDR, p.Description, p.Enabled, p.AllowAllHosts,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return NetworkPolicy{}, ErrCIDRConflict
		}
		return NetworkPolicy{}, fmt.Errorf("create network policy: %w", err)
	}
	return row.toPolicy(), nil
}

func (r *Repository) GetPolicy(ctx context.Context, id NetworkPolicyID) (*NetworkPolicy, error) {
	const query = `
		SELECT *
		FROM network_policies
		WHERE id = ?
	`
	var row dbNetworkPolicy
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get network policy: %w", err)
	}
	return new(row.toPolicy()), nil
}

func (r *Repository) UpdatePolicy(ctx context.Context, p NetworkPolicy) (*NetworkPolicy, error) {
	const query = `
		UPDATE network_policies
		SET name = ?, cidr = ?, description = ?, enabled = ?, allow_all_hosts = ?,
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?
		RETURNING id, name, cidr, description, enabled, allow_all_hosts, created_at, updated_at
	`
	var row dbNetworkPolicy
	err := r.db.GetContext(ctx, &row, query,
		p.Name, p.CIDR, p.Description, p.Enabled, p.AllowAllHosts, p.ID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isUniqueConstraint(err) {
			return nil, ErrCIDRConflict
		}
		return nil, fmt.Errorf("update network policy: %w", err)
	}
	result := row.toPolicy()
	return &result, nil
}

func (r *Repository) DeletePolicy(ctx context.Context, id NetworkPolicyID) error {
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

// SetHostAccess atomically replaces the M2M rows and allow_all_hosts flag for this policy.
func (r *Repository) SetHostAccess(ctx context.Context, id NetworkPolicyID, allowAll bool, groupIDs []int64, hostIDs []int64) error {
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
		if _, err := r.db.ExecContext(ctx,
			`DELETE FROM network_policy_allowed_hosts WHERE policy_id = ?`, id); err != nil {
			return fmt.Errorf("delete hosts: %w", err)
		}

		for _, gid := range groupIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO network_policy_allowed_host_groups (policy_id, host_group_id) VALUES (?, ?)`,
				id, gid); err != nil {
				return fmt.Errorf("insert host group: %w", err)
			}
		}
		for _, hid := range hostIDs {
			if _, err := r.db.ExecContext(ctx,
				`INSERT INTO network_policy_allowed_hosts (policy_id, known_host_id) VALUES (?, ?)`,
				id, hid); err != nil {
				return fmt.Errorf("insert host: %w", err)
			}
		}

		if _, err := r.db.ExecContext(ctx,
			`UPDATE network_policies SET allow_all_hosts = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?`,
			allowAll, id); err != nil {
			return fmt.Errorf("update host access: %w", err)
		}
		return nil
	})
}

// GetEnabledCacheEntries returns all enabled policies with their allowed host FQDNs,
// used to populate the policy cache. Kept here as an internal-cache exception despite
// crossing into the hosts domain tables.
func (r *Repository) GetEnabledCacheEntries(ctx context.Context) ([]CacheEntry, error) {
	type policyRow struct {
		PolicyID      NetworkPolicyID `db:"policy_id"`
		PolicyName    string          `db:"policy_name"`
		CIDR          string          `db:"cidr"`
		AllowAllHosts bool            `db:"allow_all_hosts"`
		FQDN          *string         `db:"fqdn"`
	}

	// Returns one row per (policy, fqdn); fqdn is NULL when no hosts assigned.
	const query = `
		SELECT np.id AS policy_id, np.name AS policy_name, np.cidr, np.allow_all_hosts,
		       allowed.fqdn
		FROM network_policies np
		LEFT JOIN (
			SELECT npah.policy_id, kh.fqdn
			FROM network_policy_allowed_hosts npah
			JOIN known_hosts kh ON kh.id = npah.known_host_id
			UNION
			SELECT npahg.policy_id, kh.fqdn
			FROM network_policy_allowed_host_groups npahg
			JOIN host_group_members hgm ON hgm.host_group_id = npahg.host_group_id
			JOIN known_hosts kh ON kh.id = hgm.known_host_id
		) allowed ON allowed.policy_id = np.id
		WHERE np.enabled = 1
		ORDER BY np.id, allowed.fqdn
	`
	var rows []policyRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get enabled cache entries: %w", err)
	}

	seen := make(map[NetworkPolicyID]int)
	var result []CacheEntry
	for _, row := range rows {
		idx, ok := seen[row.PolicyID]
		if !ok {
			idx = len(result)
			seen[row.PolicyID] = idx
			result = append(result, CacheEntry{
				PolicyID:      row.PolicyID,
				PolicyName:    row.PolicyName,
				CIDR:          row.CIDR,
				AllowAllHosts: row.AllowAllHosts,
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
