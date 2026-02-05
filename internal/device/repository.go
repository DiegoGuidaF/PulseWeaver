package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
)

type Repository struct {
	db *database.SQLite
}

func NewRepository(db *database.SQLite) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDeviceByID(ctx context.Context, id DeviceID) (*Device, error) {
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
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	return &device, nil
}

func (r *Repository) CreateDevice(ctx context.Context, name string) (*Device, error) {
	device := Device{
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	query := `
		INSERT INTO devices (name, created_at)
		VALUES (?, ?)
	`

	result, err := r.db.DB().ExecContext(ctx, query, device.Name, device.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return r.GetDeviceByID(ctx, DeviceID(id))
}

func (r *Repository) GetDevices(ctx context.Context) ([]Device, error) {
	var devices []Device

	query := `
		SELECT id, name, created_at
		FROM devices
		ORDER BY created_at DESC
	`

	if err := r.db.DB().SelectContext(ctx, &devices, query); err != nil {
		return nil, fmt.Errorf("select devices: %w", err)
	}

	if devices == nil {
		return []Device{}, nil
	}

	return devices, nil
}

func (r *Repository) CreateDeviceIP(ctx context.Context, deviceID DeviceID, ipAddress string) (*DeviceIP, error) {
	deviceIP := DeviceIP{
		DeviceID:  deviceID,
		IPAddress: ipAddress,
		CreatedAt: time.Now().UTC(),
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

	return r.GetDeviceIPByID(ctx, DeviceIpID(id))
}

func (r *Repository) GetDeviceIPByID(ctx context.Context, id DeviceIpID) (*DeviceIP, error) {
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

func (r *Repository) ListActiveDeviceIPs(ctx context.Context, deviceID DeviceID) ([]DeviceIP, error) {
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

	if ips == nil {
		return []DeviceIP{}, nil
	}

	return ips, nil
}

func (r *Repository) DisableDeviceIP(ctx context.Context, deviceID DeviceID, deviceIpId DeviceIpID) (*DeviceIP, error) {
	query := `UPDATE device_ips SET disabled_at = CURRENT_TIMESTAMP 
        		WHERE id = ? AND device_id = ? AND disabled_at IS NULL 
        		RETURNING id, device_id, ip_address, created_at, disabled_at`
	var deviceIp DeviceIP
	err := r.db.DB().GetContext(ctx, &deviceIp, query, deviceIpId, deviceID)
	if err == nil {
		return &deviceIp, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDeviceIPNotFound
	}

	return nil, fmt.Errorf("unexpected state during disable operation: %w", err)
}
