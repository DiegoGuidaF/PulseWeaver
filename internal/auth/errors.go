package auth

import "errors"

var (
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrAdminCredentialsRequired = errors.New("admin credentials are required")
)
