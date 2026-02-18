package auth

// Slog attribute key names for the auth domain. Use these constants when
// logging so keys are consistent and typo-safe across handlers and services.
const (
	AttrKeyDisplayName = "display_name"
	AttrKeyError       = "error"
	AttrKeyOperation   = "operation"
	AttrKeySessionID   = "session_id"
	AttrKeyUserID      = "user_id"
	AttrKeyUsername    = "username"
)
