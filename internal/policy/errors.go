package policy

import "errors"

var (
	ErrSecretNotConfigured = errors.New("policy secret not configured")
	ErrInvalidBearerToken  = errors.New("invalid bearer token")
	ErrIPNotEnabled        = errors.New("IP not in enabled set")
	ErrHostNotAllowed      = errors.New("host not in allowlist")
)
