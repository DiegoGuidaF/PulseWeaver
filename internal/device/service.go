package device

import (
	"context"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDevices(ctx context.Context) ([]Device, error) {
	return s.repo.GetDevices(ctx)
}

func (s *Service) CreateDevice(ctx context.Context, name string) (*Device, error) {
	return s.repo.CreateDevice(ctx, name)
}

func (s *Service) AssignIP(ctx context.Context, deviceID DeviceID, ipAddress string) (*DeviceIP, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateDeviceIP(ctx, deviceID, ipAddress)
}

func (s *Service) ListDeviceIPs(ctx context.Context, deviceID DeviceID) ([]DeviceIP, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListActiveDeviceIPs(ctx, deviceID)
}

func (s *Service) DisableDeviceIP(ctx context.Context, deviceID DeviceID, deviceIpId DeviceIpID) (*DeviceIP, error) {
	// Verify device exists
	ip, err := s.repo.GetDeviceIPByID(ctx, deviceIpId)
	if err != nil {
		return nil, err
	}

	if ip.DeviceID != deviceID {
		return nil, ErrDeviceIPWrongDevice
	}

	if ip.DisabledAt != nil {
		return nil, ErrDeviceIPDisabled
	}

	return s.repo.DisableDeviceIP(ctx, deviceIpId)
}
