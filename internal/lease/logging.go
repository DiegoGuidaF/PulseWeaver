package lease

import "forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"

// Slog attribute key names for the lease domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyError     = logging.AttrKeyError
	AttrKeyAddressID = "address_id"
	AttrKeyDeviceID  = "device_id"
	AttrKeyEventType = "event_type"
)
