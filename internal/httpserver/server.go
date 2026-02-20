package httpserver

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	slogchi "github.com/samber/slog-chi"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

type DeviceHandler = device.HTTPHandler
type AuthHandler = auth.HTTPHandler

func NewServerFromConfig(deviceHandler *DeviceHandler, authHandler *AuthHandler, logger *slog.Logger, conf config.ConfServer) (http.Handler, error) {
	trustedProxy, err := ParseTrustedProxy(conf.TrustedProxy)
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

	// Retrieve ClientIP from X-Forwarded-For header if remoteAddr is a trusted proxy,
	// otherwise extract directly from RemoteAddr.
	// This is critical for retrieving the ClientIP automatically via the request IP itself instead of having to
	// rely on the IP sent as part of the body
	if trustedProxy.IsValid() {
		r.Use(ClientIPFromXFFHeaderMiddleware(trustedProxy))
	} else {
		r.Use(ClientIpFromRequestMiddleware())
	}

	r.Use(RequestLoggerMiddleware(logger))
	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)

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

	response := api.ErrorResponse{
		Error: &msg,
	}
	json.NewEncoder(w).Encode(response)
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
		response := api.ErrorResponse{
			Error: &errorMsg,
		}
		json.NewEncoder(w).Encode(response)
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
		response := api.ErrorResponse{
			Error: &errorMsg,
		}
		json.NewEncoder(w).Encode(response)
	}
}
