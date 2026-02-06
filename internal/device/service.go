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

func (s *Service) AssignAddress(ctx context.Context, deviceID DeviceId, ipAddress string) (*Address, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.CreateAddress(ctx, deviceID, ipAddress)
}

func (s *Service) GetAddressesForDevice(ctx context.Context, deviceID DeviceId) ([]Address, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListActiveAddresses(ctx, deviceID)
}

func (s *Service) DisableAddress(ctx context.Context, deviceID DeviceId, addressID AddressId) (*Address, error) {
	return s.repo.DisableAddress(ctx, deviceID, addressID)
}

func (s *Service) PingAddress(ctx context.Context, deviceID DeviceId, ipAddress string) (*Address, bool, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, false, err
	}

	return s.repo.CreateAddressWithNew(ctx, deviceID, ipAddress)
}
