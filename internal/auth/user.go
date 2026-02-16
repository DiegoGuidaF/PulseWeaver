package auth

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const AdminRole Role = "admin"
const UserRole Role = "user"
const DeviceRole Role = "device"

var (
	usernameRegex = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

type Role string

type User struct {
	ID           UserID    `db:"id" `
	Username     string    `db:"username" `
	DisplayName  string    `db:"display_name" `
	Email        *string   `db:"email" `
	PasswordHash []byte    `db:"password_hash" `
	Role         Role      `db:"role" `
	CreatedBy    *UserID   `db:"created_by" `
	CreatedAt    time.Time `db:"created_at" `
}

func NewUser(
	username string,
	displayName string,
	email *string,
	password string,
	role Role,
	createdById *UserID,
) (*User, error) {
	err := validatePassword(password)
	if err != nil {
		return nil, err
	}

	validUsername, err := validateUsername(username)
	if err != nil {
		return nil, err
	}

	validDisplayName, err := validateDisplayName(displayName)
	if err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	return &User{
		Username:     validUsername,
		DisplayName:  validDisplayName,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedBy:    createdById,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

type UserID int64

func (id UserID) Int64() int64 {
	return int64(id)
}

func (id UserID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

func (r Role) String() string {
	return string(r)
}

func validateDisplayName(input string) (string, error) {
	clean := strings.TrimSpace(input)
	minDisplayNameLength := 1
	maxDisplayNameLength := 50

	if len(clean) < minDisplayNameLength {
		return "", fmt.Errorf("%w: too short (min %d chars)", ErrInvalidDisplayName, minDisplayNameLength)
	}

	if len(clean) > maxDisplayNameLength {
		return "", fmt.Errorf("%w: too long (max %d chars)", ErrInvalidDisplayName, maxDisplayNameLength)
	}

	return clean, nil
}

func validateUsername(username string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(username))

	minUsernameLength := 3
	maxUsernameLength := 32

	if len(normalized) < minUsernameLength {
		return "", fmt.Errorf("%w: too short (min %d chars)", ErrInvalidUsername, minUsernameLength)

	}
	if len(normalized) > maxUsernameLength {
		return "", fmt.Errorf("%w: too long (max %d chars)", ErrInvalidUsername, maxUsernameLength)
	}

	if !usernameRegex.MatchString(normalized) {
		return "", fmt.Errorf("%w: invalid characters. Only alphanumeric,- and _ are accepted", ErrInvalidUsername)
	}

	return normalized, nil
}

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func validatePassword(password string) error {
	minPasswordLength := 8
	maxPasswordLength := 32
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: too short (min %d chars)", ErrInvalidPassword, minPasswordLength)
	}

	if len(password) > maxPasswordLength {
		return fmt.Errorf("%w: too long (max %d chars)", ErrInvalidPassword, maxPasswordLength)
	}

	return nil
}
