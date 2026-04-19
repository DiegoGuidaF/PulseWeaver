package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

// UserAuthenticator defines the interface for authenticating user sessions.
// This interface is defined in the auth package to avoid import cycles.
type UserAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*Principal, error)
}

// PrincipalUserContextMiddleware resolves the authenticated principal into a user principal and injects it into the
// request context.
func PrincipalUserContextMiddleware(auth UserAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := TokenFromRequest(r)
			if err == nil {
				principal, authErr := auth.Authenticate(r.Context(), token)
				if authErr == nil {
					ctx := WithPrincipal(r.Context(), *principal)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin is an invariant enforcer: any user session principal must be an admin.
// Device-principal requests (no user principal in context) pass through unchecked.
// Must run after PrincipalUserContextMiddleware.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if ok && !principal.IsAdmin() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "admin credentials required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TokenFromRequest(r *http.Request) (string, error) {
	if cookie, err := r.Cookie(httpapi.SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	return "", errors.New("missing auth credentials")
}
