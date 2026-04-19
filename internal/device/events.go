package device

import (
	"log/slog"
	"time"
)

type EventType string

const (
	EventTypeAddressCreated         EventType = "address_created"
	EventTypeAddressEnabled         EventType = "address_enabled"
	EventTypeAddressRefreshed       EventType = "address_refreshed"
	EventTypeAddressDisabled        EventType = "address_disabled"
	EventTypeDeviceOwnershipChanged EventType = "device_ownership_changed"
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

func NewDeviceEvent(deviceID DeviceID, eventType EventType) AddressEvent {
	return AddressEvent{
		Type:       eventType,
		DeviceID:   deviceID,
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

// IsAddressEnabled returns true for Created, Enabled, and Refreshed events (address is enabled);
// false for Disabled.
func (e AddressEvent) IsAddressEnabled() bool {
	switch e.Type {
	case EventTypeAddressCreated, EventTypeAddressEnabled, EventTypeAddressRefreshed:
		return true
	case EventTypeAddressDisabled:
		return false
	default:
		return false
	}
}
