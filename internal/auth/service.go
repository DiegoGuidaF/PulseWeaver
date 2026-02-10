package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

type AuthRepository struct {
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

// // Returns: rawToken (send to user), tokenHash (store in DB), error
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

//
//func (s *Service) AuthenticateSession(ctx context.Context, rawToken string) (Principal, error) {
//	// 1. Hash the incoming token
//	hash := sha256.Sum256([]byte(rawToken))
//	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])
//
//	// 2. Lookup by HASH (Constant time lookup in DB index)
//	// We don't compare raw tokens in Go memory; we query the DB for the hash.
//	session, err := s.repo.GetSessionByHash(ctx, tokenHash)
//	if err != nil {
//		return Principal{}, errors.New("invalid session")
//	}
//
//	// 3. Check Expiry
//	if session.ExpiresAt.Before(time.Now()) {
//		return Principal{}, errors.New("session expired")
//	}
//
//	return Principal{UserID: session.UserID}, nil
//}
