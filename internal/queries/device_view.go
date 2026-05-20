package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/jmoiron/sqlx"
)

type DeviceView struct {
	ID           ids.DeviceID      `db:"id"`
	Name         string            `db:"name"`
	DeviceType   device.DeviceType `db:"device_type"`
	Description  *string           `db:"description"`
	Icon         *string           `db:"icon"`
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
	KeyPrefix    *string           `db:"key_prefix"`
	AddressCount int               `db:"address_count"`
	LastSeenAt   *database.DBTime  `db:"last_seen_at"`
	OwnerID      ids.UserID        `db:"owner_id"`
	OwnerName    string            `db:"owner_name"`
}

// GetDeviceList returns all non-deleted devices grouped by their owning user.
// Each group carries owner metadata (host groups, bypass flag, aggregate live-address count)
// and per-device rule summaries and state. Two SQL round trips: one for devices+owners,
// one for host-group memberships.
func (r *Repository) GetDeviceList(ctx context.Context) ([]httpapi.DeviceOwnerGroup, error) {
	type deviceRow struct {
		DeviceID             ids.DeviceID     `db:"id"`
		DeviceName           string           `db:"name"`
		DeviceIcon           *string          `db:"icon"`
		KeyPrefix            *string          `db:"key_prefix"`
		DeviceCreatedAt      time.Time        `db:"created_at"`
		LiveAddressCount     int              `db:"live_address_count"`
		LastSeenAt           *database.DBTime `db:"last_seen_at"`
		OwnerID              ids.UserID       `db:"owner_id"`
		OwnerUsername        string           `db:"owner_username"`
		OwnerDisplayName     string           `db:"owner_display_name"`
		OwnerRole            string           `db:"owner_role"`
		OwnerBypassHostCheck bool             `db:"owner_bypass_hosts_check"`
		LeaseEnabled         *bool            `db:"lease_rule_enabled"`
		LeaseConfig          *string          `db:"lease_rule_config"`
		MaxEnabled           *bool            `db:"max_rule_enabled"`
		MaxConfig            *string          `db:"max_rule_config"`
	}

	const deviceQuery = `
		SELECT
			d.id,
			d.name,
			d.icon,
			dk.key_prefix,
			d.created_at,
			COUNT(a.id)                        AS live_address_count,
			MAX(a.updated_at)                  AS last_seen_at,
			u.id                               AS owner_id,
			u.username                         AS owner_username,
			u.display_name                     AS owner_display_name,
			u.role                             AS owner_role,
			COALESCE(uhs.bypass_host_check, 0) AS owner_bypass_hosts_check,
			dr_lease.enabled                   AS lease_rule_enabled,
			dr_lease.config                    AS lease_rule_config,
			dr_max.enabled                     AS max_rule_enabled,
			dr_max.config                      AS max_rule_config
		FROM devices d
		JOIN  users u ON u.id = d.owner_id
		LEFT JOIN user_host_settings uhs ON uhs.user_id = u.id
		LEFT JOIN device_api_keys dk     ON dk.device_id = d.id
		LEFT JOIN addresses a            ON a.device_id = d.id AND a.is_enabled = 1
		LEFT JOIN device_rules dr_lease  ON dr_lease.device_id = d.id
		                                 AND dr_lease.rule_type = 'device_lease'
		LEFT JOIN device_rules dr_max    ON dr_max.device_id = d.id
		                                 AND dr_max.rule_type = 'max_active_addresses'
		WHERE d.deleted_at IS NULL
		GROUP BY d.id, d.name, d.icon, dk.key_prefix, d.created_at,
		         u.id, u.username, u.display_name, u.role, uhs.bypass_host_check,
		         dr_lease.enabled, dr_lease.config, dr_max.enabled, dr_max.config
		ORDER BY u.display_name, d.name ASC
	`

	var rows []deviceRow
	if err := r.db.SelectContext(ctx, &rows, deviceQuery); err != nil {
		return nil, fmt.Errorf("get device list: %w", err)
	}
	if len(rows) == 0 {
		return []httpapi.DeviceOwnerGroup{}, nil
	}

	// Collect unique owner IDs preserving ORDER BY order.
	ownerOrder := make([]ids.UserID, 0, len(rows))
	seenOwner := make(map[ids.UserID]bool, len(rows))
	for _, row := range rows {
		if !seenOwner[row.OwnerID] {
			ownerOrder = append(ownerOrder, row.OwnerID)
			seenOwner[row.OwnerID] = true
		}
	}

	// Query 2: host groups for the returned owners.
	type hostGroupRow struct {
		UserID ids.UserID `db:"user_id"`
		ID     int64      `db:"group_id"`
		Name   string     `db:"group_name"`
		Color  string     `db:"group_color"`
		Icon   string     `db:"group_icon"`
	}
	groupsByOwner := make(map[ids.UserID][]httpapi.HostGroupSummary, len(ownerOrder))

	hgQuery, hgArgs, err := sqlx.In(`
		SELECT
			uahg.user_id,
			hg.id    AS group_id,
			hg.name  AS group_name,
			hg.color AS group_color,
			hg.icon  AS group_icon
		FROM user_allowed_host_groups uahg
		JOIN host_groups hg ON hg.id = uahg.host_group_id
		WHERE uahg.user_id IN (?)
		ORDER BY uahg.user_id, hg.name
	`, ownerOrder)
	if err != nil {
		return nil, fmt.Errorf("build host groups query: %w", err)
	}
	hgQuery = r.db.Rebind(hgQuery)

	var hgRows []hostGroupRow
	if err := r.db.SelectContext(ctx, &hgRows, hgQuery, hgArgs...); err != nil {
		return nil, fmt.Errorf("get owner host groups: %w", err)
	}
	for _, hg := range hgRows {
		groupsByOwner[hg.UserID] = append(groupsByOwner[hg.UserID], httpapi.HostGroupSummary{
			Id:    hg.ID,
			Name:  hg.Name,
			Color: hg.Color,
			Icon:  hg.Icon,
		})
	}

	// Assemble groups in owner order.
	type ownerAcc struct {
		meta    deviceRow
		devices []httpapi.DeviceListEntry
		liveSum int
	}
	acc := make(map[ids.UserID]*ownerAcc, len(ownerOrder))
	for _, row := range rows {
		if _, exists := acc[row.OwnerID]; !exists {
			acc[row.OwnerID] = &ownerAcc{meta: row}
		}
		entry := httpapi.DeviceListEntry{
			Id:               row.DeviceID.Int64(),
			Name:             row.DeviceName,
			Icon:             row.DeviceIcon,
			KeyPrefix:        row.KeyPrefix,
			CreatedAt:        httpapi.UTCTime(row.DeviceCreatedAt),
			LiveAddressCount: row.LiveAddressCount,
			State:            deviceListState(row.LiveAddressCount),
			Rules:            parseRuleSummaries(row.LeaseEnabled, row.LeaseConfig, row.MaxEnabled, row.MaxConfig),
			Pairing:          nil,
		}
		if row.LastSeenAt != nil {
			entry.LastSeenAt = new(httpapi.UTCTime(row.LastSeenAt.Time))
		}
		a := acc[row.OwnerID]
		a.devices = append(a.devices, entry)
		a.liveSum += row.LiveAddressCount
	}

	groups := make([]httpapi.DeviceOwnerGroup, 0, len(ownerOrder))
	for _, ownerID := range ownerOrder {
		a := acc[ownerID]
		hgs := groupsByOwner[ownerID]
		if hgs == nil {
			hgs = []httpapi.HostGroupSummary{}
		}
		groups = append(groups, httpapi.DeviceOwnerGroup{
			Owner: httpapi.DeviceListOwner{
				Id:               a.meta.OwnerID.Int64(),
				Username:         a.meta.OwnerUsername,
				DisplayName:      a.meta.OwnerDisplayName,
				Role:             httpapi.UserRole(a.meta.OwnerRole),
				BypassHostsCheck: a.meta.OwnerBypassHostCheck,
				HostGroups:       hgs,
				DeviceCount:      len(a.devices),
				LiveAddressCount: a.liveSum,
			},
			Devices: a.devices,
		})
	}
	return groups, nil
}

// GetDevicesByUser returns all non-deleted devices owned by the given user.
func (r *Repository) GetDevicesByUser(ctx context.Context, userID ids.UserID) ([]DeviceView, error) {
	const query = `
		SELECT
			d.id,
			d.name,
			d.device_type,
			d.description,
			d.icon,
			d.created_at,
			d.updated_at,
			dk.key_prefix,
			COUNT(a.id)       AS address_count,
			MAX(a.updated_at) AS last_seen_at,
			d.owner_id,
			u.display_name    AS owner_name
		FROM devices d
		JOIN  users u             ON u.id = d.owner_id
		LEFT JOIN device_api_keys dk ON dk.device_id = d.id
		LEFT JOIN addresses a        ON a.device_id = d.id AND a.is_enabled = 1
		WHERE d.deleted_at IS NULL AND d.owner_id = ?
		GROUP BY d.id, d.name, d.device_type, d.description, d.icon,
		         d.created_at, d.updated_at, dk.key_prefix, d.owner_id, u.display_name
		ORDER BY d.name ASC
	`
	var rows []DeviceView
	if err := r.db.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, fmt.Errorf("get devices by user: %w", err)
	}
	if rows == nil {
		return []DeviceView{}, nil
	}
	return rows, nil
}

func deviceListState(liveAddressCount int) httpapi.DeviceState {
	if liveAddressCount > 0 {
		return httpapi.Healthy
	}
	return httpapi.Stale
}

// parseRuleSummaries converts nullable rule columns from the device list query
// into the API rule summary slice. Parsing delegates to the rule package to
// avoid duplicating config deserialization logic.
func parseRuleSummaries(leaseEnabled *bool, leaseConfig *string, maxEnabled *bool, maxConfig *string) []httpapi.DeviceRuleSummary {
	var rules []httpapi.DeviceRuleSummary

	if leaseEnabled != nil && leaseConfig != nil {
		r := rule.Rule{
			RuleType: rule.RuleTypeDeviceAddressLease,
			Enabled:  *leaseEnabled,
			Config:   json.RawMessage(*leaseConfig),
		}
		if parsed, err := r.ToDeviceAddressLeaseRule(); err == nil {
			rules = append(rules, httpapi.DeviceRuleSummary{
				Type:       httpapi.AutoExpiry,
				Enabled:    parsed.Enabled,
				TtlSeconds: new(parsed.Config.TTLSeconds),
			})
		}
	}

	if maxEnabled != nil && maxConfig != nil {
		r := rule.Rule{
			RuleType: rule.RuleTypeMaxActiveAddresses,
			Enabled:  *maxEnabled,
			Config:   json.RawMessage(*maxConfig),
		}
		if parsed, err := r.ToMaxActiveAddressesRule(); err == nil {
			rules = append(rules, httpapi.DeviceRuleSummary{
				Type:    httpapi.MaxActive,
				Enabled: parsed.Enabled,
				Limit:   new(parsed.Config.MaxAddresses),
			})
		}
	}

	if rules == nil {
		return []httpapi.DeviceRuleSummary{}
	}
	return rules
}
