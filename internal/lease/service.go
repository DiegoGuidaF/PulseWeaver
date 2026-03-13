package lease

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
)

type TTLConfigRetriever interface {
	GetDeviceAddressLeaseTTLSeconds(ctx context.Context, deviceID device.DeviceID) (*int, error)
}
type repository interface {
	UpsertAddressLease(ctx context.Context, addressLease *AddressLease) (*AddressLease, error)
	GetExpiredAddressIDs(ctx context.Context) ([]device.AddressID, error)
	SetDeviceAddressLeasesExpiry(ctx context.Context, deviceID device.DeviceID, expiresAt *time.Time, updatedAt time.Time) error
}

type Service struct {
	repository         repository
	ttlConfigRetriever TTLConfigRetriever
	events             chan device.AddressEvent
	ruleEvents         chan rule.RuleEvent
	logger             *slog.Logger
}

// NewService creates a new lease service.
func NewService(repository repository, ttlConfigRetriever TTLConfigRetriever, logger *slog.Logger) *Service {
	return &Service{
		repository:         repository,
		ttlConfigRetriever: ttlConfigRetriever,
		events:             make(chan device.AddressEvent, 500),
		ruleEvents:         make(chan rule.RuleEvent, 500),
		logger:             logger.With(slog.String(logging.AttrKeyComponent, "lease")),
	}
}

func (s *Service) AddAddressLease(ctx context.Context, deviceID device.DeviceID, addressID device.AddressID) (*AddressLease, error) {
	ctx = logging.WithOperation(ctx, "AddAddressLease")

	addressTTL, err := s.ttlConfigRetriever.GetDeviceAddressLeaseTTLSeconds(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	addressLease := NewAddressLease(addressID, deviceID, addressTTL)

	addressLease, err = s.repository.UpsertAddressLease(ctx, addressLease)
	if err != nil {
		return nil, err
	}

	return addressLease, nil
}

func (s *Service) ClearAddressLease(ctx context.Context, deviceID device.DeviceID, addressID device.AddressID) (*AddressLease, error) {
	ctx = logging.WithOperation(ctx, "ClearAddressLease")
	return s.repository.UpsertAddressLease(ctx, NewAddressLease(addressID, deviceID, nil))
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

func (s *Service) OnRuleEvent(ctx context.Context, event rule.RuleEvent) {
	ctx = logging.WithOperation(ctx, "OnRuleEvent")
	select {
	case <-ctx.Done():
	case s.ruleEvents <- event:
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
					s.logger.ErrorContext(ctx, "failed to upsert address lease",
						slog.Any(AttrKeyError, err),
						slog.Int64(AttrKeyAddressID, event.AddressID.Int64()),
						slog.Int64(AttrKeyDeviceID, event.DeviceID.Int64()),
					)
				}
			} else {
				if _, err := s.ClearAddressLease(ctx, event.DeviceID, event.AddressID); err != nil {
					s.logger.ErrorContext(ctx, "failed to clear address lease",
						slog.Any(AttrKeyError, err),
						slog.Int64(AttrKeyAddressID, event.AddressID.Int64()),
						slog.Int64(AttrKeyDeviceID, event.DeviceID.Int64()),
					)
				}
			}
		case event := <-s.ruleEvents:
			s.handleLeaseRuleEvent(ctx, event)
		}
	}
}

func (s *Service) handleLeaseRuleEvent(ctx context.Context, event rule.RuleEvent) {
	if event.RuleType != rule.RuleTypeDeviceAddressLease {
		return
	}

	expiresAt := expiresAtFromTTL(event.OccurredAt, event.TTLSeconds)

	if err := s.repository.SetDeviceAddressLeasesExpiry(ctx, event.DeviceID, expiresAt, event.OccurredAt); err != nil {
		s.logger.ErrorContext(ctx, "failed to sync device leases on rule event",
			slog.Any(AttrKeyError, err),
			slog.Int64(AttrKeyDeviceID, event.DeviceID.Int64()),
		)
	}
}
