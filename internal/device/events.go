package device

import (
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

type AddressEventType string

const (
	EventTypeAddressCreated   AddressEventType = "address_created"
	EventTypeAddressEnabled   AddressEventType = "address_enabled"
	EventTypeAddressRefreshed AddressEventType = "address_refreshed"
	EventTypeAddressDisabled  AddressEventType = "address_disabled"
)

type DeviceEventType string

const (
	DeviceEventTypeOwnershipChanged DeviceEventType = "device_ownership_changed"
)

type AddressEvent struct {
	Type       AddressEventType
	AddressID  ids.AddressID
	DeviceID   ids.DeviceID
	OccurredAt time.Time
}

func NewAddressEvent(address *Address, eventType AddressEventType) AddressEvent {
	return AddressEvent{
		Type:       eventType,
		AddressID:  address.ID,
		DeviceID:   address.DeviceID,
		OccurredAt: time.Now().UTC(),
	}
}

// DeviceEvent signals a device-level change.
type DeviceEvent struct {
	Type       DeviceEventType
	DeviceID   ids.DeviceID
	OccurredAt time.Time
}

func NewDeviceEvent(deviceID ids.DeviceID, eventType DeviceEventType) DeviceEvent {
	return DeviceEvent{
		Type:       eventType,
		DeviceID:   deviceID,
		OccurredAt: time.Now().UTC(),
	}
}

func (e DeviceEvent) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("type", string(e.Type)),
		slog.Int64("device_id", e.DeviceID.Int64()),
		slog.Time("occurred_at", e.OccurredAt),
	)
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
