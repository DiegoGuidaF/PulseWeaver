package auth

type Principal struct {
	UserID    UserID
	SessionID SessionID
	Role      Role
}

func NewPrincipal(userID UserID, sessionID SessionID, role Role) *Principal {
	return &Principal{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
	}
}

func PrincipalFromSession(session *SessionWithUser) *Principal {
	return NewPrincipal(session.UserID, session.ID, session.UserRole)
}

func (principal Principal) isAdmin() bool {
	return principal.Role == AdminRole
}
