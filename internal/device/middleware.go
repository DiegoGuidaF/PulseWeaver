package device

import (
	"context"
	"errors"
	"net/http"
)

const ApiKeyHeaderName = "X-API-Key"

// ApiKeyAuthenticator defines the interface for authenticating device API keys.
// This interface is defined in the device package to avoid import cycles.
type ApiKeyAuthenticator interface {
	Authenticate(ctx context.Context, rawKey string) (*Principal, error)
}

// apiKeyFromRequest extracts the API key from the X-API-Key header.
// This keeps device middleware decoupled from auth package.
func apiKeyFromRequest(r *http.Request) (string, error) {
	apiKey := r.Header.Get(ApiKeyHeaderName)
	if apiKey == "" {
		return "", errors.New("missing API key")
	}
	return apiKey, nil
}

// ClientIPContextMiddleware Retrieves the client IP from the request and injects it into request context.
func ClientIPContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP from request (RealIP middleware sets RemoteAddr)
			clientIP := r.RemoteAddr
			ctx := WithClientIP(r.Context(), clientIP)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// PrincipalDeviceContextMiddleware resolves the api key into a Device principal and injects it into the context.
func PrincipalDeviceContextMiddleware(apiKeyAuthenticator ApiKeyAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey, err := apiKeyFromRequest(r)
			if err == nil {
				principal, authErr := apiKeyAuthenticator.Authenticate(r.Context(), apiKey)
				if authErr == nil {
					ctx := WithPrincipal(r.Context(), *principal)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
