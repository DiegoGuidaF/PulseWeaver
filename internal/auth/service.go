package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/config"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

type repository interface {
	CountUsers(ctx context.Context) (int, error)
	CountAdminUsers(ctx context.Context) (int, error)
	GetAllUsers(ctx context.Context) ([]User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID UserID) (*User, error)
	UpdateUser(ctx context.Context, user *User) (*User, error)
	UpdatePasswordHash(ctx context.Context, userID UserID, newHash []byte) error
	SoftDeleteUser(ctx context.Context, userID UserID) error
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error)
	RevokeSessionByID(ctx context.Context, id SessionID) error
	RevokeAllUserSessions(ctx context.Context, userID UserID) error
	RevokeAllUserSessionsExcept(ctx context.Context, userID UserID, exceptSessionID SessionID) error
	RunInTx(ctx context.Context, fn func(repository) error) error
}
type Service struct {
	repo   repository
	logger *slog.Logger
}

func NewService(repo repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "auth")),
	}
}

func (s *Service) Login(ctx context.Context, username string, password string) (string, *User, error) {
	var rawToken string
	var user *User

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		var err error
		user, err = tx.GetUserByUsername(ctx, username)
		if err != nil {
			return err
		}

		if !checkPassword(user.PasswordHash, password) {
			return ErrInvalidCredentials
		}

		var tokenHash string
		rawToken, tokenHash, err = generateToken()
		if err != nil {
			return err
		}

		session := NewSession(user.ID, tokenHash)
		_, err = tx.CreateSession(ctx, session)
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

func (s *Service) CreateUserByAdmin(ctx context.Context, username string, displayName string, email string, password string, principal *Principal) (*User, error) {
	if !principal.isAdmin() {
		return nil, ErrAdminCredentialsRequired
	}

	return s.createUser(s.repo, ctx, username, displayName, email, password, &principal.UserID, UserRole, true)
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
	username := "admin"
	displayName := "Admin"
	password := conf.AdminPassword

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		count, err := tx.CountAdminUsers(ctx)
		if err != nil {
			s.logger.ErrorContext(ctx, "database error counting admins", slog.Any(AttrKeyError, err))
			return err
		}
		if count > 0 {
			return nil
		}

		user, err := s.createUser(tx, ctx, username, displayName, "admin@wallydic.invalid", password, nil, AdminRole, false)
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
	if updates.DisplayName == nil && updates.Username == nil && updates.Email == nil {
		return nil, ErrNoUpdateFields
	}

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if updates.DisplayName != nil {
		validDisplayName, err := validateDisplayName(*updates.DisplayName)
		if err != nil {
			return nil, err
		}
		user.DisplayName = validDisplayName
	}
	if updates.Username != nil {
		validUsername, err := validateUsername(*updates.Username)
		if err != nil {
			return nil, err
		}
		user.Username = validUsername
	}
	if updates.Email != nil {
		user.Email = *updates.Email
	}

	updatedUser, err := s.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "user profile updated", slog.Int64(AttrKeyUserID, userID.Int64()))
	return updatedUser, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID UserID, currentSessionID SessionID, currentPassword string, newPassword string) error {
	return s.repo.RunInTx(ctx, func(tx repository) error {
		user, err := tx.GetUserByID(ctx, userID)
		if err != nil {
			return err
		}

		if !checkPassword(user.PasswordHash, currentPassword) {
			return ErrInvalidCredentials
		}

		err = validatePassword(newPassword)
		if err != nil {
			return err
		}
		newPasswordHash, err := hashPassword(newPassword)
		if err != nil {
			return fmt.Errorf("hashing failed: %w", err)
		}

		err = tx.UpdatePasswordHash(ctx, userID, newPasswordHash)
		if err != nil {
			return err
		}

		err = tx.RevokeAllUserSessionsExcept(ctx, userID, currentSessionID)
		if err != nil {
			return err
		}

		s.logger.InfoContext(ctx, "password changed", slog.Int64(AttrKeyUserID, userID.Int64()))
		return nil
	})
}

type AdminUserUpdates struct {
	Role *Role
}

func (s *Service) AdminUpdateUser(ctx context.Context, requesterID UserID, targetID UserID, updates AdminUserUpdates) (*User, error) {
	var updatedUser *User

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		target, err := tx.GetUserByID(ctx, targetID)
		if err != nil {
			return err
		}

		if updates.Role == nil {
			return ErrNoUpdateFields
		}

		if requesterID == targetID {
			return ErrSelfRoleChangeForbidden
		}
		if *updates.Role != UserRole && *updates.Role != AdminRole {
			return ErrInvalidRole
		}

		if target.Role == AdminRole && *updates.Role == UserRole {
			adminCount, err := tx.CountAdminUsers(ctx)
			if err != nil {
				return err
			}
			if adminCount <= 1 {
				return ErrLastAdminChangeForbidden
			}
		}

		target.Role = *updates.Role
		updatedUser, err = tx.UpdateUser(ctx, target)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "admin updated user", slog.Int64(AttrKeyUserID, targetID.Int64()))
	return updatedUser, nil
}

func (s *Service) DeleteUser(ctx context.Context, requesterID UserID, targetID UserID) error {
	return s.repo.RunInTx(ctx, func(tx repository) error {
		if requesterID == targetID {
			return ErrSelfDeleteForbidden
		}

		target, err := tx.GetUserByID(ctx, targetID)
		if err != nil {
			return err
		}

		if target.Role == AdminRole {
			adminCount, err := tx.CountAdminUsers(ctx)
			if err != nil {
				return err
			}
			if adminCount <= 1 {
				return ErrLastAdminChangeForbidden
			}
		}

		err = tx.RevokeAllUserSessions(ctx, targetID)
		if err != nil {
			return err
		}

		err = tx.SoftDeleteUser(ctx, targetID)
		if err != nil {
			return err
		}

		s.logger.InfoContext(ctx, "user deleted", slog.Int64(AttrKeyUserID, targetID.Int64()))
		return nil
	})
}

func (s *Service) createUser(tx repository, ctx context.Context, username string, displayName string, email string, password string, createdBy *UserID, role Role, mustChangePassword bool) (*User, error) {
	newUser, err := NewUser(
		username,
		displayName,
		email,
		password,
		role,
		createdBy,
	)
	if err != nil {
		return nil, err
	}
	newUser.MustChangePassword = mustChangePassword

	user, err := tx.CreateUser(ctx, newUser)
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
