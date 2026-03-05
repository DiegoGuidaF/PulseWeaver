package device

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/logging"
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
	GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
	RunInTx(ctx context.Context, fn func(repository) error) error
}

type AddressObserver interface {
	OnAddressEvent(ctx context.Context, event AddressEvent)
}

type Service struct {
	repo      repository
	observers []AddressObserver
	logger    *slog.Logger
}

func NewService(repo repository, logger *slog.Logger) *Service {
	s := &Service{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "device")),
	}
	return s
}

func (s *Service) AddAddressObserver(o AddressObserver) {
	if o == nil {
		return
	}
	s.observers = append(s.observers, o)
}

func (s *Service) notifyObservers(ctx context.Context, event AddressEvent) {
	for _, o := range s.observers {
		o.OnAddressEvent(ctx, event)
	}
}

func (s *Service) GetDevices(ctx context.Context) ([]Device, error) {
	devices, err := s.repo.GetDevices(ctx)
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (s *Service) GetDevice(ctx context.Context, deviceID DeviceID) (*Device, error) {
	device, err := s.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (s *Service) DeleteDevice(ctx context.Context, deviceID DeviceID) error {
	err := s.repo.DeleteDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "device deleted", slog.Int64(AttrKeyDeviceID, deviceID.Int64()))
	return nil
}

func (s *Service) CreateDevice(ctx context.Context, name string) (*Device, string, error) {
	createDeviceParams, rawKey, err := NewCreateDeviceParams(name)
	if err != nil {
		return nil, "", err
	}

	createdDevice, err := s.repo.CreateDevice(ctx, createDeviceParams)
	if err != nil {
		return nil, "", err
	}

	s.logger.InfoContext(ctx, "device created", slog.Int64(AttrKeyDeviceID, createdDevice.ID.Int64()))

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
	createAddressParams, err := NewCreateAddressParams(deviceID, inputIP)
	if err != nil {
		return nil, false, err
	}

	var address *Address
	var wasCreated bool

	err = s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
		if err != nil {
			return err
		}

		address, err = tx.GetAddressForDeviceByIP(ctx, deviceID, createAddressParams.IP)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) {
				wasCreated = true
				address, err = tx.CreateAddress(ctx, createAddressParams)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			wasCreated = false
			address, err = tx.EnableAddress(ctx, address.ID, source)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}

	s.notifyObservers(ctx, NewAddressEvent(address, EventTypeAddressAssigned))

	s.logger.InfoContext(ctx, "address assigned",
		slog.String(AttrKeyAddressIP, address.IP),
		slog.Int64(AttrKeyAddressID, address.ID.Int64()),
		slog.Bool(AttrKeyWasCreated, wasCreated),
	)

	return address, wasCreated, nil
}

func (s *Service) GetAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]Address, error) {
	var addresses []Address

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
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

func (s *Service) DisableAddress(ctx context.Context, deviceID DeviceID, addressID AddressID) (*Address, error) {
	var disabledAddress *Address

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
		if err != nil {
			return err
		}

		err = tx.CheckAddressOwnership(ctx, deviceID, addressID)
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

	s.notifyObservers(ctx, NewAddressEvent(disabledAddress, EventTypeAddressDisabled))

	s.logger.InfoContext(ctx, "address disabled",
		slog.String(AttrKeyAddressIP, disabledAddress.IP),
		slog.Int64(AttrKeyAddressID, disabledAddress.ID.Int64()),
	)

	return disabledAddress, nil
}

func (s *Service) GetEnabledUniqueIPs(ctx context.Context) ([]string, error) {
	ips, err := s.repo.GetEnabledUniqueIPs(ctx)
	if err != nil {
		return nil, err
	}
	return ips, nil
}

func (s *Service) DisableAddresses(ctx context.Context, addressIDs []AddressID, source StatusSource) error {
	disabledAddresses, err := s.repo.DisableAddresses(ctx, addressIDs, source)
	if err != nil {
		return err
	}

	for _, disabledAddress := range disabledAddresses {
		s.notifyObservers(ctx, NewAddressEvent(&disabledAddress, EventTypeAddressDisabled))
	}

	s.logger.InfoContext(ctx, "addresses disabled",
		slog.Int(AttrKeyCount, len(disabledAddresses)),
	)

	return nil
}
