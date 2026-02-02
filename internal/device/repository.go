package device

import (
	"context"
	"fmt"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"github.com/google/uuid"
)

type Repository struct {
	db *database.SQLite
}

func NewRepository(db *database.SQLite) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateDevice(ctx context.Context, name string) (*Device, error) {
	device := Device{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: database.Time{Time: time.Now().UTC()},
	}

	query := `
		INSERT INTO devices (id, name, created_at)
		VALUES (?, ?, ?)
	`

	_, err := r.db.DB().ExecContext(ctx, query,
		device.ID, device.Name, device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	return &device, nil
}

func (r *Repository) GetDevices(ctx context.Context) ([]Device, error) {
	var devices []Device

	query := `
		SELECT id, name, created_at
		FROM devices
		ORDER BY created_at DESC
	`

	// sqlx's Select scans directly into the struct slice
	if err := r.db.DB().SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	if devices == nil {
		return []Device{}, nil
	}

	return devices, nil
}
