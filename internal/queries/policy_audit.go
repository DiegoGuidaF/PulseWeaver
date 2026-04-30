package queries

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	UserID           int64            `db:"user_id"`
	UserName         string           `db:"user_name"`
}

// getPolicyEnrichmentData fetches display metadata for the given address IDs.
func (r *Repository) getPolicyEnrichmentData(ctx context.Context, addressIDs []device.AddressID) (map[device.AddressID]policyEnrichmentRow, error) {
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
		return nil, fmt.Errorf("get policy enrichment data: %w", err)
	}

	result := make(map[device.AddressID]policyEnrichmentRow, len(rows))
	for _, row := range rows {
		result[row.AddressID] = row
	}
	return result, nil
}

// GetPolicyMap Allows retrieving the current policy map used for the verify-ip authorization flow
func (h *HTTPHandler) GetPolicyMap(
	ctx context.Context,
	_ httpapi.GetPolicyMapRequestObject,
) (httpapi.GetPolicyMapResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetPolicyMap")

	snap := h.policyReader.GetPolicyMap()

	// Collect unique address IDs for a single enrichment query.
	var addressIDs []device.AddressID
	seen := make(map[device.AddressID]struct{})
	for _, e := range snap.Entries {
		for _, c := range e.Contributors {
			if _, ok := seen[c.AddressID]; !ok {
				seen[c.AddressID] = struct{}{}
				addressIDs = append(addressIDs, c.AddressID)
			}
		}
	}

	enrichment, err := h.repo.getPolicyEnrichmentData(ctx, addressIDs)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to enrich policy map", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetPolicyMap500JSONResponse(errorMsgResponse("Failed to load policy map")), nil
	}

	entries := make([]httpapi.PolicyMapEntry, len(snap.Entries))
	for i, e := range snap.Entries {
		// Build a set of effective hosts for this entry once, used to compute trimmed_hosts per contributor.
		effectiveHosts := make(map[string]struct{}, len(e.AllowedHosts))
		for _, h := range e.AllowedHosts {
			effectiveHosts[h] = struct{}{}
		}

		contributors := make([]httpapi.PolicyMapContributor, len(e.Contributors))
		for j, c := range e.Contributors {
			meta := enrichment[c.AddressID]

			var trimmedHosts []string
			if !c.UserBypass {
				for _, h := range c.UserAllowedHosts {
					if _, ok := effectiveHosts[h]; !ok {
						trimmedHosts = append(trimmedHosts, h)
					}
				}
			}
			if trimmedHosts == nil {
				trimmedHosts = []string{}
			}

			contributors[j] = httpapi.PolicyMapContributor{
				DeviceId:         meta.DeviceID.Int64(),
				DeviceName:       meta.DeviceName,
				AddressId:        c.AddressID.Int64(),
				AddressUpdatedAt: httpapi.UTCTime(meta.AddressUpdatedAt),
				UserId:           meta.UserID,
				UserName:         meta.UserName,
				UserBypass:       c.UserBypass,
				UserAllowedHosts: c.UserAllowedHosts,
				TrimmedHosts:     trimmedHosts,
			}
		}
		entries[i] = httpapi.PolicyMapEntry{
			Ip:                  e.IP,
			BypassAllowlist:     e.BypassAllowlist,
			AllowedHosts:        e.AllowedHosts,
			IntersectionApplied: e.IntersectionApplied,
			Contributors:        contributors,
		}
	}

	audit := httpapi.PolicyMapAudit{
		RefreshedAt:       httpapi.UTCTime(snap.LastRefreshedAt),
		RefreshDurationMs: int(snap.LastRefreshDurationMs),
		Entries:           entries,
	}
	return httpapi.GetPolicyMap200JSONResponse(audit), nil
}
