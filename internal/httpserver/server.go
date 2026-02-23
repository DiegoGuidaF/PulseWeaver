package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpapi"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

type DeviceHandler = device.HTTPHandler
type AuthHandler = auth.HTTPHandler

func NewServerFromConfig(deviceHandler *DeviceHandler, authHandler *AuthHandler, logger *slog.Logger, conf config.ConfServer) (http.Handler, error) {
	trustedProxy, err := parseTrustedProxy(conf.TrustedProxy)
	if err != nil {
		return nil, fmt.Errorf("parse trusted proxy: %w", err)
	}

	return NewServer(deviceHandler, authHandler, logger, trustedProxy), nil
}

func NewServer(deviceHandler *DeviceHandler, authHandler *AuthHandler, logger *slog.Logger, trustedProxy netip.Addr) http.Handler {
	r := chi.NewRouter()

	loggerConfig := slogchi.Config{
		WithRequestID: true,
	}
	r.Use(middleware.RequestID)
	r.Use(RequestLoggerMiddleware(logger))
	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)

	// Retrieve ClientIP from X-Forwarded-For header if remoteAddr is a trusted proxy,
	// otherwise extract directly from RemoteAddr.
	// This is critical for retrieving the ClientIP automatically via the request IP itself instead of having to
	// rely on the IP sent as part of the body
	if trustedProxy.IsValid() {
		r.Use(ClientIPFromXFFHeaderMiddleware(trustedProxy))
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

	addRoutes(r, deviceHandler, authHandler)

	return r
}

// validationErrorHandler OpenApi validation errors match rest of app JSON with "error" key
func validationErrorHandler(w http.ResponseWriter, msg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := httpapi.ErrorResponse{
		Error: &msg,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// If encoding fails, response headers are already sent, log error
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// createRequestErrorHandler creates a request error handler that logs errors with request context
// and returns proper JSON error responses.
func createRequestErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger := logging.FromCtx(r.Context())
		logger.Warn("request decode error",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Any("error", err),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		errorMsg := err.Error()
		response := httpapi.ErrorResponse{
			Error: &errorMsg,
		}
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			// If encoding fails, response headers are already sent, log error
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

// createResponseErrorHandler creates a response error handler that logs errors with request context
// and returns proper JSON error responses.
func createResponseErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger := logging.FromCtx(r.Context())
		logger.Error("response error",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Any("error", err),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		errorMsg := "Internal server error"
		response := httpapi.ErrorResponse{
			Error: &errorMsg,
		}
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			// If encoding fails, response headers are already sent, log error
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}

// ParseTrustedProxy parses a single plain IP address (IPv4 or IPv6).
// Returns an error if the IP is invalid, contains CIDR notation, or contains commas.
func parseTrustedProxy(s string) (netip.Addr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Addr{}, nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("TRUSTED_PROXY must be a single plain IP address (no CIDR notation, no comma-separated lists). Got: '%s'", s)
	}

	return addr, nil
}
