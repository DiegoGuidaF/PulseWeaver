package device

import "github.com/DiegoGuidaF/WallyDex/internal/logging"

// Slog attribute key names for the device domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyComponent  = logging.AttrKeyComponent
	AttrKeyError      = logging.AttrKeyError
	AttrKeyOperation  = logging.AttrKeyOperation
	AttrKeyAddressID  = "address_id"
	AttrKeyAddressIP  = "address_ip"
	AttrKeyClientIP   = "client_ip"
	AttrKeyCount      = "count"
	AttrKeyWasCreated = "was_created"
	AttrKeyDeviceID   = "device_id"
	AttrKeyDeviceName = "device_name"
)
