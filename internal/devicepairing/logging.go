package devicepairing

import "github.com/DiegoGuidaF/PulseWeaver/internal/logging"

// Slog attribute key names for the device pairing domain. Use these constants
// when logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyComponent = logging.AttrKeyComponent
	AttrKeyOperation = logging.AttrKeyOperation
	AttrKeyError     = logging.AttrKeyError
)
