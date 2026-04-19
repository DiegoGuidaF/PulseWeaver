package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type repository interface {
	GetDevice(ctx context.Context, id DeviceID) (*Device, error)
	CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error)
	DeleteDevice(ctx context.Context, id DeviceID) error
	UpdateDevice(ctx context.Context, device *Device) (*Device, error)
	UpsertAPIKey(ctx context.Context, deviceID DeviceID, keyHash string, keyPrefix string) error
	DeleteAPIKey(ctx context.Context, deviceID DeviceID) error
	CreateAddress(ctx context.Context, params CreateAddressParams, source EventSource) (*Address, error)
	GetAddressForDeviceByIP(ctx context.Context, deviceID DeviceID, ip netip.Addr) (*Address, error)
	DisableAddress(ctx context.Context, addressID AddressID) (*Address, error)
	DisableAddresses(ctx context.Context, addressIDs []AddressID, source EventSource) ([]Address, error)
	EnableAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error)
	RefreshAddress(ctx context.Context, addressID AddressID, source EventSource) (*Address, error)
	CheckAddressOwnership(ctx context.Context, deviceID DeviceID, addressID AddressID) error
	GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error)
	GetEnabledIPEntries(ctx context.Context) ([]IPEntry, error)
	GetAddressHistory(ctx context.Context, query AddressHistoryQuery) (AddressHistory, error)
	GetEnabledAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]Address, error)
}

type AddressObserver interface {
	OnAddressEvent(ctx context.Context, event AddressEvent)
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repo         repository
	tx           transactor
	observers    []AddressObserver
	logger       *slog.Logger
	trustedProxy netip.Addr
}

func NewService(repo repository, transactor transactor, logger *slog.Logger, trustedProxy netip.Addr) *Service {
	s := &Service{
		repo:         repo,
		tx:           transactor,
		logger:       logger.With(slog.String(logging.AttrKeyComponent, "device")),
		trustedProxy: trustedProxy,
	}
	return s
}

func (s *Service) Authenticate(ctx context.Context, rawKey string) (*Principal, error) {
	// Validate key format (must start with prefix)
	if len(rawKey) < len(APIKeyPrefix) || rawKey[:len(APIKeyPrefix)] != APIKeyPrefix {
		return nil, ErrInvalidAPIKey
	}

	// Hash the key
	keyHash := HashAPIKey(rawKey)

	// Look up device by key hash
	device, err := s.repo.GetDeviceByAPIKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	return PrincipalFromDevice(device), nil
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

func (s *Service) GetDevice(ctx context.Context, deviceID DeviceID) (*Device, error) {
	device, err := s.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (s *Service) CreateDevice(ctx context.Context, principal *auth.Principal, name string, requestedOwnerID *auth.UserID) (*Device, error) {
	ownerID := principal.UserID
	if requestedOwnerID != nil {
		ownerID = *requestedOwnerID
	}

	createdDevice, err := s.repo.CreateDevice(ctx, CreateDeviceParams{Name: name, OwnerID: ownerID})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "device created", slog.Int64(AttrKeyDeviceID, createdDevice.ID.Int64()))

	return createdDevice, nil
}

func (s *Service) DeleteDevice(ctx context.Context, deviceID DeviceID) error {
	err := s.repo.DeleteDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "device deleted", slog.Int64(AttrKeyDeviceID, deviceID.Int64()))
	return nil
}

// UpdateDeviceInput carries the raw nullable API values for a device profile update.
// nil pointer = field was absent in the request (leave unchanged).
// For Description and Icon, **string semantics apply: nil = absent, *nil = clear, *&s = set.
type UpdateDeviceInput struct {
	Name        *string
	DeviceType  *string
	Description **string
	Icon        **string
	OwnerID     *auth.UserID
}

func (s *Service) UpdateDevice(ctx context.Context, deviceID DeviceID, input UpdateDeviceInput) (*Device, error) {
	device, err := s.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if err := device.Update(input.Name, input.DeviceType, input.Description, input.Icon, input.OwnerID); err != nil {
		return nil, err
	}

	return s.repo.UpdateDevice(ctx, device)
}

func (s *Service) GetAddressHistory(ctx context.Context, query AddressHistoryQuery) (AddressHistory, error) {
	if err := query.Validate(); err != nil {
		return AddressHistory{}, err
	}
	history, err := s.repo.GetAddressHistory(ctx, query)
	if err != nil {
		return AddressHistory{}, err
	}
	history.QueryLimit = query.Limit
	return history, nil
}

func (s *Service) RegisterAddressActivity(ctx context.Context, deviceID DeviceID, inputIP string, source EventSource) (*Address, EventType, error) {
	createAddressParams, err := NewCreateAddressParams(deviceID, inputIP, s.trustedProxy)
	if err != nil {
		return nil, "", err
	}

	var address *Address
	var eventType EventType

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, err := s.repo.GetDevice(ctx, deviceID)
		if err != nil {
			return err
		}

		existingAddress, err := s.repo.GetAddressForDeviceByIP(ctx, deviceID, createAddressParams.IP)
		if err != nil {
			if errors.Is(err, ErrAddressNotFound) {
				eventType = EventTypeAddressCreated
				address, err = s.repo.CreateAddress(ctx, createAddressParams, source)
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
			address, err = s.repo.EnableAddress(ctx, existingAddress.ID, source)
		case existingAddress.IsEnabled:
			eventType = EventTypeAddressRefreshed
			address, err = s.repo.RefreshAddress(ctx, existingAddress.ID, source)
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

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, err := s.repo.GetDevice(ctx, deviceID)
		if err != nil {
			return err
		}

		err = s.repo.CheckAddressOwnership(ctx, deviceID, addressID)
		if err != nil {
			return err
		}

		disabledAddress, err = s.repo.DisableAddress(ctx, addressID)
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

func (s *Service) RegenerateAPIKey(ctx context.Context, deviceID DeviceID) (*Device, string, error) {
	rawKey, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate api key: %w", err)
	}

	var device *Device
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		// Validate device exists (also checks deleted_at) inside the transaction
		// so the existence check and key upsert are atomic.
		if _, err = s.repo.GetDevice(ctx, deviceID); err != nil {
			return err
		}
		if err = s.repo.UpsertAPIKey(ctx, deviceID, keyHash, keyPrefix); err != nil {
			return err
		}
		// Fetch fresh device inside the transaction so KeyPrefix reflects the upsert.
		device, err = s.repo.GetDevice(ctx, deviceID)
		return err
	})
	if err != nil {
		return nil, "", err
	}

	return device, rawKey, nil
}

func (s *Service) DeleteAPIKey(ctx context.Context, deviceID DeviceID) error {
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Validate device exists before attempting key deletion.
		if _, err := s.repo.GetDevice(ctx, deviceID); err != nil {
			return err
		}
		return s.repo.DeleteAPIKey(ctx, deviceID)
	})
	if err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "device api key deleted", slog.Int64(AttrKeyDeviceID, deviceID.Int64()))
	return nil
}

func (s *Service) GetEnabledIPEntries(ctx context.Context) ([]IPEntry, error) {
	return s.repo.GetEnabledIPEntries(ctx)
}

// GetEnabledAddressesForDevice returns all enabled addresses for a device, ordered by updated_at DESC.
func (s *Service) GetEnabledAddressesForDevice(ctx context.Context, deviceID DeviceID) ([]Address, error) {
	return s.repo.GetEnabledAddressesForDevice(ctx, deviceID)
}

// CreateDeviceWithAPIKey creates a new device and assigns it an API key, all within a single
// transaction. It is the entry point for the registration domain when claiming an invite.
// Returns the new device ID and the plaintext API key (one-time; never stored after this call).
func (s *Service) CreateDeviceWithAPIKey(ctx context.Context, name string, ownerID auth.UserID) (DeviceID, string, error) {
	var deviceID DeviceID
	var rawAPIKey string

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		rawKey, keyHash, keyPrefix, err := GenerateAPIKey()
		if err != nil {
			return fmt.Errorf("generate api key: %w", err)
		}

		dev, err := s.repo.CreateDevice(ctx, CreateDeviceParams{
			Name:       name,
			OwnerID:    ownerID,
			DeviceType: "mobile",
		})
		if err != nil {
			return err
		}

		if err := s.repo.UpsertAPIKey(ctx, dev.ID, keyHash, keyPrefix); err != nil {
			return err
		}

		deviceID = dev.ID
		rawAPIKey = rawKey
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	return deviceID, rawAPIKey, nil
}
