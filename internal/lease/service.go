package lease

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
)

type TTLConfigRetriever interface {
	GetDeviceAddressLeaseTTLSeconds(ctx context.Context, deviceID ids.DeviceID) (*int, error)
}
type repository interface {
	UpsertAddressLease(ctx context.Context, addressLease *AddressLease) (*AddressLease, error)
	DeleteAddressLease(ctx context.Context, addressID ids.AddressID) error
	GetExpiredAddressIDs(ctx context.Context) ([]ids.AddressID, error)
	SetDeviceAddressLeasesExpiry(ctx context.Context, deviceID ids.DeviceID, expiresAt *time.Time, updatedAt time.Time) error
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

func (s *Service) AddAddressLease(ctx context.Context, deviceID ids.DeviceID, addressID ids.AddressID) (*AddressLease, error) {
	ctx = logging.WithOperation(ctx, "AddAddressLease")

	addressTTL, err := s.ttlConfigRetriever.GetDeviceAddressLeaseTTLSeconds(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	addressLease, err := s.repository.UpsertAddressLease(ctx, new(NewAddressLease(addressID, deviceID, addressTTL)))
	if err != nil {
		return nil, err
	}

	return addressLease, nil
}

func (s *Service) ClearAddressLease(ctx context.Context, deviceID ids.DeviceID, addressID ids.AddressID) error {
	ctx = logging.WithOperation(ctx, "ClearAddressLease")
	return s.repository.DeleteAddressLease(ctx, addressID)
}

func (s *Service) GetExpiredAddressIDs(ctx context.Context) ([]ids.AddressID, error) {
	ctx = logging.WithOperation(ctx, "GetExpiredAddressIDs")
	return s.repository.GetExpiredAddressIDs(ctx)
}

func (s *Service) OnAddressEvent(ctx context.Context, event device.AddressEvent) {
	ctx = logging.WithOperation(ctx, "OnAddressEvent")
	select {
	case <-ctx.Done():
		return
	case s.events <- event:
	default:
		s.logger.Warn("address lease event channel full, dropping event",
			slog.Int64("device_id", event.DeviceID.Int64()),
		)
	}
}

func (s *Service) OnRuleEvent(ctx context.Context, event rule.RuleEvent) {
	ctx = logging.WithOperation(ctx, "OnRuleEvent")
	select {
	case <-ctx.Done():
	case s.ruleEvents <- event:
	default:
		s.logger.Warn("address lease rule change channel full, dropping event",
			slog.Int64("device_id", event.DeviceID.Int64()),
		)
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
				if err := s.ClearAddressLease(ctx, event.DeviceID, event.AddressID); err != nil {
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
