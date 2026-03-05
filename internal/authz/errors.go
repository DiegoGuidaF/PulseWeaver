package authz

import "errors"

var (
	ErrSecretNotConfigured = errors.New("authz secret not configured")
	ErrInvalidBearerToken  = errors.New("invalid bearer token")
	ErrIPNotEnabled        = errors.New("IP not in enabled set")
)
