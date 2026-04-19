package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

const (
	BootstrapAdminUsername    = "admin"
	BootstrapAdminDisplayName = "Admin"
	BootstrapAdminEmail       = "admin@pulseweaver.invalid"
)

type repository interface {
	CountUsers(ctx context.Context) (int, error)
	CountAdminUsers(ctx context.Context) (int, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID UserID) (*User, error)
	UpdateUser(ctx context.Context, user *User) (*User, error)
	UpdatePasswordHash(ctx context.Context, userID UserID, newHash []byte, mustChangePassword bool) error
	NullifyPasswordHash(ctx context.Context, userID UserID) error
	SoftDeleteUser(ctx context.Context, userID UserID) error
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error)
	RevokeSessionByID(ctx context.Context, id SessionID) error
	RevokeAllUserSessions(ctx context.Context, userID UserID) error
	RevokeAllUserSessionsExcept(ctx context.Context, userID UserID, exceptSessionID SessionID) error
}

type transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type Service struct {
	repo   repository
	tx     transactor
	logger *slog.Logger
}

func NewService(repo repository, tx transactor, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		tx:     tx,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "auth")),
	}
}

func (s *Service) Login(ctx context.Context, username string, password string) (string, *User, error) {
	var rawToken string
	var user *User

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		user, err = s.repo.GetUserByUsername(ctx, username)
		if err != nil {
			return err
		}

		if user.PasswordHash == nil || !checkPassword(user.PasswordHash, password) {
			return ErrInvalidCredentials
		}

		var tokenHash string
		rawToken, tokenHash, err = generateToken()
		if err != nil {
			return err
		}

		_, err = s.repo.CreateSession(ctx, new(NewSession(user.ID, tokenHash)))
		if err != nil {
			return err
		}

		s.logger.InfoContext(ctx, "user logged in", slog.Int64(AttrKeyUserID, user.ID.Int64()))
		return nil
	})
	if err != nil {
		return "", nil, err
	}

	return rawToken, user, nil
}

func (s *Service) GetUserFromPrincipal(ctx context.Context, principal *Principal) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, principal.UserID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) RevokeSession(ctx context.Context, sessionID SessionID) error {
	err := s.repo.RevokeSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "session revoked", slog.Int64(AttrKeySessionID, sessionID.Int64()))
	return nil
}

func (s *Service) CreateUser(ctx context.Context, username string, displayName string, email string, principal *Principal) (*User, error) {
	var newUser User
	var err error

	newUser, err = NewUserAccount(username, displayName, email, &principal.UserID)
	if err != nil {
		return nil, err
	}

	return s.createUser(ctx, &newUser)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*Principal, error) {
	tokenHash := hashRawToken(rawToken)
	sessionWithUser, err := s.repo.GetSessionWithRoleByTokenHash(ctx, tokenHash)
	if err != nil {
		s.logger.DebugContext(ctx, "session not found or invalid")
		return nil, err
	}

	principal := PrincipalFromSession(sessionWithUser)
	return principal, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, conf config.ConfServer) error {
	password := conf.AdminPassword

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		count, err := s.repo.CountAdminUsers(ctx)
		if err != nil {
			s.logger.ErrorContext(ctx, "database error counting admins", slog.Any(AttrKeyError, err))
			return err
		}
		if count > 0 {
			return nil
		}

		newUser, err := NewAdminUser(BootstrapAdminUsername, BootstrapAdminDisplayName, BootstrapAdminEmail, password, nil, false)
		if err != nil {
			return err
		}
		user, err := s.createUser(ctx, &newUser)
		if err != nil {
			return fmt.Errorf("failed to bootstrap admin: %w", err)
		}

		s.logger.InfoContext(ctx, "bootstrap admin created", slog.String(AttrKeyUsername, user.Username))
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.GetAllUsers(ctx)
}

type ProfileUpdates struct {
	DisplayName *string
	Username    *string
	Email       *string
}

func (s *Service) UpdateOwnProfile(ctx context.Context, userID UserID, updates ProfileUpdates) (*User, error) {
	var updatedUser *User

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}

		err = user.Update(updates)
		if err != nil {
			return err
		}

		updatedUser, err = s.repo.UpdateUser(ctx, user)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "user profile updated", slog.Int64(AttrKeyUserID, userID.Int64()))
	return updatedUser, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID UserID, sessionID SessionID, currentPassword string, newPassword string) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {

		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}

		if user.PasswordHash == nil || !checkPassword(user.PasswordHash, currentPassword) {
			return ErrInvalidCredentials
		}

		err = ValidatePassword(newPassword)
		if err != nil {
			return err
		}
		newPasswordHash, err := hashPassword(newPassword)
		if err != nil {
			return fmt.Errorf("hashing failed: %w", err)
		}

		err = s.repo.UpdatePasswordHash(ctx, userID, newPasswordHash, false)
		if err != nil {
			return err
		}

		err = s.repo.RevokeAllUserSessionsExcept(ctx, userID, sessionID)
		if err != nil {
			return err
		}

		s.logger.InfoContext(ctx, "password changed", slog.Int64(AttrKeyUserID, userID.Int64()))
		return nil
	})
}

func (s *Service) PromoteUser(ctx context.Context, principal *Principal, targetID UserID, newPassword string) (*User, error) {
	var updatedUser *User
	if principal.UserID == targetID {
		return nil, ErrSelfRoleChangeForbidden
	}

	if err := ValidatePassword(newPassword); err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		target, err := s.repo.GetUserByID(ctx, targetID)
		if err != nil {
			return err
		}

		if target.Role == AdminRole {
			return ErrPromoteAlreadyAdmin
		}

		target.Role = AdminRole
		updatedUser, err = s.repo.UpdateUser(ctx, target)
		if err != nil {
			return err
		}

		return s.repo.UpdatePasswordHash(ctx, targetID, passwordHash, true)
	})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "user promoted to admin", slog.Int64(AttrKeyUserID, targetID.Int64()))
	return updatedUser, nil
}

func (s *Service) DemoteUser(ctx context.Context, principal *Principal, targetID UserID) (*User, error) {
	var updatedUser *User
	if principal.UserID == targetID {
		return nil, ErrSelfRoleChangeForbidden
	}

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		target, err := s.repo.GetUserByID(ctx, targetID)
		if err != nil {
			return err
		}

		target.Role = UserRole
		updatedUser, err = s.repo.UpdateUser(ctx, target)
		if err != nil {
			return err
		}

		if err = s.repo.NullifyPasswordHash(ctx, targetID); err != nil {
			return err
		}

		return s.repo.RevokeAllUserSessions(ctx, targetID)
	})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "admin demoted to user", slog.Int64(AttrKeyUserID, targetID.Int64()))
	return updatedUser, nil
}

func (s *Service) DeleteUser(ctx context.Context, principal *Principal, targetID UserID) error {
	if principal.UserID == targetID {
		return ErrSelfDeleteForbidden
	}

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Ensure user to delete exists and is valid
		_, err := s.repo.GetUserByID(ctx, targetID)
		if err != nil {
			return err
		}

		err = s.repo.RevokeAllUserSessions(ctx, targetID)
		if err != nil {
			return err
		}

		err = s.repo.SoftDeleteUser(ctx, targetID)
		if err != nil {
			return err
		}

		s.logger.InfoContext(ctx, "user deleted", slog.Int64(AttrKeyUserID, targetID.Int64()))
		return nil
	})

	return err
}

func (s *Service) createUser(ctx context.Context, newUser *User) (*User, error) {
	user, err := s.repo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "user created", slog.Int64(AttrKeyUserID, user.ID.Int64()), slog.String(AttrKeyUsername, user.Username))
	return user, nil
}

func hashRawToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func checkPassword(hash []byte, password string) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

// generateToken Returns the rawToken (send to user), tokenHash (store in DB), error
func generateToken() (string, string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", "", err
	}
	// URL-safe base64, no padding
	rawToken := base64.RawURLEncoding.EncodeToString(b)

	// Hash immediately for storage
	tokenHash := hashRawToken(rawToken)

	return rawToken, tokenHash, nil
}
