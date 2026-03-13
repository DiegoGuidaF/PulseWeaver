// internal/auth/cookie.go
package auth

import (
	"net/http"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

const CookieDuration = 7 * 24 * time.Hour

// CookieConfig holds environment-specific cookie settings
type CookieConfig struct {
	Name     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
	Domain   string // Empty for host-only (recommended)
}

// Default production config
var DefaultCookieConfig = CookieConfig{
	Name:     httpapi.SessionCookieName,
	Secure:   true, // Must be true for __Host- prefix
	HTTPOnly: true,
	SameSite: http.SameSiteLaxMode,
	Domain:   "",
}

// NewSessionCookie creates a standardized auth cookie
func NewSessionCookie(token string, cfg CookieConfig) *http.Cookie {
	return &http.Cookie{
		Name:     cfg.Name,
		Value:    token,
		Path:     "/",
		HttpOnly: cfg.HTTPOnly,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Domain:   cfg.Domain,
		MaxAge:   int(CookieDuration.Seconds()),
		Expires:  time.Now().Add(CookieDuration),
	}
}

// ExpireSessionCookie creates a cookie to clear the session
func ExpireSessionCookie(cfg CookieConfig) *http.Cookie {
	return &http.Cookie{
		Name:     cfg.Name,
		Value:    "",
		Path:     "/",
		HttpOnly: cfg.HTTPOnly,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
		Domain:   cfg.Domain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
}
