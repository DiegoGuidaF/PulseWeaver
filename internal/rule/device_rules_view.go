package rule

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	sq "github.com/Masterminds/squirrel"
)

// RulesForDevices returns the configured rules of the given devices, keyed by
// device id and ordered by rule type. Devices with no configured rules are
// absent from the map rather than mapped to an empty slice.
func (r *Repository) RulesForDevices(ctx context.Context, deviceIDs []ids.DeviceID) (map[ids.DeviceID][]httpapi.DeviceRuleSummary, error) {
	if len(deviceIDs) == 0 {
		return map[ids.DeviceID][]httpapi.DeviceRuleSummary{}, nil
	}

	type row struct {
		DeviceID ids.DeviceID    `db:"device_id"`
		RuleType RuleType        `db:"rule_type"`
		Enabled  bool            `db:"enabled"`
		Config   json.RawMessage `db:"config"`
	}
	query, args, err := sq.
		Select("dr.device_id", "dr.rule_type", "dr.enabled", "dr.config").
		From("device_rules dr").
		Where(sq.Eq{"dr.device_id": deviceIDs}).
		OrderBy("dr.device_id", "dr.rule_type").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device rules query: %w", err)
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get rules for devices: %w", err)
	}

	byDevice := make(map[ids.DeviceID][]httpapi.DeviceRuleSummary, len(rows))
	for _, rw := range rows {
		summary, ok := ruleSummaryFromRow(rw.RuleType, rw.Enabled, rw.Config)
		if !ok {
			continue
		}
		byDevice[rw.DeviceID] = append(byDevice[rw.DeviceID], summary)
	}
	return byDevice, nil
}

// ruleSummaryFromRow converts one device_rules row into its API summary,
// skipping rows whose config fails to parse (ok=false) rather than failing
// the whole batch for a data problem on one device.
func ruleSummaryFromRow(ruleType RuleType, enabled bool, config json.RawMessage) (httpapi.DeviceRuleSummary, bool) {
	raw := Rule{RuleType: ruleType, Enabled: enabled, Config: config}
	switch ruleType {
	case RuleTypeDeviceAddressLease:
		lease, err := raw.ToDeviceAddressLeaseRule()
		if err != nil {
			return httpapi.DeviceRuleSummary{}, false
		}
		return httpapi.DeviceRuleSummary{
			Type:       httpapi.AutoExpiry,
			Enabled:    lease.Enabled,
			TtlSeconds: new(lease.Config.TTLSeconds),
		}, true
	case RuleTypeMaxActiveAddresses:
		maxRule, err := raw.ToMaxActiveAddressesRule()
		if err != nil {
			return httpapi.DeviceRuleSummary{}, false
		}
		return httpapi.DeviceRuleSummary{
			Type:    httpapi.MaxActive,
			Enabled: maxRule.Enabled,
			Limit:   new(maxRule.Config.MaxAddresses),
		}, true
	default:
		return httpapi.DeviceRuleSummary{}, false
	}
}
