package auth

import (
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

const tokenDuration = time.Hour * 24 * 7

type Session struct {
	ID         ids.SessionID `db:"id" `
	UserID     ids.UserID    `db:"user_id" `
	TokenHash  string        `db:"token_hash" `
	CreatedAt  time.Time     `db:"created_at" `
	ExpiresAt  time.Time     `db:"expires_at" `
	LastUsedAt *time.Time    `db:"last_used_at" `
	RevokedAt  *time.Time    `db:"revoked_at" `
}

func NewSession(userID ids.UserID, tokenHash string) Session {
	return Session{
		UserID:     userID,
		TokenHash:  tokenHash,
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(tokenDuration),
		LastUsedAt: nil,
		RevokedAt:  nil,
	}
}

type SessionWithUser struct {
	Session
	UserRole Role `db:"user_role"` // Alias in SQL Join
}
