package rule

import "github.com/DiegoGuidaF/PulseWeaver/internal/logging"

// Slog attribute key names for the rule domain. Use these constants when
// logging so keys are consistent and typo-safe across services.
const (
	AttrKeyComponent = logging.AttrKeyComponent
	AttrKeyOperation = logging.AttrKeyOperation
	AttrKeyError     = logging.AttrKeyError

	AttrKeyRuleID               = "rule_id"
	AttrKeyRuleType             = "rule_type"
	AttrKeyDeviceID             = "device_id"
	AttrDeviceAutoExpiryRuleTTL = "device_autoexpiry_ttl_sec"
)
