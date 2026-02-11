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

func (r *repository) CreateUser(
	ctx context.Context,
	name string,
	email string,
	passwordHash []byte,
	createdBy *UserID,
	role Role,
) (*User, error) {
	user := User{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now().UTC(),
	}

	query := `
        INSERT INTO users (name, email, password_hash, role, created_by, created_at)
        VALUES (?, ?, ?, ?,?, ?) RETURNING *
    `

	err := r.db.GetContext(ctx, &user, query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedBy,
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

	query := `SELECT * FROM users WHERE email = ?`

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

func (r *repository) CountUsers(ctx context.Context) (int, error) {
	var userCount int

	query := `SELECT count(*) FROM users`

	err := r.db.GetContext(ctx, &userCount, query)
	if err != nil {
		return -1, fmt.Errorf("failed to get user count: %w", err)
	}

	return userCount, nil
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
		VALUES (?, ?, ?, ?) RETURNING *
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

// GetSessionWithRoleByTokenHash Finds and retrieves valid session(non-expired or revoked) given a tokenHash.
// Also returns the user_role
func (r *repository) GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
	var session SessionWithUser

	query := `SELECT s.*, u.role as user_role FROM sessions s
          	  JOIN users u ON s.user_id = u.id
			  WHERE  token_hash = ?
          		AND revoked_at IS NULL
          		AND expires_at > CURRENT_TIMESTAMP
	`

	err := r.db.GetContext(ctx, &session, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (r *repository) RevokeSessionById(ctx context.Context, id SessionID) error {
	query := `UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	return nil
}
