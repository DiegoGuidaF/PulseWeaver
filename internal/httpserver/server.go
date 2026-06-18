package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"
)

func NewServer(
	deviceHandler *DeviceHandler,
	authHandler *AuthHandler,
	ruleHandler *RuleHandler,
	queriesHandler *QueriesHandler,
	policyHandler *PolicyHandler,
	accessLogHandler *AccessLogHandler,
	rollupHandler *RollupHandler,
	pairingHandler *DevicePairingHandler,
	hostsHandler *HostsHandler,
	userAccessHandler *UserAccessHandler,
	networkPoliciesHandler *NetworkPoliciesHandler,
	logger *slog.Logger,
	trustedProxy netip.Addr,
) http.Handler {
	r := chi.NewRouter()

	loggerConfig := slogchi.Config{
		WithRequestID: true,
	}
	r.Use(middleware.RequestID)
	r.Use(RequestLoggerMiddleware(logger))
	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)

	// Retrieve client IP from X-Real-IP when the direct peer is a trusted proxy
	// prefix, otherwise extract directly from RemoteAddr.
	if trustedProxy.IsValid() {
		r.Use(ClientIPFromRealIPMiddleware(trustedProxy, logger))
	} else {
		r.Use(ClientIPFromRequestMiddleware())
	}

	// Set security policies
	r.Use(middleware.SetHeader("X-Content-Type-Options", "nosniff"))
	r.Use(middleware.SetHeader("X-Frame-Options", "DENY"))
	r.Use(middleware.SetHeader("Strict-Transport-Security", "max-age=63072000; includeSubDomains"))
	r.Use(middleware.SetHeader("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'"))
	r.Use(middleware.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin"))
	r.Use(middleware.SetHeader("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()"))
	r.Use(MaxBodySizeMiddleware(256 * 1024)) // 256KB

	addRoutes(r, deviceHandler, authHandler, ruleHandler, queriesHandler, policyHandler, accessLogHandler, rollupHandler, pairingHandler, hostsHandler, userAccessHandler, networkPoliciesHandler, logger)

	return r
}

// createValidationErrorHandler returns an OpenAPI validation error handler that logs
// rejected requests and responds with the standard JSON error shape.
// Note: the ErrorHandler signature has no *http.Request, so request-scoped fields
// (request ID, path) are unavailable here — they appear in the slog-chi access log instead.
func createValidationErrorHandler(logger *slog.Logger) func(http.ResponseWriter, string, int) {
	return func(w http.ResponseWriter, msg string, statusCode int) {
		logger.Warn("openapi validation error",
			slog.String(logging.AttrKeyError, msg),
			slog.Int("status", statusCode),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		response := httpapi.ErrorResponse{
			Error: &msg,
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

// createRequestErrorHandler creates a request error handler that logs errors with request context
// and returns proper JSON error responses.
func createRequestErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger.WarnContext(r.Context(), "request decode error",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Any(logging.AttrKeyError, err),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		response := httpapi.ErrorResponse{
			Error: new(err.Error()),
		}
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			// If encoding fails, response headers are already sent, log error
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

// createResponseErrorHandler creates a response error handler that logs errors with request context
// and returns proper JSON error responses.
func createResponseErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		// Database write contention is a transient availability condition, not a
		// server fault: degrade to 503 + Retry-After so clients retry instead of
		// treating it as a 500. Logged at Warn — it is expected under write bursts.
		if errors.Is(err, database.ErrContended) {
			logger.WarnContext(r.Context(), "write contention, responding 503",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Any(logging.AttrKeyError, err),
			)
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			if encodeErr := json.NewEncoder(w).Encode(httpapi.ErrorResponse{
				Error: new("Service temporarily unavailable, please retry"),
			}); encodeErr != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
			return
		}

		logger.ErrorContext(r.Context(), "response error",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Any(logging.AttrKeyError, err),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		response := httpapi.ErrorResponse{
			Error: new("Internal server error"),
		}
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			// If encoding fails, response headers are already sent, log error
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}
