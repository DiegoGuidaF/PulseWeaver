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
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, userId UserID, tokenHash string) (*Session, error)
	CreateUser(ctx context.Context, name string, email string, passwordHash []byte, createdById *UserID, role Role) (*User, error)
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

func (s *Service) Login(ctx context.Context, email, password string) (string, *User, error) {
	// Ensure user exists and get info
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}

	// Check password
	if !s.checkPassword(user.PasswordHash, password) {
		return "", nil, ErrInvalidCredentials
	}

	// Create session token
	rawToken, tokenHash, err := s.generateToken()
	if err != nil {
		return "", nil, err
	}

	// Create session
	_, err = s.repo.CreateSession(ctx, user.ID, tokenHash)
	if err != nil {
		return "", nil, err
	}

	// 5. Return Raw Token (for Cookie) and User (for JSON response)
	return rawToken, user, nil
}

func (s *Service) RevokeSession(ctx context.Context, sessionId SessionID) error {
	return s.repo.RevokeSessionById(ctx, sessionId)
}

func (s *Service) CreateUserByAdmin(ctx context.Context, name string, email string, password string, principal *Principal) (*User, error) {
	if !principal.isAdmin() {
		return nil, ErrAdminCredentialsRequired
	}

	return s.createUser(ctx, name, email, password, &principal.UserID, UserRole)
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
	email := "admin@example.com"
	password := conf.AdminPassword

	if password == "" {
		var err error
		password, err = generateSecurePassword(16)
		if err != nil {
			return fmt.Errorf("generate admin password: %w", err)
		}
		generated = true
	}

	user, err := s.createUser(ctx, "Admin", "adminemail@example.com", password, nil, AdminRole)
	if err != nil {
		return fmt.Errorf("failed to bootstrap admin: %w", err)
	}

	if generated {
		s.logger.Warn("🚨 GENERATED ADMIN PASSWORD 🚨 - Store this securely and change it immediately",
			"name", user.Name,
			"email", email,
			"password", password,
		)
	} else {
		s.logger.Info("bootstrap admin created from config", "email", email, "user_id", user.ID)
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

func (s *Service) hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func (s *Service) checkPassword(hash []byte, password string) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

func (s *Service) createUser(ctx context.Context, name string, email string, password string, createdBy *UserID, role Role) (*User, error) {
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, name, email, passwordHash, createdBy, role)
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
