package device

import (
	"context"
	"errors"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpapi"
)

// APIKeyAuthenticator defines the interface for authenticating device API keys.
type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, rawKey string) (*Principal, error)
}

// apiKeyFromRequest extracts the API key from the X-API-Key header.
func apiKeyFromRequest(r *http.Request) (string, error) {
	apiKey := r.Header.Get(httpapi.APIKeyHeaderName)
	if apiKey == "" {
		return "", errors.New("missing API key")
	}
	return apiKey, nil
}

// PrincipalDeviceContextMiddleware resolves the api key into a Device Principal and injects it into the context.
func PrincipalDeviceContextMiddleware(apiKeyAuthenticator APIKeyAuthenticator) func(http.Handler) http.Handler {
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
