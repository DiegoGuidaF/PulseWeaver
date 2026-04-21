package auth

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	BootstrapAdminUsername    = "admin"
	BootstrapAdminDisplayName = "Admin"
	BootstrapAdminEmail       = "admin@pulseweaver.invalid"

	SuperAdminRole Role = "superadmin"
	AdminRole      Role = "admin"
	UserRole       Role = "user"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

type Role string

type User struct {
	ID                 UserID     `db:"id"`
	Username           string     `db:"username"`
	DisplayName        string     `db:"display_name"`
	Email              string     `db:"email"`
	PasswordHash       []byte     `db:"password_hash"`
	Role               Role       `db:"role"`
	MustChangePassword bool       `db:"must_change_password"`
	CreatedBy          *UserID    `db:"created_by"`
	CreatedAt          time.Time  `db:"created_at"`
	DeletedAt          *time.Time `db:"deleted_at"`
}

func NewBootstrappedAdmin(password string) (User, error) {
	if err := ValidatePassword(password); err != nil {
		return User{}, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return User{}, fmt.Errorf("hashing failed: %w", err)
	}
	return User{
		Username:           BootstrapAdminUsername,
		DisplayName:        BootstrapAdminDisplayName,
		Email:              BootstrapAdminEmail,
		PasswordHash:       passwordHash,
		Role:               SuperAdminRole,
		MustChangePassword: false,
		CreatedBy:          nil,
		CreatedAt:          time.Now().UTC(),
	}, nil
}

// NewAdminUser creates an admin-role user; password is required and hashed.
// MustChangePassword is set to true so the admin must change the assigned password on first login.
func NewAdminUser(username, displayName, email, password string, createdByID *UserID, mustChangePassword bool) (User, error) {
	if err := ValidatePassword(password); err != nil {
		return User{}, err
	}

	validUsername, err := ValidateUsername(username)
	if err != nil {
		return User{}, err
	}

	validDisplayName, err := ValidateDisplayName(displayName)
	if err != nil {
		return User{}, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return User{}, fmt.Errorf("hashing failed: %w", err)
	}

	return User{
		Username:           validUsername,
		DisplayName:        validDisplayName,
		Email:              email,
		PasswordHash:       passwordHash,
		Role:               AdminRole,
		MustChangePassword: mustChangePassword,
		CreatedBy:          createdByID,
		CreatedAt:          time.Now().UTC(),
	}, nil
}

// NewUserAccount creates a user-role account with no password.
// Non-admin users cannot log in; they exist solely to own devices.
func NewUserAccount(username, displayName, email string, createdByID *UserID) (User, error) {
	validUsername, err := ValidateUsername(username)
	if err != nil {
		return User{}, err
	}

	validDisplayName, err := ValidateDisplayName(displayName)
	if err != nil {
		return User{}, err
	}

	return User{
		Username:     validUsername,
		DisplayName:  validDisplayName,
		Email:        email,
		PasswordHash: nil,
		Role:         UserRole,
		CreatedBy:    createdByID,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func (u *User) Update(up ProfileUpdates) error {
	if up.DisplayName == nil && up.Username == nil && up.Email == nil {
		return ErrNoUpdateFields
	}

	if up.DisplayName != nil {
		validDisplayName, err := ValidateDisplayName(*up.DisplayName)
		if err != nil {
			return err
		}
		u.DisplayName = validDisplayName
	}

	if up.Username != nil {
		validUsername, err := ValidateUsername(*up.Username)
		if err != nil {
			return err
		}
		u.Username = validUsername
	}

	if up.Email != nil {
		u.Email = *up.Email
	}

	return nil
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

// TODO: Make private, make user_tests go through the NewUser public path instead
func ValidateDisplayName(input string) (string, error) {
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

// TODO: Make private, make user_tests go through the NewUser public path instead
func ValidateUsername(username string) (string, error) {
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

// TODO: Make private, make user_tests go through the NewUser public path instead
func ValidatePassword(password string) error {
	minPasswordLength := 8
	maxPasswordLength := 72
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: too short (min %d chars)", ErrInvalidPassword, minPasswordLength)
	}

	if len(password) > maxPasswordLength {
		return fmt.Errorf("%w: too long (max %d chars)", ErrInvalidPassword, maxPasswordLength)
	}

	return nil
}
