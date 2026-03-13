package queries

import (
	"context"
	"fmt"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) DeviceExists(ctx context.Context, deviceID device.DeviceID) (bool, error) {
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
