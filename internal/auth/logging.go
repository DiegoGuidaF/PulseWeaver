package auth

// Slog attribute key names for the auth domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyOperation   = "operation"
	AttrKeyError       = "error"
	AttrKeyUserID      = "user_id"
	AttrKeyDisplayName = "display_name"
	AttrKeySessionID   = "session_id"
	AttrKeyUsername    = "username"
)
