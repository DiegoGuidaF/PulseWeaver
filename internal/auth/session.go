package auth

import (
	"strconv"
	"time"
)

type Session struct {
	ID         SessionID  `db:"id" json:"id"`
	UserId     UserID     `db:"user_id" json:"user_id"`
	TokenHash  string     `db:"token_hash" json:"-"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt  time.Time  `db:"expires_at" json:"expires_at"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
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
