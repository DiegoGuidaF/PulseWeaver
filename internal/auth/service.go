package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, userId UserID, tokenHash string) (*Session, error)
	CreateUser(ctx context.Context, name string, email string, passwordHash []byte) (*User, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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
	rawToken, tokenHash, err := s.GenerateToken()
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

func (s *Service) SignUp(ctx context.Context, name string, email string, password string) (string, *User, error) {
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return "", nil, fmt.Errorf("hashing failed: %w", err)
	}

	user, err := s.repo.CreateUser(ctx, name, email, passwordHash)
	if err != nil {
		return "", nil, err
	}

	// Auto-login after signup
	return s.Login(ctx, user.Email, password)
}

// GenerateToken Returns the rawToken (send to user), tokenHash (store in DB), error
func (s *Service) GenerateToken() (string, string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", "", err
	}
	// URL-safe base64, no padding
	rawToken := base64.RawURLEncoding.EncodeToString(b)

	// Hash immediately for storage
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	return rawToken, tokenHash, nil
}

func (s *Service) hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func (s *Service) checkPassword(hash []byte, password string) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}
