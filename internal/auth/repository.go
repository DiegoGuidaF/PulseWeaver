package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type repository struct {
	db     DBInterface
	rootDB *sqlx.DB
}

type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, userId UserID, tokenHash string) (*Session, error)
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

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User

	query := `SELECT id, name, email, password_hash, created_at FROM users WHERE email = ?`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials // Generic error to prevent user enumeration
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *repository) CreateSession(ctx context.Context, userId UserID, tokenHash string) (*Session, error) {
	var session Session

	query := `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?)
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

//func (r *Repository) GetSessionByHash(ctx context.Context, passwd_hash string) (*User, error) {
//	var user User
//	query := `
//		SELECT id, name, email, password_hash, created_at
//		FROM users
//		WHERE password_hash = ?
//	`
//	err := r.db.GetContext(ctx, &user, query, passwd_hash)
//	if err != nil {
//		if errors.Is(err, sql.ErrNoRows) {
//			return nil, ErrUserNotFound
//		}
//		return nil, fmt.Errorf("failed to get user: %w", err)
//	}
//	return &user, nil
//
//}
