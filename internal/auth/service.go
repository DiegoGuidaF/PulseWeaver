package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	CountUsers(ctx context.Context) (int, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userId UserID) (*User, error)
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error)
	RevokeSessionById(ctx context.Context, id SessionID) error
	RunInTx(ctx context.Context, fn func(UserRepository) error) error
}
type Service struct {
	repo   UserRepository
	logger *slog.Logger
}

func NewService(repo UserRepository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Login(ctx context.Context, username string, password string) (string, *User, error) {
	var rawToken string
	var user *User

	err := s.repo.RunInTx(ctx, func(tx UserRepository) error {
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

		return nil
	})
	if err != nil {
		return "", nil, err
	}

	return rawToken, user, nil
}

func (s *Service) GetUserFromPrincipal(ctx context.Context, principal *Principal) (*User, error) {
	return s.repo.GetUserByID(ctx, principal.UserID)
}

func (s *Service) RevokeSession(ctx context.Context, sessionId SessionID) error {
	return s.repo.RevokeSessionById(ctx, sessionId)
}

func (s *Service) CreateUserByAdmin(ctx context.Context, username string, displayName string, email *string, password string, principal *Principal) (*User, error) {
	if !principal.isAdmin() {
		return nil, ErrAdminCredentialsRequired
	}

	return s.createUser(s.repo, ctx, username, displayName, email, password, &principal.UserID, UserRole)
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*Principal, error) {
	tokenHash := hashRawToken(rawToken)
	sessionWithUser, err := s.repo.GetSessionWithRoleByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	principal := PrincipalFromSession(sessionWithUser)

	return principal, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, conf config.ConfServer) error {
	username := "admin"
	displayName := "Admin"
	password := conf.AdminPassword

	err := s.repo.RunInTx(ctx, func(tx UserRepository) error {
		count, err := tx.CountUsers(ctx)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		user, err := s.createUser(tx, ctx, username, displayName, nil, password, nil, AdminRole)
		if err != nil {
			return fmt.Errorf("failed to bootstrap admin: %w", err)
		}

		s.logger.Info("bootstrap admin created", "username", user.Username)
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) createUser(tx UserRepository, ctx context.Context, username string, displayName string, email *string, password string, createdBy *UserID, role Role) (*User, error) {
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

	user, err := tx.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

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
