package device

import "github.com/DiegoGuidaF/PulseWeaver/internal/logging"

// Slog attribute key names for the device domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyError            = logging.AttrKeyError
	AttrKeyAddressID        = "address_id"
	AttrKeyAddressIP        = "address_ip"
	AttrKeyClientIP         = "client_ip"
	AttrKeyCount            = "count"
	AttrKeyDeviceID         = "device_id"
	AttrKeyDeviceName       = "device_name"
	AttrKeyAddressEventType = "address_event_type"
)
