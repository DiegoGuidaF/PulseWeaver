package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db     dBInterface
	rootDB *sqlx.DB
}

type dBInterface interface {
	sqlx.ExtContext
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		rootDB: db,
		db:     db,
	}
}

func (r *Repository) CreateUser(ctx context.Context, user *User) (*User, error) {
	query := `
        INSERT INTO users (username, display_name, email, password_hash, role, created_by, created_at)
        VALUES (?, ?, ?, ?, ?,?, ?) RETURNING *
    `

	err := r.db.GetContext(ctx, user, query,
		user.Username,
		user.DisplayName,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedBy,
		user.CreatedAt,
	)
	if err != nil {
		if conflictErr, ok := mapUserCreationUniqueConstraintError(err); ok {
			return nil, conflictErr
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user := &User{}

	query := `SELECT * FROM users WHERE username = ?`

	err := r.db.GetContext(ctx, user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
func (r *Repository) GetUserByID(ctx context.Context, userID UserID) (*User, error) {
	user := &User{}

	query := `SELECT * FROM users WHERE id = ?`

	err := r.db.GetContext(ctx, user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var userCount int

	query := `SELECT count(*) FROM users`

	err := r.db.GetContext(ctx, &userCount, query)
	if err != nil {
		return -1, fmt.Errorf("failed to get user count: %w", err)
	}

	return userCount, nil
}

func (r *Repository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	query := `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?) RETURNING *
	`

	err := r.db.GetContext(ctx, session, query,
		session.UserID,
		session.TokenHash,
		session.CreatedAt,
		session.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSessionWithRoleByTokenHash Finds and retrieves valid session(non-expired or revoked) given a tokenHash.
// Also returns the user_role
func (r *Repository) GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
	session := &SessionWithUser{}

	query := `SELECT s.*, u.role as user_role FROM sessions s
          	  JOIN users u ON s.user_id = u.id
			  WHERE  token_hash = ?
          		AND revoked_at IS NULL
          		AND expires_at > CURRENT_TIMESTAMP
	`

	err := r.db.GetContext(ctx, session, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

func (r *Repository) RevokeSessionByID(ctx context.Context, id SessionID) error {
	query := `UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	return nil
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *Repository) RunInTx(ctx context.Context, fn func(repository) error) error {
	if r.rootDB == nil {
		// We are already in a transaction. Do not nest it.
		return fn(r)
	}

	// Start the transaction
	tx, err := r.rootDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		//nolint:staticcheck // Empty branch is intentional - ErrTxDone is expected after commit
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			// Rollback error is only significant if transaction wasn't already committed/rolled back
		}
	}()

	// Copy of the repository without rootDB so we can't do nested transactions
	txRepo := &Repository{
		rootDB: nil,
		db:     tx,
	}

	// Run function
	if err := fn(txRepo); err != nil {
		return err
	}

	return tx.Commit()
}

func mapUserCreationUniqueConstraintError(err error) (error, bool) {
	// Check if error is a unique constraint violation
	// modernc.org/sqlite returns errors with "UNIQUE constraint failed" in the message
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint failed") {
		switch {
		case strings.Contains(message, "users.username"):
			return ErrUsernameTaken, true
		case strings.Contains(message, "users.email"):
			return ErrEmailTaken, true
		default:
			return ErrUsernameTaken, true
		}
	}
	return nil, false
}
