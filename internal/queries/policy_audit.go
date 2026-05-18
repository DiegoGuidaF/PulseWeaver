package queries

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/jmoiron/sqlx"
)

// AuditNetworkPoliciesProvider is the interface the queries package consumes to
// load network policy data for the policy audit view. Implemented by
// networkpolicies.Repository.
type AuditNetworkPoliciesProvider interface {
	GetEnabledCacheEntries(ctx context.Context) ([]networkpolicies.CacheEntry, error)
}

// PolicyMapReader is the cross-domain interface consumed by the queries package.
// Implemented by policy.Service.
type PolicyMapReader interface {
	GetPolicyMap() policy.PolicyMapSnapshot
}

// policyEnrichmentRow holds SQL-joined metadata for a single address contributor.
type policyEnrichmentRow struct {
	AddressID        ids.AddressID `db:"address_id"`
	AddressUpdatedAt time.Time     `db:"address_updated_at"`
	DeviceID         ids.DeviceID  `db:"device_id"`
	DeviceName       string        `db:"device_name"`
	UserID           ids.UserID    `db:"user_id"`
	UserName         string        `db:"user_name"`
}

// getPolicyAddressEnrichment fetches display metadata for the given address IDs.
func (r *Repository) getPolicyAddressEnrichment(ctx context.Context, addressIDs []ids.AddressID) (map[ids.AddressID]policyEnrichmentRow, error) {
	if len(addressIDs) == 0 {
		return map[ids.AddressID]policyEnrichmentRow{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT
			a.id           AS address_id,
			a.updated_at   AS address_updated_at,
			d.id           AS device_id,
			d.name         AS device_name,
			u.id           AS user_id,
			u.display_name AS user_name
		FROM addresses a
		JOIN devices d ON d.id = a.device_id
		JOIN users u ON u.id = d.owner_id
		WHERE a.id IN (?)`, addressIDs)
	if err != nil {
		return nil, fmt.Errorf("build policy audit enrichment query: %w", err)
	}
	query = r.db.Rebind(query)

	var rows []policyEnrichmentRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get policy address enrichment: %w", err)
	}

	result := make(map[ids.AddressID]policyEnrichmentRow, len(rows))
	for _, row := range rows {
		result[row.AddressID] = row
	}
	return result, nil
}

// getAllUsersForPolicyAudit returns every non-deleted user with their bypass flag,
// plus a map of pre-intersection allowed FQDNs keyed by user ID.
// Two queries assembled in Go per queries-read-models.md pattern.
func (r *Repository) getAllUsersForPolicyAudit(ctx context.Context) ([]policyAuditUserRow, map[ids.UserID][]string, error) {
	const usersQuery = `
		SELECT u.id           AS user_id,
		       u.username     AS username,
		       u.display_name AS user_name,
		       u.role IN ('admin', 'superadmin') AS is_admin,
		       COALESCE(uhs.bypass_host_allowlist, 0) AS bypass_allowlist
		FROM users u
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		WHERE u.deleted_at IS NULL
		ORDER BY u.display_name, u.id
	`
	var userRows []policyAuditUserRow
	if err := r.db.SelectContext(ctx, &userRows, usersQuery); err != nil {
		return nil, nil, fmt.Errorf("list users for policy audit: %w", err)
	}

	const hostsQuery = `
		SELECT uah.user_id, kh.fqdn
		FROM user_allowed_hosts uah
		JOIN known_hosts kh ON kh.id = uah.known_host_id
		UNION
		SELECT uahg.user_id, kh.fqdn
		FROM user_allowed_host_groups uahg
		JOIN host_group_members hgm ON hgm.host_group_id = uahg.host_group_id
		JOIN known_hosts kh ON kh.id = hgm.known_host_id
		ORDER BY 1, 2
	`

	type hostRow struct {
		UserID ids.UserID `db:"user_id"`
		FQDN   string     `db:"fqdn"`
	}
	var hostRows []hostRow
	if err := r.db.SelectContext(ctx, &hostRows, hostsQuery); err != nil {
		return nil, nil, fmt.Errorf("list user allowed hosts for policy audit: %w", err)
	}

	allowedHostsByUser := make(map[ids.UserID][]string, len(userRows))
	for _, h := range hostRows {
		allowedHostsByUser[h.UserID] = append(allowedHostsByUser[h.UserID], h.FQDN)
	}

	return userRows, allowedHostsByUser, nil
}

// collectAddressIDs gathers all unique address IDs referenced in a snapshot.
func collectAddressIDs(snap policy.PolicyMapSnapshot) []ids.AddressID {
	seen := make(map[ids.AddressID]struct{})
	var ids []ids.AddressID
	for _, e := range snap.Entries {
		for _, c := range e.Contributors {
			if _, ok := seen[c.AddressID]; !ok {
				seen[c.AddressID] = struct{}{}
				ids = append(ids, c.AddressID)
			}
		}
	}
	return ids
}

// BuildPolicyUserMap is the single business-logic entry point for the user-pivoted
// policy audit view. The handler is a thin wrapper around it. This is the
// integration-test target for orchestration; pure assembly is tested separately
// via assemblePolicyUserMap.
func (r *Repository) BuildPolicyUserMap(
	ctx context.Context,
	reader PolicyMapReader,
	npProvider AuditNetworkPoliciesProvider,
) (httpapi.PolicyUserMapAudit, error) {
	snap := reader.GetPolicyMap()
	addressIDs := collectAddressIDs(snap)

	addrEnrichment, err := r.getPolicyAddressEnrichment(ctx, addressIDs)
	if err != nil {
		return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy address enrichment: %w", err)
	}

	allUsers, allowedHostsByUser, err := r.getAllUsersForPolicyAudit(ctx)
	if err != nil {
		return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy audit users: %w", err)
	}

	// Load enabled network policy entries for the audit view.
	var npEntries []networkpolicies.CacheEntry
	if npProvider != nil {
		npEntries, err = npProvider.GetEnabledCacheEntries(ctx)
		if err != nil {
			return httpapi.PolicyUserMapAudit{}, fmt.Errorf("policy audit network policies: %w", err)
		}
	}

	audit := assemblePolicyUserMap(snap, addrEnrichment, allUsers, allowedHostsByUser)

	// Attach network policy data.
	npAPIEntries := make([]httpapi.PolicyNetworkPolicyEntry, 0, len(npEntries))
	totalHostCount := audit.TotalHostCount
	for _, e := range npEntries {
		effective := len(e.AllowedHostFQDNs)
		if e.AllowAllHosts {
			effective = totalHostCount
		}
		npAPIEntries = append(npAPIEntries, httpapi.PolicyNetworkPolicyEntry{
			PolicyId:           e.PolicyID.Int64(),
			PolicyName:         e.PolicyName,
			Cidr:               e.CIDR,
			Enabled:            true,
			AllowAllHosts:      e.AllowAllHosts,
			EffectiveHostCount: effective,
			TotalHostCount:     totalHostCount,
		})
	}
	audit.NetworkPolicies = npAPIEntries
	audit.TotalNetworkPolicyCount = len(npAPIEntries)

	return audit, nil
}

// GetPolicyUserMap returns the user-pivoted policy cache audit view.
func (h *HTTPHandler) GetPolicyUserMap(
	ctx context.Context,
	_ httpapi.GetPolicyUserMapRequestObject,
) (httpapi.GetPolicyUserMapResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetPolicyUserMap")

	audit, err := h.repo.BuildPolicyUserMap(ctx, h.policyReader, h.npProvider)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to build policy user map", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetPolicyUserMap500JSONResponse(errorMsgResponse("Failed to load policy user map")), nil
	}
	return httpapi.GetPolicyUserMap200JSONResponse(audit), nil
}
