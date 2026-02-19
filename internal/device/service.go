package device

import (
	"context"
	"errors"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

type repository interface {
	GetDeviceByID(ctx context.Context, id DeviceID) (*Device, error)
	CreateDevice(ctx context.Context, device *Device) (*Device, error)
	GetDevices(ctx context.Context) ([]DeviceWithApiKeyPrefix, error)
	CreateAddress(ctx context.Context, address *Address) (*Address, error)
	GetAddressForDeviceByIp(ctx context.Context, deviceId DeviceID, ip string) (*AddressWithStatus, error)
	ListAddresses(ctx context.Context, deviceId DeviceID) ([]AddressWithStatus, error)
	DisableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	EnableAddress(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	GetAddressWithStatus(ctx context.Context, addressId AddressID) (*AddressWithStatus, error)
	CheckAddressOwnership(ctx context.Context, deviceId DeviceID, addressId AddressID) error
	CreateDeviceApiKey(ctx context.Context, apiKey *ApiKey) (*ApiKey, error)
	GetDeviceByApiKeyHash(ctx context.Context, keyHash string) (*Device, error)
	RunInTx(ctx context.Context, fn func(repository) error) error
}

type Service struct {
	repo             repository
	statusChangeChan chan<- struct{}
}

func (s *Service) WithStatusChangeChannel(ch chan<- struct{}) {
	s.statusChangeChan = ch
}

func NewService(repo repository) *Service {
	s := &Service{
		repo: repo,
	}
	return s
}

func (s *Service) GetDevices(ctx context.Context) ([]DeviceWithApiKeyPrefix, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("listing devices")

	devices, err := s.repo.GetDevices(ctx)
	if err != nil {
		logger.Error("database error listing devices", slog.Any(AttrKeyError, err))
		return nil, err
	}
	return devices, nil
}

func (s *Service) CreateDevice(ctx context.Context, name string) (*DeviceWithApiKeyPrefix, string, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("creating device")

	var deviceWithApiKeyPrefix *DeviceWithApiKeyPrefix
	var rawKey string

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		device := NewDevice(name)
		device, err := tx.CreateDevice(ctx, device)
		if err != nil {
			logger.Error("database error creating device", slog.Any(AttrKeyError, err))
			return err
		}

		var apiKey *ApiKey

		apiKey, rawKey, err = NewApiKey(device.ID)
		if err != nil {
			return err
		}

		apiKey, err = tx.CreateDeviceApiKey(ctx, apiKey)
		if err != nil {
			logger.Error("database error creating device API key", slog.Any(AttrKeyError, err))
			return err
		}

		deviceWithApiKeyPrefix = &DeviceWithApiKeyPrefix{
			Device:    *device,
			KeyPrefix: apiKey.KeyPrefix,
		}

		logger.Info("device created", slog.Int64(AttrKeyDeviceID, device.ID.Int64()))
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return deviceWithApiKeyPrefix, rawKey, nil
}

func (s *Service) Authenticate(ctx context.Context, rawKey string) (*Principal, error) {
	// Validate key format (must start with prefix)
	if len(rawKey) < len(ApiKeyPrefix) || rawKey[:len(ApiKeyPrefix)] != ApiKeyPrefix {
		return nil, ErrInvalidApiKey
	}

	// Hash the key
	keyHash := hashApiKey(rawKey)

	// Look up device by key hash
	device, err := s.repo.GetDeviceByApiKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	return PrincipalFromDevice(device), nil
}

func (s *Service) AssignAddress(ctx context.Context, deviceID DeviceID, inputIp string) (*AddressWithStatus, bool, error) {
	logger := logging.FromCtx(ctx)

	logger.Debug("assigning address")

	newAddress, err := NewAddress(deviceID, inputIp)
	if err != nil {
		logger.Warn("invalid IP format")
		return nil, false, err
	}

	var resultAddr *AddressWithStatus
	var wasCreated bool

	err = s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDeviceByID(ctx, deviceID)
		if err != nil {
			if errors.Is(err, ErrDeviceNotFound) {
				logger.Warn("device not found")
				return err
			}
			logger.Error("database error fetching device", slog.Any(AttrKeyError, err))
			return err
		}

		resultAddr, err = tx.GetAddressForDeviceByIp(ctx, deviceID, newAddress.IP)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) {
				logger.Debug("address not found, creating new address")
				addr, err := tx.CreateAddress(ctx, newAddress)
				if err != nil {
					logger.Error("database error creating address", slog.Any(AttrKeyError, err))
					return err
				}

				resultAddr, err = tx.EnableAddress(ctx, addr.ID)
				if err != nil {
					logger.Error("database error enabling address", slog.Any(AttrKeyError, err))
					return err
				}

				logger.Info("address created", slog.Int64(AttrKeyAddressID, addr.ID.Int64()))

				wasCreated = true
			} else {
				logger.Error("database error checking address", slog.Any(AttrKeyError, err))
				return err
			}
		} else {
			logger.Debug("address already exists", slog.Int64(AttrKeyAddressID, resultAddr.Id.Int64()))
			wasCreated = false
		}

		// If it was not created, enable it
		if !wasCreated {
			logger.Info("address exists add status enabled", slog.Int64(AttrKeyAddressID, resultAddr.Id.Int64()))
			resultAddr, err = tx.EnableAddress(ctx, resultAddr.Id)
			if err != nil {
				logger.Error("database error enabling existing address", slog.Any(AttrKeyError, err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, false, err
	}
	logger.Info("address assigned",
		slog.String(AttrKeyAddressIP, resultAddr.IP),
		slog.Int64(AttrKeyAddressID, resultAddr.Id.Int64()),
		slog.Bool(AttrKeyCreated, wasCreated),
	)
	// Notify whitelist service of status change (non-blocking)
	if s.statusChangeChan != nil {
		select {
		case s.statusChangeChan <- struct{}{}:
		default:
		}
	}
	return resultAddr, wasCreated, nil
}

func (s *Service) GetAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]AddressWithStatus, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("listing addresses for device")

	var addresses []AddressWithStatus

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDeviceByID(ctx, deviceID)
		if err != nil {
			if errors.Is(err, ErrDeviceNotFound) {
				logger.Warn("device not found")
				return err
			}
			logger.Error("database error fetching device", slog.Any(AttrKeyError, err))
			return err
		}

		addresses, err = tx.ListAddresses(ctx, deviceID)
		if err != nil {
			logger.Error("database error listing addresses", slog.Any(AttrKeyError, err))
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
	logger := logging.FromCtx(ctx)
	logger.Debug("disabling address")

	var disabledAddress *AddressWithStatus

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		err := tx.CheckAddressOwnership(ctx, deviceID, addressID)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) || errors.Is(err, ErrAddressNotOwnedByDevice) {
				logger.Warn("address not found or not owned by device")
				return err
			}
			logger.Error("database error checking address ownership", slog.Any(AttrKeyError, err))
			return err
		}

		disabledAddress, err = tx.DisableAddress(ctx, addressID)
		if err != nil {
			logger.Error("database error disabling address", slog.Any(AttrKeyError, err))
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	logger.Info("address disabled",
		slog.String(AttrKeyAddressIP, disabledAddress.IP),
		slog.Int64(AttrKeyAddressID, disabledAddress.Id.Int64()),
	)
	// Notify whitelist service of status change (non-blocking)
	if s.statusChangeChan != nil {
		select {
		case s.statusChangeChan <- struct{}{}:
		default:
		}
	}
	return disabledAddress, nil
}
