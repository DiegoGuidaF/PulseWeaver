package queries

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DeviceExists(ctx context.Context, deviceID ids.DeviceID) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1
			FROM devices d
			WHERE d.id = ? AND d.deleted_at IS NULL
		)
	`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, deviceID); err != nil {
		return false, fmt.Errorf("check device existence: %w", err)
	}

	return exists, nil
}
