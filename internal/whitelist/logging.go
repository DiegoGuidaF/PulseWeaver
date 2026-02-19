package whitelist

import "forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"

// Slog attribute key names for the whitelist domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyComponent     = logging.AttrKeyComponent
	AttrKeyError         = logging.AttrKeyError
	AttrKeyWhitelistFile = "whitelist_file"
	AttrKeyIPCount       = "ip_count"
)
