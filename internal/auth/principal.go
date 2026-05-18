package auth

import "github.com/DiegoGuidaF/PulseWeaver/internal/ids"

type Principal struct {
	UserID    ids.UserID
	SessionID ids.SessionID
	Role      Role
}

func NewPrincipal(userID ids.UserID, sessionID ids.SessionID, role Role) *Principal {
	return &Principal{
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
	}
}

func PrincipalFromSession(session *SessionWithUser) *Principal {
	return NewPrincipal(session.UserID, session.ID, session.UserRole)
}

func (principal Principal) IsAdmin() bool {
	return principal.Role == AdminRole || principal.IsSuperAdmin()
}

func (principal Principal) IsSuperAdmin() bool {
	return principal.Role == SuperAdminRole
}
