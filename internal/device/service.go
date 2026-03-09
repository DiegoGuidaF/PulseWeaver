package device

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"

	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

type repository interface {
	GetDevice(ctx context.Context, id DeviceID) (*Device, error)
	CreateDevice(ctx context.Context, params *CreateDeviceParams) (*Device, error)
	GetDevices(ctx context.Context) ([]Device, error)
	DeleteDevice(ctx context.Context, id DeviceID) error
	CreateAddress(ctx context.Context, params *CreateAddressParams) (*Address, error)
	GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip netip.Addr) (*Address, error)
	DisableAddress(ctx context.Context, addressID AddressID) (*Address, error)
	DisableAddresses(ctx context.Context, addressIDs []AddressID, source EventSource) ([]Address, error)
	EnableAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error)
	RefreshAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error)
	CheckAddressOwnership(ctx context.Context, deviceID DeviceID, addressID AddressID) error
	GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error)
	GetEnabledUniqueIPs(ctx context.Context) ([]string, error)
	RunInTx(ctx context.Context, fn func(repository) error) error
}

type AddressObserver interface {
	OnAddressEvent(ctx context.Context, event AddressEvent)
}

type Service struct {
	repo         repository
	observers    []AddressObserver
	logger       *slog.Logger
	trustedProxy netip.Addr
}

func NewService(repo repository, logger *slog.Logger, trustedProxy netip.Addr) *Service {
	s := &Service{
		repo:         repo,
		logger:       logger.With(slog.String(logging.AttrKeyComponent, "device")),
		trustedProxy: trustedProxy,
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

func (s *Service) RegisterAddressActivity(ctx context.Context, deviceID DeviceID, inputIP string, source EventSource) (*Address, EventType, error) {
	createAddressParams, err := NewCreateAddressParams(deviceID, inputIP, s.trustedProxy)
	if err != nil {
		return nil, "", err
	}

	var address *Address
	var eventType EventType

	err = s.repo.RunInTx(ctx, func(tx repository) error {
		_, err := tx.GetDevice(ctx, deviceID)
		if err != nil {
			return err
		}

		existingAddress, err := tx.GetAddressForDeviceByIP(ctx, deviceID, createAddressParams.IP)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) {
				eventType = EventTypeAddressCreated
				address, err = tx.CreateAddress(ctx, createAddressParams)
				if err != nil {
					return err
				}
				return nil
			}
			return err
		}

		switch {
		case !existingAddress.IsEnabled:
			eventType = EventTypeAddressEnabled
			address, err = tx.EnableAddress(ctx, existingAddress.ID, source)
		case existingAddress.IsEnabled:
			eventType = EventTypeAddressRefreshed
			address, err = tx.RefreshAddress(ctx, existingAddress.ID, source)
		}
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	s.notifyObservers(ctx, NewAddressEvent(address, eventType))

	s.logger.InfoContext(ctx, "address activity registered",
		slog.String(AttrKeyAddressIP, address.IP),
		slog.Int64(AttrKeyAddressID, address.ID.Int64()),
		slog.String("event_type", string(eventType)),
	)

	return address, eventType, nil
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

func (s *Service) DisableAddresses(ctx context.Context, addressIDs []AddressID, source EventSource) error {
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
