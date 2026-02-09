package device

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"
)

// DeviceRepository defines the persistence operations for devices and addresses.
type DeviceRepository interface {
	GetDeviceByID(ctx context.Context, id DeviceID) (*Device, error)
	CreateDevice(ctx context.Context, name string) (*Device, error)
	GetDevices(ctx context.Context) ([]Device, error)
	CreateAddress(ctx context.Context, deviceId DeviceID, ipAddress string) (*Address, error)
	FindAddressForDeviceByIp(ctx context.Context, deviceId DeviceID, ip string) (*Address, error)
	ListAddresses(ctx context.Context, deviceId DeviceID) ([]AddressWithStatus, error)
	DisableAddress(ctx context.Context, deviceId DeviceID, addressId AddressID) (*AddressWithStatus, error)
	EnableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	GetAddressWithStatus(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	RunInTx(ctx context.Context, fn func(DeviceRepository) error) error
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

func (s *Service) AssignAddress(ctx context.Context, deviceID DeviceID, ipInput string) (*AddressWithStatus, bool, error) {
	ipAddress, err := parseAndValidateIP(ipInput)
	if err != nil {
		return nil, false, err
	}

	return s.getOrCreateAddress(ctx, deviceID, ipAddress)
}

func (s *Service) GetAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]AddressWithStatus, error) {
	// Check device exists
	_, err := s.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListAddresses(ctx, deviceID)
}

func (s *Service) DisableAddress(ctx context.Context, deviceID DeviceID, addressID AddressID) (*AddressWithStatus, error) {
	return s.repo.DisableAddress(ctx, deviceID, addressID)
}

func (s *Service) Heartbeat(ctx context.Context, deviceID DeviceID, ipInput string) (*AddressWithStatus, bool, error) {
	var resultAddr *AddressWithStatus
	var wasCreated bool

	ipAddress, err := parseAndValidateIP(ipInput)
	if err != nil {
		return nil, false, err
	}

	err = s.repo.RunInTx(ctx, func(tx DeviceRepository) error {
		// Check device exists
		_, err = tx.GetDeviceByID(ctx, deviceID)
		if err != nil {
			return err
		}

		// This would run a nested transaction right now...
		addr, created, err := s.getOrCreateAddressWithTx(ctx, tx, deviceID, ipAddress)
		if err != nil {
			return err
		}

		resultAddr = addr
		wasCreated = created
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	return resultAddr, wasCreated, nil
}

// parseAndValidateIP parses and validates that the given string is a valid IPv4 or IPv6 address.
// It ignores the port if present and only cares about the IP component.
func parseAndValidateIP(ipInput string) (string, error) {
	// Try to parse as IP without port
	if ip, err := netip.ParseAddr(ipInput); err == nil {
		ipStr := ip.String()
		return ipStr, nil
	}

	// If that fails, try to parse as IP with port
	if ap, err := netip.ParseAddrPort(ipInput); err == nil {
		ipStr := ap.Addr().String()
		return ipStr, nil
	}

	// If both fail, return error
	return "", ErrInvalidIPFormat
}
func (s *Service) getOrCreateAddress(ctx context.Context, deviceID DeviceID, ipAddress string) (*AddressWithStatus, bool, error) {
	return s.getOrCreateAddressWithTx(ctx, s.repo, deviceID, ipAddress)
}

func (s *Service) getOrCreateAddressWithTx(ctx context.Context, repo DeviceRepository, deviceID DeviceID, ipAddress string) (*AddressWithStatus, bool, error) {
	var resultAddr *AddressWithStatus
	var wasCreated bool

	err := repo.RunInTx(ctx, func(tx DeviceRepository) error {
		addr, err := tx.FindAddressForDeviceByIp(ctx, deviceID, ipAddress)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// If not found create address
				addr, err = tx.CreateAddress(ctx, deviceID, ipAddress)
				if err != nil {
					return err
				}

				// And enable it
				_, err := tx.EnableAddress(ctx, addr.ID)
				if err != nil {
					return err
				}

				wasCreated = true
			} else {
				return err
			}
		} else {
			wasCreated = false
		}
		resultAddr, err = tx.GetAddressWithStatus(ctx, addr.ID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, false, err
	}

	return resultAddr, wasCreated, nil
}
