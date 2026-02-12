package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"
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

func NewRepository(db *sqlx.DB) UserRepository {
	return &repository{
		rootDB: db,
		db:     db,
	}
}

func (r *repository) CreateUser(ctx context.Context, user *User) (*User, error) {
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

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
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
func (r *repository) GetUserByID(ctx context.Context, userId UserID) (*User, error) {
	user := &User{}

	query := `SELECT * FROM users WHERE id = ?`

	err := r.db.GetContext(ctx, user, query, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
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

func (r *repository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	query := `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?) RETURNING *
	`

	err := r.db.GetContext(ctx, session, query,
		session.UserId,
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
func (r *repository) GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
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

func (r *repository) RevokeSessionById(ctx context.Context, id SessionID) error {
	query := `UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	return nil
}

// RunInTx runs the callback function inside a transaction.
// If already running in a transaction context, do not create a new one and reuse it
func (r *repository) RunInTx(ctx context.Context, fn func(UserRepository) error) error {
	if r.rootDB == nil {
		// We are already in a transaction. Do not nest it.
		return fn(r)
	}

	// Start the transaction
	tx, err := r.rootDB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Copy of the repository without rootDB so we can't do nested transactions
	txRepo := &repository{
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
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			message := strings.ToLower(sqliteErr.Error())
			switch {
			case strings.Contains(message, "users.username"):
				return ErrUsernameTaken, true
			case strings.Contains(message, "users.email"):
				return ErrEmailTaken, true
			default:
				return ErrUsernameTaken, true
			}
		}
	}
	return nil, false
}
