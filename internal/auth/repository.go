package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateUser(ctx context.Context, user *User) (*User, error) {
	query := `
        INSERT INTO users (username, display_name, email, password_hash, role, must_change_password, created_by, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING *`

	created := new(User)
	err := r.db.GetContext(ctx, created, query,
		user.Username,
		user.DisplayName,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.MustChangePassword,
		user.CreatedBy,
		user.CreatedAt,
	)
	if err != nil {
		if conflictErr, ok := mapUserCreationUniqueConstraintError(err); ok {
			return nil, conflictErr
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return created, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user := new(User)

	query := `SELECT * FROM users WHERE username = ? AND deleted_at IS NULL`

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
	user := new(User)

	query := `SELECT * FROM users WHERE id = ? AND deleted_at IS NULL`

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

	query := `SELECT count(*) FROM users WHERE deleted_at IS NULL`

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

	created := new(Session)
	err := r.db.GetContext(ctx, created, query,
		session.UserID,
		session.TokenHash,
		session.CreatedAt,
		session.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return created, nil
}

func (r *Repository) CountAdminUsers(ctx context.Context) (int, error) {
	var adminCount int

	query := `SELECT count(*) FROM users WHERE role = ? AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &adminCount, query, AdminRole)
	if err != nil {
		return -1, fmt.Errorf("failed to get admin count: %w", err)
	}

	return adminCount, nil
}

func (r *Repository) GetAllUsers(ctx context.Context) ([]User, error) {
	users := make([]User, 0)

	query := `SELECT * FROM users WHERE deleted_at IS NULL ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	return users, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *User) (*User, error) {
	const query = `
        UPDATE users
        SET username = ?, display_name = ?, email = ?, role = ?, must_change_password = ?
        WHERE id = ? AND deleted_at IS NULL
        RETURNING *`

	updated := new(User)
	err := r.db.GetContext(ctx, updated, query,
		user.Username, user.DisplayName, user.Email, user.Role, user.MustChangePassword,
		user.ID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		if conflictErr, ok := mapUserCreationUniqueConstraintError(err); ok {
			return nil, conflictErr
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return updated, nil
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, userID UserID, newHash []byte) error {
	const query = `UPDATE users SET password_hash = ?, must_change_password = false WHERE id = ? AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, newHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password hash: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) SoftDeleteUser(ctx context.Context, userID UserID) error {
	query := `UPDATE users SET deleted_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for soft delete: %w", err)
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *Repository) RevokeAllUserSessions(ctx context.Context, userID UserID) error {
	query := `
		UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
			AND revoked_at IS NULL
			AND expires_at > CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user sessions: %w", err)
	}

	return nil
}

func (r *Repository) RevokeAllUserSessionsExcept(ctx context.Context, userID UserID, exceptSessionID SessionID) error {
	query := `
		UPDATE sessions
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE user_id = ?
			AND id != ?
			AND revoked_at IS NULL
			AND expires_at > CURRENT_TIMESTAMP
	`
	_, err := r.db.ExecContext(ctx, query, userID, exceptSessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions except current: %w", err)
	}

	return nil
}

// GetSessionWithRoleByTokenHash Finds and retrieves valid session(non-expired or revoked) given a tokenHash.
// Also returns the user_role
func (r *Repository) GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
	session := new(SessionWithUser)

	query := `SELECT s.*, u.role as user_role FROM sessions s
          	  JOIN users u ON s.user_id = u.id
			  WHERE  token_hash = ?
          		AND revoked_at IS NULL
          		AND expires_at > CURRENT_TIMESTAMP
				AND u.deleted_at IS NULL
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

func mapUserCreationUniqueConstraintError(err error) (error, bool) {
	// Check if error is a unique constraint violation
	// modernc.org/sqlite returns errors with "UNIQUE constraint failed" in the message
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique constraint failed") {
		switch {
		case strings.Contains(message, "users.username"), strings.Contains(message, "idx_users_username_active"):
			return ErrUsernameTaken, true
		case strings.Contains(message, "users.email"), strings.Contains(message, "idx_users_email_active"):
			return ErrEmailTaken, true
		default:
			return ErrUsernameTaken, true
		}
	}
	return nil, false
}
