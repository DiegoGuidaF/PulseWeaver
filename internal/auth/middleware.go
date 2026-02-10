package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*Principal, error)
}

const SessionCookieName = "__Host-wdc_session"

// AuthenticationFunc is called by the OapiRequestValidator to verify security schemes
func AuthenticationFunc(auth Authenticator) func(context.Context, *openapi3filter.AuthenticationInput) error {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		// input.SecuritySchemeName tells you which scheme is being validated
		// ("cookieAuth" or "bearerAuth")

		var token string

		switch input.SecuritySchemeName {
		case "cookieAuth":
			// Extract from Cookie
			if input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil {
				return errors.New("no request context")
			}
			cookie, err := input.RequestValidationInput.Request.Cookie(SessionCookieName)
			if err != nil {
				return errors.New("missing session cookie")
			}
			token = cookie.Value

		case "bearerAuth":
			// Extract from Authorization ggHeader
			if input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil {
				return errors.New("no request context")
			}
			authHeader := input.RequestValidationInput.Request.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return errors.New("missing bearer token")
			}
			token = strings.TrimPrefix(authHeader, "Bearer ")

		default:
			return errors.New("unknown security scheme")
		}

		// Validate the token. We cannot store it in the context here
		_, err := auth.Authenticate(ctx, token)
		if err != nil {
			return errors.New("invalid credentials")
		}

		return nil
	}
}

// PrincipalContextMiddleware resolves the authenticated principal and injects it into request context.
func PrincipalContextMiddleware(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := tokenFromRequest(r)
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

func tokenFromRequest(r *http.Request) (string, error) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != "" {
			return token, nil
		}
	}

	return "", errors.New("missing auth credentials")
}
