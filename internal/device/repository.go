package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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

func (r *Repository) GetDeviceByID(ctx context.Context, id string) (*Device, error) {
	var device Device
	query := `
		SELECT id, name, created_at
		FROM devices
		WHERE id = ?
	`
	err := r.db.DB().GetContext(ctx, &device, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("failed to get device IP: %w", err)
	}
	return &device, nil
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

	_, err := r.db.DB().ExecContext(ctx, query, device.ID, device.Name, device.CreatedAt)
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

func (r *Repository) CreateDeviceIP(ctx context.Context, deviceID string, ipAddress string) (*DeviceIP, error) {
	deviceIP := DeviceIP{
		DeviceID:  deviceID,
		IPAddress: ipAddress,
		CreatedAt: database.Time{Time: time.Now().UTC()},
	}

	query := `
		INSERT INTO device_ips (device_id, ip_address, created_at)
		VALUES (?, ?, ?)
	`
	result, err := r.db.DB().ExecContext(ctx, query, deviceIP.DeviceID, deviceIP.IPAddress, deviceIP.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create device IP: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return r.GetDeviceIPByID(ctx, strconv.FormatInt(id, 10))
}

func (r *Repository) GetDeviceIPByID(ctx context.Context, id string) (*DeviceIP, error) {
	var ip DeviceIP
	query := `SELECT id, device_id, ip_address, created_at, disabled_at FROM device_ips WHERE id = ?`
	err := r.db.DB().GetContext(ctx, &ip, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceIPNotFound
		}
		return nil, fmt.Errorf("failed to get device IP: %w", err)
	}
	return &ip, nil
}

func (r *Repository) ListActiveDeviceIPs(ctx context.Context, deviceID string) ([]DeviceIP, error) {
	var ips []DeviceIP
	query := `
		SELECT id, device_id, ip_address, created_at, disabled_at 
		FROM device_ips 
		WHERE device_id = ? AND disabled_at IS NULL
		ORDER BY created_at DESC
	`
	err := r.db.DB().SelectContext(ctx, &ips, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list device IPs: %w", err)
	}
	return ips, nil
}

func (r *Repository) DisableDeviceIP(ctx context.Context, id string) error {
	query := `UPDATE device_ips SET disabled_at = CURRENT_TIMESTAMP WHERE id = ? AND disabled_at IS NULL`
	result, err := r.db.DB().ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to disable device IP: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		// Could be: IP doesn't exist OR already disabled
		// We need to check which case it is
		_, err := r.GetDeviceIPByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrDeviceIPNotFound) {
				return ErrDeviceIPNotFound
			}
			return err
		}
		// IP exists but was already disabled
		return ErrDeviceIPDisabled
	}

	return nil
}
