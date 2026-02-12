package auth

import "errors"

var (
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrAdminCredentialsRequired = errors.New("admin credentials are required")
	ErrUsernameTaken            = errors.New("username already taken")
	ErrEmailTaken               = errors.New("email already taken")
	ErrUserNotFound             = errors.New("user not found")
	ErrInvalidUsername          = errors.New("invalid username")
	ErrInvalidDisplayName       = errors.New("invalid display name")
	ErrInvalidPassword          = errors.New("invalid password")
)
