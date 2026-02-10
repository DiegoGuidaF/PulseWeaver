package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type repository struct {
	db     DBInterface
	rootDB *sqlx.DB
}

type DBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{
		rootDB: db,
		db:     db,
	}
}

func (r *repository) CreateUser(ctx context.Context, name string, email string, passwordHash []byte) (*User, error) {
	user := User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	query := `
        INSERT INTO users (name, email, password_hash, created_at)
        VALUES (?, ?, ?, ?) RETURNING id, name, email, password_hash, created_at
    `

	err := r.db.GetContext(ctx, &user, query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.CreatedAt,
	)
	if err != nil {
		//TODO: Can return here emailAlreadyExists error if UNIQUE constraint fails
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	query := `SELECT id, name, email, password_hash, created_at FROM users WHERE email = ?`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		//TODO: Return more specific error, handler would not propagate that one to user but a generic one instead
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *repository) CreateSession(ctx context.Context, userId UserID, tokenHash string) (*Session, error) {
	tokenDuration := time.Hour * 24 * 7
	session := Session{
		UserId:     userId,
		TokenHash:  tokenHash,
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(tokenDuration),
		LastUsedAt: nil,
		RevokedAt:  nil,
	}

	query := `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?) RETURNING id, user_id, token_hash, created_at, expires_at, last_used_at, revoked_at
	`

	err := r.db.GetContext(ctx, &session, query,
		session.UserId,
		session.TokenHash,
		session.CreatedAt,
		session.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, nil
}

func (r *repository) GetSessionByRawToken(ctx context.Context, tokenHash string) (*Session, error) {
	var session Session

	query := `SELECT id, user_id, token_hash, created_at, expires_at, last_used_at, revoked_at FROM sessions
				WHERE  token_hash = ?`

	err := r.db.GetContext(ctx, &session, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}
