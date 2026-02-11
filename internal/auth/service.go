package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	CountUsers(ctx context.Context) (int, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error)
	RevokeSessionById(ctx context.Context, id SessionID) error
}
type Service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Login(ctx context.Context, username string, password string) (string, *User, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", nil, err
	}

	if !s.checkPassword(user.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}

	rawToken, tokenHash, err := s.generateToken()
	if err != nil {
		return "", nil, err
	}

	session := NewSession(user.ID, tokenHash)

	_, err = s.repo.CreateSession(ctx, session)
	if err != nil {
		return "", nil, err
	}

	return rawToken, user, nil
}

func (s *Service) RevokeSession(ctx context.Context, sessionId SessionID) error {
	return s.repo.RevokeSessionById(ctx, sessionId)
}

func (s *Service) CreateUserByAdmin(ctx context.Context, username string, displayName string, email *string, password string, principal *Principal) (*User, error) {

	if !principal.isAdmin() {
		return nil, ErrAdminCredentialsRequired
	}

	return s.createUser(ctx, username, displayName, email, password, &principal.UserID, UserRole)
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
	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	generated := false
	username := "admin"
	displayName := "Admin"
	password := conf.AdminPassword

	if password == "" {
		var err error
		password, err = generateSecurePassword(16)
		if err != nil {
			return fmt.Errorf("generate admin password: %w", err)
		}
		generated = true
	}

	user, err := s.createUser(ctx, username, displayName, nil, password, nil, AdminRole)
	if err != nil {
		return fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	if generated {
		s.logger.Warn("🚨 GENERATED ADMIN PASSWORD 🚨 - Store this securely and change it immediately",
			"username", user.Username,
			"password", password,
		)
	} else {
		s.logger.Info("bootstrap admin created using password from ADMIN_PASSWORD env variable", "username", user.Username)
	}
	return nil
}

// generateToken Returns the rawToken (send to user), tokenHash (store in DB), error
func (s *Service) generateToken() (string, string, error) {
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

func hashRawToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func (s *Service) checkPassword(hash []byte, password string) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

func (s *Service) createUser(ctx context.Context, username string, displayName string, email *string, password string, createdBy *UserID, role Role) (*User, error) {
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

	user, err := s.repo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

	return user, err
}

func generateSecurePassword(length int) (string, error) {
	// Round up to multiple of 3 bytes for base64
	byteLen := (length*4+2)/3 + 1
	b := make([]byte, byteLen)

	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	// base64url (no padding) is safe for passwords, URLs, and filenames
	return base64.RawURLEncoding.EncodeToString(b)[:length], nil
}
