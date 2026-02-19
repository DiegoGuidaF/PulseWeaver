package auth

import "forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"

// Slog attribute key names for the auth domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyComponent   = logging.AttrKeyComponent
	AttrKeyOperation   = logging.AttrKeyOperation
	AttrKeyError       = logging.AttrKeyError
	AttrKeyUserID      = "user_id"
	AttrKeyDisplayName = "display_name"
	AttrKeySessionID   = "session_id"
	AttrKeyUsername    = "username"
)
