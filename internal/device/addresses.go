package device

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

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

func (s *Service) RegisterAddressActivity(ctx context.Context, deviceID ids.DeviceID, inputIP string, source EventSource) (*Address, EventType, error) {
	createAddressParams, err := NewCreateAddressParams(deviceID, inputIP, s.trustedProxy)
	if err != nil {
		return nil, "", err
	}

	var address *Address
	var eventType EventType

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// A disabled device may only have its addresses disabled — no address can be
		// created, enabled, or refreshed on it. This also validates existence
		// (ErrDeviceNotFound for a missing or deleted device).
		disabled, err := s.repo.IsDeviceDisabled(ctx, deviceID)
		if err != nil {
			return err
		}
		if disabled {
			return ErrDeviceDisabled
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

func (s *Service) DisableAddress(ctx context.Context, deviceID ids.DeviceID, addressID ids.AddressID) (*Address, error) {
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

func (s *Service) DisableAddresses(ctx context.Context, addressIDs []ids.AddressID, source EventSource) error {
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

func (s *Service) GetEnabledIPEntries(ctx context.Context) ([]IPEntry, error) {
	return s.repo.GetEnabledIPEntries(ctx)
}

// GetEnabledAddressesForDevice returns all enabled addresses for a device, ordered by updated_at DESC.
func (s *Service) GetEnabledAddressesForDevice(ctx context.Context, deviceID ids.DeviceID) ([]Address, error) {
	return s.repo.GetEnabledAddressesForDevice(ctx, deviceID)
}
