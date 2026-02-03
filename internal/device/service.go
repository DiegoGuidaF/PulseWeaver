package device

import (
	"context"
	"fmt"
	"net"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDevices(ctx context.Context) ([]Device, error) {
	devices, err := s.repo.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("get devices: %w", err)
	}

	return devices, nil
}

func (s *Service) CreateDevice(ctx context.Context, name string) (*Device, error) {
	device, err := s.repo.CreateDevice(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}

	return device, nil
}

func (s *Service) AssignIP(ctx context.Context, deviceID string, ipAddress string) (*DeviceIP, error) {
	// Validate IPv4 format
	if err := validateIPv4(ipAddress); err != nil {
		return nil, err
	}

	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	return s.repo.CreateDeviceIP(ctx, deviceID, ipAddress)
}

func (s *Service) ListDeviceIPs(ctx context.Context, deviceID string) ([]DeviceIP, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	return s.repo.ListActiveDeviceIPs(ctx, deviceID)
}

func (s *Service) DisableDeviceIP(ctx context.Context, deviceID string, deviceIpId string) error {
	// Get IP to verify it belongs to the device and is active
	ip, err := s.repo.GetDeviceIPByID(ctx, deviceIpId)
	if err != nil {
		return fmt.Errorf("device IP not found: %w", err)
	}

	if ip.DeviceID != deviceID {
		return fmt.Errorf("device IP does not belong to device")
	}

	if ip.DisabledAt != nil {
		return fmt.Errorf("device IP already disabled")
	}

	return s.repo.DisableDeviceIP(ctx, deviceIpId)
}

func validateIPv4(ipAddress string) error {
	ip := net.ParseIP(ipAddress)
	if ip == nil {
		return fmt.Errorf("invalid IP address format")
	}

	// Check it's IPv4 (net.ParseIP accepts both IPv4 and IPv6)
	if ip.To4() == nil {
		return fmt.Errorf("only IPv4 addresses are supported")
	}

	return nil
}
