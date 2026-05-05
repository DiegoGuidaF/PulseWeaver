package queries

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/jmoiron/sqlx"
)

// PolicyMapReader is the cross-domain interface consumed by the queries package.
// Implemented by policy.Service.
type PolicyMapReader interface {
	GetPolicyMap() policy.PolicyMapSnapshot
}

// policyEnrichmentRow holds SQL-joined metadata for a single address contributor.
type policyEnrichmentRow struct {
	AddressID        device.AddressID `db:"address_id"`
	AddressUpdatedAt time.Time        `db:"address_updated_at"`
	DeviceID         device.DeviceID  `db:"device_id"`
	DeviceName       string           `db:"device_name"`
	UserID           auth.UserID      `db:"user_id"`
	UserName         string           `db:"user_name"`
}

// getPolicyAddressEnrichment fetches display metadata for the given address IDs.
func (r *Repository) getPolicyAddressEnrichment(ctx context.Context, addressIDs []device.AddressID) (map[device.AddressID]policyEnrichmentRow, error) {
	if len(addressIDs) == 0 {
		return map[device.AddressID]policyEnrichmentRow{}, nil
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

	result := make(map[device.AddressID]policyEnrichmentRow, len(rows))
	for _, row := range rows {
		result[row.AddressID] = row
	}
	return result, nil
}

// getAllUsersForPolicyAudit returns every non-deleted user with their bypass flag,
// plus a map of pre-intersection allowed FQDNs keyed by user ID.
// Two queries assembled in Go per queries-read-models.md pattern.
func (r *Repository) getAllUsersForPolicyAudit(ctx context.Context) ([]policyAuditUserRow, map[auth.UserID][]string, error) {
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
		UserID auth.UserID `db:"user_id"`
		FQDN   string      `db:"fqdn"`
	}
	var hostRows []hostRow
	if err := r.db.SelectContext(ctx, &hostRows, hostsQuery); err != nil {
		return nil, nil, fmt.Errorf("list user allowed hosts for policy audit: %w", err)
	}

	allowedHostsByUser := make(map[auth.UserID][]string, len(userRows))
	for _, h := range hostRows {
		allowedHostsByUser[h.UserID] = append(allowedHostsByUser[h.UserID], h.FQDN)
	}

	return userRows, allowedHostsByUser, nil
}

// collectAddressIDs gathers all unique address IDs referenced in a snapshot.
func collectAddressIDs(snap policy.PolicyMapSnapshot) []device.AddressID {
	seen := make(map[device.AddressID]struct{})
	var ids []device.AddressID
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

	return assemblePolicyUserMap(snap, addrEnrichment, allUsers, allowedHostsByUser), nil
}

// GetPolicyUserMap returns the user-pivoted policy cache audit view.
func (h *HTTPHandler) GetPolicyUserMap(
	ctx context.Context,
	_ httpapi.GetPolicyUserMapRequestObject,
) (httpapi.GetPolicyUserMapResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetPolicyUserMap")

	audit, err := h.repo.BuildPolicyUserMap(ctx, h.policyReader)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to build policy user map", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetPolicyUserMap500JSONResponse(errorMsgResponse("Failed to load policy user map")), nil
	}
	return httpapi.GetPolicyUserMap200JSONResponse(audit), nil
}
