package auth

import (
	"strconv"
	"time"
)

const tokenDuration = time.Hour * 24 * 7

type Session struct {
	ID         SessionID  `db:"id" `
	UserID     UserID     `db:"user_id" `
	TokenHash  string     `db:"token_hash" `
	CreatedAt  time.Time  `db:"created_at" `
	ExpiresAt  time.Time  `db:"expires_at" `
	LastUsedAt *time.Time `db:"last_used_at" `
	RevokedAt  *time.Time `db:"revoked_at" `
}

func NewSession(userID UserID, tokenHash string) *Session {
	return &Session{
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

type SessionID int64

func (id SessionID) Int64() int64 {
	return int64(id)
}

func (id SessionID) String() string {
	return strconv.FormatInt(int64(id), 10)
}
