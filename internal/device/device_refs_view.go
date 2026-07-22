package device

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// GetDeviceRefs returns every non-deleted device as a flat {id, name,
// owner_id} reference, for device pickers and the device→owner reverse map.
func (r *Repository) GetDeviceRefs(ctx context.Context) ([]httpapi.DeviceRef, error) {
	type row struct {
		ID      ids.DeviceID `db:"id"`
		Name    string       `db:"name"`
		OwnerID ids.UserID   `db:"owner_id"`
	}
	const query = `
		SELECT id, name, owner_id
		FROM devices
		WHERE deleted_at IS NULL
		ORDER BY name ASC
	`
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get device refs: %w", err)
	}

	refs := make([]httpapi.DeviceRef, len(rows))
	for i, rw := range rows {
		refs[i] = httpapi.DeviceRef{
			Id:      rw.ID.Int64(),
			Name:    rw.Name,
			OwnerId: rw.OwnerID.Int64(),
		}
	}
	return refs, nil
}
