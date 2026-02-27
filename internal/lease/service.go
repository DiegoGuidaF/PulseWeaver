package lease

import (
	"context"
	"errors"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
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
}

// NewService creates a new lease service.
func NewService(repository repository, ttlConfigRetriever TTLConfigRetriever) *Service {
	return &Service{
		repository:         repository,
		ttlConfigRetriever: ttlConfigRetriever,
	}
}

func (s *Service) AddAddressLease(ctx context.Context, deviceID device.DeviceID, addressID device.AddressID) (*AddressLease, error) {
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
	return s.repository.GetExpiredAddressIDs(ctx)
}

// RunListener blocks and processes events. Run this in a goroutine.
func (s *Service) RunListener(ctx context.Context, events <-chan device.AddressEvent) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events:
			switch event.Type {
			case device.EventTypeAddressAssigned:
				_, _ = s.AddAddressLease(ctx, event.DeviceID, event.AddressID)
			case device.EventTypeAddressDisabled:
				_ = s.DeleteAddressLease(ctx, event.AddressID)
			}
		}
	}
}
