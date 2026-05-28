package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type repository interface {
	GetDevice(ctx context.Context, id ids.DeviceID) (*Device, error)
	GetDeviceIDsByOwner(ctx context.Context, ownerID ids.UserID) ([]ids.DeviceID, error)
	CreateDevice(ctx context.Context, params CreateDeviceParams) (*Device, error)
	DeleteDevice(ctx context.Context, id ids.DeviceID) error
	UpdateDevice(ctx context.Context, device *Device) (*Device, error)
	UpsertAPIKey(ctx context.Context, deviceID ids.DeviceID, keyHash string, keyPrefix string) error
	DeleteAPIKey(ctx context.Context, deviceID ids.DeviceID) error
	CreateAddress(ctx context.Context, params CreateAddressParams, source EventSource) (*Address, error)
	GetAddressForDeviceByIP(ctx context.Context, deviceID ids.DeviceID, ip netip.Addr) (*Address, error)
	DisableAddress(ctx context.Context, addressID ids.AddressID) (*Address, error)
	DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source EventSource) ([]Address, error)
	EnableAddress(ctx context.Context, addressID ids.AddressID, source EventSource) (*Address, error)
	RefreshAddress(ctx context.Context, addressID ids.AddressID, source EventSource) (*Address, error)
	CheckAddressOwnership(ctx context.Context, deviceID ids.DeviceID, addressID ids.AddressID) error
	GetDeviceByAPIKeyHash(ctx context.Context, keyHash string) (*Device, error)
	GetEnabledIPEntries(ctx context.Context) ([]IPEntry, error)
	GetAddressHistory(ctx context.Context, query AddressHistoryQuery) (AddressHistory, error)
	GetEnabledAddressesForDevice(ctx context.Context, deviceID ids.DeviceID) ([]Address, error)
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

func (s *Service) GetDevice(ctx context.Context, deviceID ids.DeviceID) (*Device, error) {
	device, err := s.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (s *Service) CreateDevice(ctx context.Context, principal *auth.Principal, name string, requestedOwnerID *ids.UserID) (*Device, error) {
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

func (s *Service) DeleteDevice(ctx context.Context, deviceID ids.DeviceID) error {
	var disabledAddresses []Address
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		err := s.repo.DeleteDevice(ctx, deviceID)
		if err != nil {
			return err
		}
		addresses, err := s.repo.GetEnabledAddressesForDevice(ctx, deviceID)
		if err != nil {
			return err
		}
		addressesToDisable := make([]ids.AddressID, 0, len(addresses))
		for _, address := range addresses {
			addressesToDisable = append(addressesToDisable, address.ID)
		}

		// Disable currently active addresses
		disabledAddresses, err = s.repo.DisableAddresses(ctx, addressesToDisable, EventSourceManual)
		if err != nil {
			return err
		}

		// Disable the API to be sure it can't be used
		err = s.repo.DeleteAPIKey(ctx, deviceID)
		if err != nil {
			// No error if there's no API key to delete here
			if errors.Is(err, ErrNoAPIKey) {
				return nil
			}
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, disabledAddress := range disabledAddresses {
		s.notifyObservers(ctx, NewAddressEvent(&disabledAddress, EventTypeAddressDisabled))
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
	OwnerID     *ids.UserID
}

func (s *Service) UpdateDevice(ctx context.Context, deviceID ids.DeviceID, input UpdateDeviceInput) (*Device, error) {
	device, err := s.repo.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	ownershipChanged := input.OwnerID != nil && *input.OwnerID != device.OwnerID

	if err := device.Update(input.Name, input.DeviceType, input.Description, input.Icon, input.OwnerID); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateDevice(ctx, device)
	if err != nil {
		return nil, err
	}

	if ownershipChanged {
		s.notifyObservers(ctx, NewDeviceEvent(deviceID, EventTypeDeviceOwnershipChanged))
	}

	return updated, nil
}

func (s *Service) RegenerateAPIKey(ctx context.Context, deviceID ids.DeviceID) (*Device, string, error) {
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

func (s *Service) DeleteAPIKey(ctx context.Context, deviceID ids.DeviceID) error {
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

// OnUserEvent implements auth.UserObserver. On deletion it soft-deletes every
// device owned by the user and disables their addresses, firing the
// AddressDisabled events that trigger a policy cache refresh.
func (s *Service) OnUserEvent(ctx context.Context, event auth.UserEvent) {
	if event.Type != auth.EventTypeUserDeleted {
		return
	}
	_ = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		deviceIDs, err := s.repo.GetDeviceIDsByOwner(ctx, event.UserID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get devices for deleted user",
				slog.Int64("user_id", event.UserID.Int64()),
				slog.Any(logging.AttrKeyError, err),
			)
		}
		for _, deviceID := range deviceIDs {
			if err := s.DeleteDevice(ctx, deviceID); err != nil {
				s.logger.ErrorContext(ctx, "failed to delete device for deleted user",
					slog.Int64(AttrKeyDeviceID, deviceID.Int64()),
					slog.Any(logging.AttrKeyError, err),
				)
			}
		}
		return nil
	})
}
