package lease

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

type TTLConfigRetriever interface {
	GetDeviceAddressLeaseTTLSeconds(ctx context.Context, deviceID device.DeviceID) (*int, error)
}
type repository interface {
	UpsertAddressLease(ctx context.Context, addressLease *AddressLease) (*AddressLease, error)
	DeleteAddressLeaseByAddressID(ctx context.Context, addressID device.AddressID) error
	GetExpiredAddressIDs(ctx context.Context) ([]device.AddressID, error)
}

type Service struct {
	repository         repository
	ttlConfigRetriever TTLConfigRetriever
	events             chan device.AddressEvent
	logger             *slog.Logger
}

// NewService creates a new lease service.
func NewService(repository repository, ttlConfigRetriever TTLConfigRetriever, logger *slog.Logger) *Service {
	return &Service{
		repository:         repository,
		ttlConfigRetriever: ttlConfigRetriever,
		events:             make(chan device.AddressEvent, 500),
		logger:             logger.With(slog.String(logging.AttrKeyComponent, "lease")),
	}
}

func (s *Service) AddAddressLease(ctx context.Context, deviceID device.DeviceID, addressID device.AddressID) (*AddressLease, error) {
	ctx = logging.WithOperation(ctx, "AddAddressLease")

	addressTTL, err := s.ttlConfigRetriever.GetDeviceAddressLeaseTTLSeconds(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// If no TTL found, do not add a lease
	if addressTTL == nil {
		return nil, nil
	}

	addressLease := NewAddressLease(addressID, *addressTTL)

	addressLease, err = s.repository.UpsertAddressLease(ctx, addressLease)
	if err != nil {
		return nil, err
	}

	return addressLease, nil
}

func (s *Service) DeleteAddressLease(ctx context.Context, addressID device.AddressID) error {
	ctx = logging.WithOperation(ctx, "DeleteAddressLease")

	err := s.repository.DeleteAddressLeaseByAddressID(ctx, addressID)
	if err != nil {
		// No lease found, not an error
		if errors.Is(err, ErrAddressLeaseNotFound) {
			return nil
		}
		return err
	}

	return nil
}

func (s *Service) GetExpiredAddressIDs(ctx context.Context) ([]device.AddressID, error) {
	ctx = logging.WithOperation(ctx, "GetExpiredAddressIDs")
	return s.repository.GetExpiredAddressIDs(ctx)
}

func (s *Service) OnAddressEvent(ctx context.Context, event device.AddressEvent) {
	ctx = logging.WithOperation(ctx, "OnAddressEvent")
	select {
	case <-ctx.Done():
		return
	case s.events <- event:
	}
}

// RunListener blocks and processes events. Run this in a goroutine.
func (s *Service) RunListener(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-s.events:
			if event.IsAddressEnabled() {
				if _, err := s.AddAddressLease(ctx, event.DeviceID, event.AddressID); err != nil {
					s.logger.ErrorContext(ctx, "failed to add address lease",
						slog.Any(AttrKeyError, err),
						slog.Int64(AttrKeyAddressID, event.AddressID.Int64()),
						slog.Int64(AttrKeyDeviceID, event.DeviceID.Int64()),
					)
				}
			} else {
				if err := s.DeleteAddressLease(ctx, event.AddressID); err != nil {
					s.logger.ErrorContext(ctx, "failed to delete address lease",
						slog.Any(AttrKeyError, err),
						slog.Int64(AttrKeyAddressID, event.AddressID.Int64()),
					)
				}
			}
		}
	}
}
