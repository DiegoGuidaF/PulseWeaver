package auth

type Principal struct {
	UserID    UserID
	SessionID SessionID // device token id, for audit
	DeviceID  *string   // nil for browser session
}
