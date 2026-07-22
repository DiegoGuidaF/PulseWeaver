package devicepairing

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	sq "github.com/Masterminds/squirrel"
)

// PairingsForDevices returns the latest pairing of each given device, keyed by
// device id. Devices that have never been paired are absent from the map.
func (r *Repository) PairingsForDevices(ctx context.Context, deviceIDs []ids.DeviceID) (map[ids.DeviceID]httpapi.DevicePairingSummary, error) {
	if len(deviceIDs) == 0 {
		return map[ids.DeviceID]httpapi.DevicePairingSummary{}, nil
	}

	type row struct {
		DeviceID  ids.DeviceID `db:"device_id"`
		Status    string       `db:"status"`
		ExpiresAt time.Time    `db:"expires_at"`
	}
	const latestPerDevice = `dp.id = (
		SELECT id FROM device_pairings dp2
		WHERE dp2.device_id = dp.device_id
		ORDER BY dp2.created_at DESC, dp2.id DESC
		LIMIT 1
	)`
	query, args, err := sq.
		Select("dp.device_id", "dp.status", "dp.expires_at").
		From("device_pairings dp").
		Where(sq.Eq{"dp.device_id": deviceIDs}).
		Where(sq.Expr(latestPerDevice)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device pairings query: %w", err)
	}

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("get pairings for devices: %w", err)
	}

	byDevice := make(map[ids.DeviceID]httpapi.DevicePairingSummary, len(rows))
	for _, rw := range rows {
		byDevice[rw.DeviceID] = httpapi.DevicePairingSummary{
			Status:    PairingStatusToAPI(EvalStatus(rw.Status, rw.ExpiresAt)),
			ExpiresAt: httpapi.UTCTime(rw.ExpiresAt),
		}
	}
	return byDevice, nil
}
