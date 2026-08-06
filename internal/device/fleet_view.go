package device

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	sq "github.com/Masterminds/squirrel"
)

// FleetRow is one device in list shape, tagged with its owner so a composer can
// group without a second lookup. Rules and pairing are attached by the composer
// from the domains that own them.
type FleetRow struct {
	OwnerID ids.UserID
	Device  httpapi.FleetDevice
}

// FleetRows returns every non-deleted device in the device-list shape, ordered
// by name, optionally narrowed to one owner. This is the only query in the fleet
// composition that knows about owner scoping; everything downstream is keyed by
// device id.
func (r *Repository) FleetRows(ctx context.Context, ownerID *ids.UserID) ([]FleetRow, error) {
	type deviceRow struct {
		ID               ids.DeviceID     `db:"id"`
		OwnerID          ids.UserID       `db:"owner_id"`
		Name             string           `db:"name"`
		Icon             *string          `db:"icon"`
		Description      *string          `db:"description"`
		KeyPrefix        *string          `db:"key_prefix"`
		CreatedAt        time.Time        `db:"created_at"`
		DisabledAt       *database.DBTime `db:"disabled_at"`
		LiveAddressCount int              `db:"live_address_count"`
		LastSeenAt       *database.DBTime `db:"last_seen_at"`
	}

	builder := sq.Select(
		"d.id",
		"d.owner_id",
		"d.name",
		"d.icon",
		"d.description",
		"dk.key_prefix",
		"d.created_at",
		"d.disabled_at",
		"COUNT(CASE WHEN a.is_enabled = 1 THEN a.id END) AS live_address_count",
		"MAX(a.updated_at)                              AS last_seen_at",
	).
		From("devices d").
		LeftJoin("device_api_keys dk ON dk.device_id = d.id").
		LeftJoin("addresses a        ON a.device_id = d.id").
		Where(sq.Eq{"d.deleted_at": nil})
	if ownerID != nil {
		builder = builder.Where(sq.Eq{"d.owner_id": *ownerID})
	}
	query, args, err := builder.
		GroupBy("d.id", "d.owner_id", "d.name", "d.icon", "d.description", "dk.key_prefix", "d.created_at", "d.disabled_at").
		OrderBy("d.name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build fleet rows query: %w", err)
	}

	var rows []deviceRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get fleet rows: %w", err)
	}

	fleet := make([]FleetRow, len(rows))
	for i, rw := range rows {
		entry := httpapi.FleetDevice{
			Id:               rw.ID.Int64(),
			Name:             rw.Name,
			Icon:             rw.Icon,
			Description:      rw.Description,
			ApiKeyPrefix:     rw.KeyPrefix,
			CreatedAt:        httpapi.UTCTime(rw.CreatedAt),
			LiveAddressCount: rw.LiveAddressCount,
			State:            deviceListState(rw.LiveAddressCount, rw.DisabledAt != nil),
			Rules:            []httpapi.DeviceRuleSummary{},
		}
		if rw.LastSeenAt != nil {
			entry.LastSeenAt = new(httpapi.UTCTime(rw.LastSeenAt.Time))
		}
		fleet[i] = FleetRow{OwnerID: rw.OwnerID, Device: entry}
	}
	return fleet, nil
}

// deviceListState derives the lifecycle state shown in device list rows.
func deviceListState(liveAddressCount int, disabled bool) httpapi.DeviceState {
	if disabled {
		return httpapi.DeviceStateDisabled
	}
	if liveAddressCount > 0 {
		return httpapi.DeviceStateHealthy
	}
	return httpapi.DeviceStateStale
}
