package auth

type Principal struct {
	UserID    UserID
	DeviceID  *string    // nil for browser session
	SessionID *SessionID // device token id, for audit
}
