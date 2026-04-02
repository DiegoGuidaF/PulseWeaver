package device

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/go-chi/chi/v5"
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

// OwnershipMiddleware enforces device ownership for routes that include a {device_id} path parameter.
// Admins always pass through. Regular users receive 404 if the device does not belong to them.
// Routes without a device_id parameter (e.g. GET /devices) are unaffected.
func OwnershipMiddleware(service *Service, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawID := chi.URLParam(r, "device_id")
			if rawID == "" {
				next.ServeHTTP(w, r)
				return
			}

			principal, ok := auth.PrincipalFromContext(r.Context())
			if !ok || principal.IsAdmin() {
				next.ServeHTTP(w, r)
				return
			}

			deviceID, err := strconv.ParseInt(rawID, 10, 64)
			if err != nil {
				// Malformed ID — let the downstream handler return a proper 400.
				next.ServeHTTP(w, r)
				return
			}

			device, err := service.GetDevice(r.Context(), DeviceID(deviceID))
			if err != nil || device.OwnerID != principal.UserID {
				logger.WarnContext(r.Context(), "unauthorized device access attempt",
					slog.Int64("user_id", principal.UserID.Int64()),
					slog.Int64("device_id", deviceID),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "device not found"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
