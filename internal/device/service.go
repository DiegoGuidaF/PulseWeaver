package device

import (
	"context"
	"errors"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
)

type repository interface {
	GetDevice(ctx context.Context, id DeviceID) (*Device, error)
	CreateDevice(ctx context.Context, params *CreateDeviceParams) (*Device, error)
	GetDevices(ctx context.Context) ([]Device, error)
	DeleteDevice(ctx context.Context, id DeviceID) error
	CreateAddress(ctx context.Context, params *CreateAddressParams) (*Address, error)
	GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip string) (*Address, error)
	ListAddresses(ctx context.Context, deviceID DeviceID) ([]Address, error)
	DisableAddress(ctx context.Context, addressID AddressID) (*Address, error)
	DisableAddresses(ctx context.Context, addressIDs []AddressID, source StatusSource) ([]Address, error)
	EnableAddress(ctx context.Context, addressID AddressID, source StatusSource) (*Address, error)
	CheckAddressOwnership(ctx context.Context, deviceID DeviceID, addressID AddressID) error
	GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error)
	RunInTx(ctx context.Context, fn func(repository) error) error
}

type Service struct {
	repo                repository
	events              chan<- AddressEvent // Carries data
	addressStateChanged chan<- struct{}     // Dumb signal on address changes
}

func NewService(repo repository, events chan<- AddressEvent, addressStateChanged chan<- struct{}) *Service {
	s := &Service{
		repo:                repo,
		events:              events,
		addressStateChanged: addressStateChanged,
	}
	return s
}

func (s *Service) GetDevices(ctx context.Context) ([]Device, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("listing devices")

	devices, err := s.repo.GetDevices(ctx)
	if err != nil {
		logger.Error("database error listing devices", slog.Any(AttrKeyError, err))
		return nil, err
	}
	return devices, nil
}

func (s *Service) DeleteDevice(ctx context.Context, deviceID DeviceID) error {
	logger := logging.FromCtx(ctx)
	logger.Debug("deleting device")

	err := s.repo.DeleteDevice(ctx, deviceID)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			logger.Warn("device not found")
			return err
		}
		logger.Error("database error deleting device", slog.Any(AttrKeyError, err))
		return err
	}
	logger.Info("device deleted", slog.Int64(AttrKeyDeviceID, deviceID.Int64()))
	return nil
}

func (s *Service) CreateDevice(ctx context.Context, name string) (*Device, string, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("creating device")

	createDeviceParams, rawKey, err := NewCreateDeviceParams(name)
	if err != nil {
		logger.Error("invalid create device params", slog.Any(AttrKeyError, err))
		return nil, "", err
	}

	createdDevice, err := s.repo.CreateDevice(ctx, createDeviceParams)
	if err != nil {
		if errors.Is(err, ErrDuplicateDeviceName) {
			return nil, "", err
		}
		logger.Error("database error creating device", slog.Any(AttrKeyError, err))
		return nil, "", err
	}

	logger.Info("device created", slog.Int64(AttrKeyDeviceID, createdDevice.ID.Int64()))

	return createdDevice, rawKey, nil
}

func (s *Service) Authenticate(ctx context.Context, rawKey string) (*Principal, error) {
	// Validate key format (must start with prefix)
	if len(rawKey) < len(APIKeyPrefix) || rawKey[:len(APIKeyPrefix)] != APIKeyPrefix {
		return nil, ErrInvalidAPIKey
	}

	// Hash the key
	keyHash := hashAPIKey(rawKey)

	// Look up device by key hash
	device, err := s.repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	return PrincipalFromDevice(device), nil
}

func (s *Service) AssignAddress(ctx context.Context, deviceID DeviceID, inputIP string, source StatusSource) (*Address, bool, error) {
	logger := logging.FromCtx(ctx)

	logger.Debug("assigning address")

	createAddressParams, err := NewCreateAddressParams(deviceID, inputIP)
	if err != nil {
		logger.Warn("invalid create address params", slog.Any(AttrKeyError, err))
		return nil, false, err
	}

	var address *Address
	var wasCreated bool

	err = s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
		if err != nil {
			if errors.Is(err, ErrDeviceNotFound) {
				logger.Warn("device not found")
				return err
			}
			logger.Error("database error fetching device", slog.Any(AttrKeyError, err))
			return err
		}

		address, err = tx.GetAddressForDeviceByIP(ctx, deviceID, createAddressParams.IP)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) {
				wasCreated = true
				logger.Debug("address not found, creating new address")
				address, err = tx.CreateAddress(ctx, createAddressParams)
				if err != nil {
					logger.Error("database error creating address", slog.Any(AttrKeyError, err))
					return err
				}
				logger.Info("address created", slog.Int64(AttrKeyAddressID, address.ID.Int64()))
			} else {
				logger.Error("database error checking address", slog.Any(AttrKeyError, err))
				return err
			}
		} else {
			wasCreated = false
			logger.Info("address exists add enabled status", slog.Int64(AttrKeyAddressID, address.ID.Int64()))
			address, err = tx.EnableAddress(ctx, address.ID, source)
			if err != nil {
				logger.Error("database error recording status change for existing address", slog.Any(AttrKeyError, err))
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	s.publishAddressEvent(ctx, NewAddressEvent(address, EventTypeAddressAssigned))
	s.signalAddressStateChanged(ctx)

	logger.Info("address assigned",
		slog.String(AttrKeyAddressIP, address.IP),
		slog.Int64(AttrKeyAddressID, address.ID.Int64()),
		slog.Bool(AttrKeyWasCreated, wasCreated),
	)

	return address, wasCreated, nil
}

func (s *Service) GetAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]Address, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("listing addresses for device")

	var addresses []Address

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
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

func (s *Service) DisableAddress(ctx context.Context, deviceID DeviceID, addressID AddressID) (*Address, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("disabling address")

	var disabledAddress *Address

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
		if err != nil {
			if errors.Is(err, ErrDeviceNotFound) {
				logger.Warn("device not found")
				return err
			}
			logger.Error("database error fetching device", slog.Any(AttrKeyError, err))
			return err
		}

		err = tx.CheckAddressOwnership(ctx, deviceID, addressID)
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

	s.publishAddressEvent(ctx, NewAddressEvent(disabledAddress, EventTypeAddressDisabled))
	s.signalAddressStateChanged(ctx)

	logger.Info("address disabled",
		slog.String(AttrKeyAddressIP, disabledAddress.IP),
		slog.Int64(AttrKeyAddressID, disabledAddress.ID.Int64()),
	)

	return disabledAddress, nil
}

func (s *Service) DisableAddresses(ctx context.Context, addressIDs []AddressID, source StatusSource) error {
	logger := logging.FromCtx(ctx)
	logger.Debug("disabling addresses")

	disabledAddresses, err := s.repo.DisableAddresses(ctx, addressIDs, source)
	if err != nil {
		logger.Error("database error disabling addresses", slog.Any(AttrKeyError, err))
		return err
	}

	for _, disabledAddress := range disabledAddresses {
		s.publishAddressEvent(ctx, NewAddressEvent(&disabledAddress, EventTypeAddressDisabled))
	}
	s.signalAddressStateChanged(ctx)

	logger.Info("addresses disabled",
		slog.Int(AttrKeyCount, len(disabledAddresses)),
	)

	return nil
}
