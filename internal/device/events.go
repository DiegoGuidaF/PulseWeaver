package device

import (
	"log/slog"
	"time"
)

type EventType string

const (
	EventTypeAddressAssigned EventType = "address_assigned"
	EventTypeAddressDisabled EventType = "address_disabled"
)

type AddressEvent struct {
	Type       EventType
	AddressID  AddressID
	DeviceID   DeviceID
	OccurredAt time.Time
}

func NewAddressEvent(address *Address, eventType EventType) AddressEvent {
	return AddressEvent{
		Type:       eventType,
		AddressID:  address.ID,
		DeviceID:   address.DeviceID,
		OccurredAt: time.Now().UTC(),
	}
}

func (e AddressEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(e.Type)),
		slog.Int64("address_id", e.AddressID.Int64()),
		slog.Int64("device_id", e.DeviceID.Int64()),
		slog.Time("occurred_at", e.OccurredAt),
	)
}
