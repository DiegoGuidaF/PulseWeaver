package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

type repository interface {
	CountUsers(ctx context.Context) (int, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID UserID) (*User, error)
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error)
	RevokeSessionByID(ctx context.Context, id SessionID) error
	RunInTx(ctx context.Context, fn func(repository) error) error
}
type Service struct {
	repo repository
}

func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Login(ctx context.Context, username string, password string) (string, *User, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("authenticating user")

	var rawToken string
	var user *User

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		var err error
		user, err = tx.GetUserByUsername(ctx, username)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				logger.Warn("user not found")
				return err
			}
			logger.Error("database error fetching user", slog.Any(AttrKeyError, err))
			return err
		}

		if !checkPassword(user.PasswordHash, password) {
			logger.Warn("invalid password")
			return ErrInvalidCredentials
		}

		var tokenHash string
		rawToken, tokenHash, err = generateToken()
		if err != nil {
			logger.Error("failed to generate token", slog.Any(AttrKeyError, err))
			return err
		}

		session := NewSession(user.ID, tokenHash)
		_, err = tx.CreateSession(ctx, session)
		if err != nil {
			logger.Error("database error creating session", slog.Any(AttrKeyError, err))
			return err
		}

		logger.Info("user logged in", slog.Int64(AttrKeyUserID, user.ID.Int64()))
		return nil
	})
	if err != nil {
		return "", nil, err
	}

	return rawToken, user, nil
}

func (s *Service) GetUserFromPrincipal(ctx context.Context, principal *Principal) (*User, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("fetching user from principal")

	user, err := s.repo.GetUserByID(ctx, principal.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			logger.Warn("user not found")
			return nil, err
		}
		logger.Error("database error fetching user", slog.Any(AttrKeyError, err))
		return nil, err
	}
	return user, nil
}

func (s *Service) RevokeSession(ctx context.Context, sessionID SessionID) error {
	logger := logging.FromCtx(ctx)
	logger.Debug("revoking session")

	err := s.repo.RevokeSessionByID(ctx, sessionID)
	if err != nil {
		logger.Error("database error revoking session", slog.Any(AttrKeyError, err))
		return err
	}
	logger.Info("session revoked", slog.Int64(AttrKeySessionID, sessionID.Int64()))
	return nil
}

func (s *Service) CreateUserByAdmin(ctx context.Context, username string, displayName string, email *string, password string, principal *Principal) (*User, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("creating user by admin")

	if !principal.isAdmin() {
		logger.Warn("admin credentials required")
		return nil, ErrAdminCredentialsRequired
	}

	return s.createUser(s.repo, ctx, username, displayName, email, password, &principal.UserID, UserRole)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*Principal, error) {
	logger := logging.FromCtx(ctx)
	logger.Debug("authenticating token")

	tokenHash := hashRawToken(rawToken)
	sessionWithUser, err := s.repo.GetSessionWithRoleByTokenHash(ctx, tokenHash)
	if err != nil {
		logger.Warn("session not found or invalid")
		return nil, err
	}

	principal := PrincipalFromSession(sessionWithUser)
	return principal, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, conf config.ConfServer) error {
	logger := logging.FromCtx(ctx)
	logger.Debug("bootstrapping admin")

	username := "admin"
	displayName := "Admin"
	password := conf.AdminPassword

	err := s.repo.RunInTx(ctx, func(tx repository) error {
		count, err := tx.CountUsers(ctx)
		if err != nil {
			logger.Error("database error counting users", slog.Any(AttrKeyError, err))
			return err
		}
		if count > 0 {
			return nil
		}

		user, err := s.createUser(tx, ctx, username, displayName, nil, password, nil, AdminRole)
		if err != nil {
			return fmt.Errorf("failed to bootstrap admin: %w", err)
		}

		logger.Info("bootstrap admin created", slog.String(AttrKeyUsername, user.Username))
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) createUser(tx repository, ctx context.Context, username string, displayName string, email *string, password string, createdBy *UserID, role Role) (*User, error) {
	logger := logging.FromCtx(ctx)

	newUser, err := NewUser(
		username,
		displayName,
		email,
		password,
		role,
		createdBy,
	)
	if err != nil {
		logger.Warn("invalid user input", slog.Any(AttrKeyError, err))
		return nil, err
	}

	user, err := tx.CreateUser(ctx, newUser)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrEmailTaken) {
			logger.Warn("username or email already taken")
			return nil, err
		}
		logger.Error("database error creating user", slog.Any(AttrKeyError, err))
		return nil, err
	}

	logger.Info("user created", slog.Int64(AttrKeyUserID, user.ID.Int64()), slog.String(AttrKeyUsername, user.Username))
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
