package device

import (
	"context"
	"errors"
	"net/netip"
)

// DeviceRepository defines the persistence operations for devices and addresses.
type DeviceRepository interface {
	GetDeviceByID(ctx context.Context, id DeviceID) (*Device, error)
	CreateDevice(ctx context.Context, name string) (*Device, error)
	GetDevices(ctx context.Context) ([]Device, error)
	CreateAddress(ctx context.Context, deviceId DeviceID, ipAddress string) (*Address, error)
	GetAddressForDeviceByIp(ctx context.Context, deviceId DeviceID, ip string) (*AddressWithStatus, error)
	ListAddresses(ctx context.Context, deviceId DeviceID) ([]AddressWithStatus, error)
	DisableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	EnableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	GetAddressWithStatus(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	CheckAddressOwnership(ctx context.Context, deviceId DeviceID, addressId AddressID) error
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

func (s *Service) AssignAddress(ctx context.Context, deviceID DeviceID, inputIp string) (*AddressWithStatus, bool, error) {
	var resultAddr *AddressWithStatus
	var wasCreated bool

	ipAddress, err := parseAndValidateIP(inputIp)
	if err != nil {
		return nil, false, err
	}

	err = s.repo.RunInTx(ctx, func(tx DeviceRepository) error {
		// Verify device exists before trying to create an address
		_, err := tx.GetDeviceByID(ctx, deviceID)
		if err != nil {
			return err
		}

		resultAddr, err = tx.GetAddressForDeviceByIp(ctx, deviceID, ipAddress)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) {
				// If not found create address
				addr, err := tx.CreateAddress(ctx, deviceID, ipAddress)
				if err != nil {
					return err
				}

				// And enable it
				resultAddr, err = tx.EnableAddress(ctx, addr.ID)
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

		// If it was not created, add an enabled record
		if !wasCreated {
			resultAddr, err = tx.EnableAddress(ctx, resultAddr.AddressId)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, false, err
	}

	return resultAddr, wasCreated, nil
}

func (s *Service) GetAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]AddressWithStatus, error) {
	var addresses []AddressWithStatus

	err := s.repo.RunInTx(ctx, func(tx DeviceRepository) error {
		// Verify device exists before trying to create an address
		_, err := tx.GetDeviceByID(ctx, deviceID)
		if err != nil {
			return err
		}

		addresses, err = tx.ListAddresses(ctx, deviceID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return addresses, nil
}

func (s *Service) DisableAddress(ctx context.Context, deviceID DeviceID, addressID AddressID) (*AddressWithStatus, error) {
	var disabledAddress *AddressWithStatus

	err := s.repo.RunInTx(ctx, func(tx DeviceRepository) error {
		err := tx.CheckAddressOwnership(ctx, deviceID, addressID)
		if err != nil {
			return err
		}

		disabledAddress, err = tx.DisableAddress(ctx, addressID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return disabledAddress, nil
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
