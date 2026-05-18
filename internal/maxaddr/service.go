package maxaddr

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
)

// MaxAddressesProvider is implemented by *rule.Service.
type MaxAddressesProvider interface {
	GetMaxActiveAddresses(ctx context.Context, deviceID ids.DeviceID) (*int, error)
}

// EnabledAddressFetcher is implemented by *device.Service.
type EnabledAddressFetcher interface {
	GetEnabledAddressesForDevice(ctx context.Context, deviceID ids.DeviceID) ([]device.Address, error)
}

// AddressDisabler is implemented by *device.Service.
type AddressDisabler interface {
	DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source device.EventSource) error
}

// Service listens for address events and enforces the max active addresses rule asynchronously.
type Service struct {
	provider   MaxAddressesProvider
	fetcher    EnabledAddressFetcher
	disabler   AddressDisabler
	events     chan device.AddressEvent
	ruleEvents chan rule.RuleEvent
	logger     *slog.Logger
}

// NewService creates a new maxaddr enforcement service.
func NewService(provider MaxAddressesProvider, fetcher EnabledAddressFetcher, disabler AddressDisabler, logger *slog.Logger) *Service {
	return &Service{
		provider:   provider,
		fetcher:    fetcher,
		disabler:   disabler,
		events:     make(chan device.AddressEvent, 500),
		ruleEvents: make(chan rule.RuleEvent, 100),
		logger:     logger.With(slog.String(logging.AttrKeyComponent, "maxaddr")),
	}
}

// OnAddressEvent implements device.AddressObserver. It filters events before enqueuing:
// only Created and Enabled events trigger enforcement.
func (s *Service) OnAddressEvent(ctx context.Context, event device.AddressEvent) {
	logging.WithOperation(ctx, "OnAddressEvent")
	if event.Type != device.EventTypeAddressCreated && event.Type != device.EventTypeAddressEnabled {
		return
	}
	select {
	case s.events <- event:
	default:
		s.logger.Warn("max active addresses event channel full, dropping event",
			slog.Int64("device_id", event.DeviceID.Int64()),
		)
	}
}

// OnRuleEvent implements rule.RuleObserver. It triggers enforcement when the
// max active addresses rule is enabled or updated.
func (s *Service) OnRuleEvent(ctx context.Context, event rule.RuleEvent) {
	if event.RuleType != rule.RuleTypeMaxActiveAddresses || event.Type != rule.RuleEventTypeEnabled {
		return
	}
	select {
	case s.ruleEvents <- event:
	default:
		s.logger.Warn("max active addresses rule event channel full, dropping event",
			slog.Int64("device_id", event.DeviceID.Int64()),
		)
	}
}

// RunListener processes address and rule events until the context is cancelled.
// Run this in a goroutine.
func (s *Service) RunListener(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-s.events:
			s.enforce(ctx, event.DeviceID, event.AddressID)
		case event := <-s.ruleEvents:
			// No justRegisteredID to protect — evict purely by updated_at order.
			s.enforce(ctx, event.DeviceID, 0)
		}
	}
}

// enforce applies the max active addresses rule for the given device, evicting the
// least-recently-updated addresses while protecting justRegisteredID.
func (s *Service) enforce(ctx context.Context, deviceID ids.DeviceID, justRegisteredID ids.AddressID) {
	maxAddresses, err := s.provider.GetMaxActiveAddresses(ctx, deviceID)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get max active addresses rule", slog.Any("error", err))
		return
	}
	if maxAddresses == nil {
		return
	}

	enabledAddresses, err := s.fetcher.GetEnabledAddressesForDevice(ctx, deviceID)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to get enabled addresses for enforcement", slog.Any("error", err))
		return
	}

	excess := len(enabledAddresses) - *maxAddresses
	if excess <= 0 {
		return
	}

	toDisable := make([]ids.AddressID, 0, excess)

	// Traverse from oldest (end of slice) to newest.
	for i := len(enabledAddresses) - 1; i >= 0; i-- {
		if enabledAddresses[i].ID == justRegisteredID {
			continue
		}

		toDisable = append(toDisable, enabledAddresses[i].ID)

		// Stop once we've collected the exact number of addresses to evict.
		if len(toDisable) == excess {
			break
		}
	}

	if len(toDisable) == 0 {
		return
	}
	s.logger.DebugContext(ctx, "exceeded max active addresses for device, dropping addresses", slog.Any("addresses", toDisable))

	if err := s.disabler.DisableAddresses(ctx, toDisable, device.EventSourceLimitExceeded); err != nil {
		s.logger.WarnContext(ctx, "failed to evict addresses for max active rule",
			slog.Any("error", err),
			slog.Int("to_evict", len(toDisable)),
		)
	}
}
