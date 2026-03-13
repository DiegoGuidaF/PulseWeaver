package scheduler

import "github.com/DiegoGuidaF/PulseWeaver/internal/logging"

// Slog attribute key names for the scheduler domain. Use these constants when
// logging so keys are consistent and typo-safe across services.
const (
	AttrKeyComponent = logging.AttrKeyComponent
	AttrKeyOperation = logging.AttrKeyOperation
	AttrKeyError     = logging.AttrKeyError

	AttrKeyRuleID   = "rule_id"
	AttrKeyRuleType = "rule_type"
	AttrKeyDeviceID = "device_id"
	AttrKeyCount    = "count"
)
