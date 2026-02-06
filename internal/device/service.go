package device

import (
	"context"
	"net/netip"
)

// DeviceRepository defines the persistence operations for devices and addresses.
type DeviceRepository interface {
	GetDeviceByID(ctx context.Context, id DeviceId) (*Device, error)
	CreateDevice(ctx context.Context, name string) (*Device, error)
	GetDevices(ctx context.Context) ([]Device, error)
	CreateAddress(ctx context.Context, deviceId DeviceId, ipAddress string) (*Address, error)
	CreateAddressWithNew(ctx context.Context, deviceId DeviceId, ipAddress string) (*Address, bool, error)
	ListActiveAddresses(ctx context.Context, deviceId DeviceId) ([]Address, error)
	DisableAddress(ctx context.Context, deviceId DeviceId, addressId AddressId) (*Address, error)
}

type Service struct {
	repo DeviceRepository
}

func NewService(repo DeviceRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDevices(ctx context.Context) ([]Device, error) {
	return s.repo.GetDevices(ctx)
}

func (s *Service) CreateDevice(ctx context.Context, name string) (*Device, error) {
	return s.repo.CreateDevice(ctx, name)
}

func (s *Service) AssignAddress(ctx context.Context, deviceID DeviceId, ipAddress string) (*Address, error) {
	if err := parseAndValidateIP(ipAddress); err != nil {
		return nil, err
	}

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

func (s *Service) Heartbeat(ctx context.Context, deviceID DeviceId, ipAddress string) (*Address, bool, error) {
	if err := parseAndValidateIP(ipAddress); err != nil {
		return nil, false, err
	}

	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, false, err
	}

	return s.repo.CreateAddressWithNew(ctx, deviceID, ipAddress)
}

// parseAndValidateIP parses and validates that the given string is a valid IPv4 or IPv6 address.
func parseAndValidateIP(ipAddress string) error {
	_, err := netip.ParseAddr(ipAddress)
	if err != nil {
		return ErrInvalidIPFormat
	}
	return nil
}
