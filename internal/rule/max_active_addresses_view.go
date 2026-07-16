package rule

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// CountEnabledAddresses returns the number of currently enabled addresses for a device.
// Read-only join against the device domain's addresses table, isolated here per ADR-009
// so a schema change there is greppable: `git grep -l 'FROM addresses' -- '**/*_view.go'`.
func (r *Repository) CountEnabledAddresses(ctx context.Context, deviceID ids.DeviceID) (int, error) {
	var count int
	const query = `SELECT COUNT(*) FROM addresses WHERE device_id = ? AND is_enabled = 1`
	if err := r.db.GetContext(ctx, &count, query, deviceID); err != nil {
		return 0, fmt.Errorf("count enabled addresses: %w", err)
	}
	return count, nil
}
