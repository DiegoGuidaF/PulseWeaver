package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/ui"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
	slogchi "github.com/samber/slog-chi"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

type DeviceHandler = device.HTTPHandler
type AuthHandler = auth.HTTPHandler

type CompositeHandler struct {
	*DeviceHandler
	*AuthHandler
}

func NewServer(deviceHandler *DeviceHandler, authHandler *AuthHandler, logger *slog.Logger, trustedProxy netip.Addr) http.Handler {
	r := chi.NewRouter()

	loggerConfig := slogchi.Config{
		WithRequestID: true,
	}
	r.Use(middleware.RequestID)

	// Retrieve ClientApi from X-Forwarded-For header if remoteAddr is a trusted proxy
	// This is critical for retrieving the ClientIP automatically via the request IP itself instead of having to
	// rely on the IP sent as part of the body
	r.Use(ClientIPFromXFFHeader(trustedProxy))

	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)

	// Set security policies
	r.Use(middleware.SetHeader("X-Content-Type-Options", "nosniff"))
	r.Use(middleware.SetHeader("X-Frame-Options", "DENY"))
	r.Use(middleware.SetHeader("Strict-Transport-Security", "max-age=63072000; includeSubDomains"))
	r.Use(middleware.SetHeader("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'"))
	r.Use(middleware.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin"))
	r.Use(middleware.SetHeader("Permissions-Policy", "camera=(), microphone=(), geolocation=(), interest-cohort=()"))
	r.Use(MaxBodySize(256 * 1024)) // 256KB

	addRoutes(r, deviceHandler, authHandler)

	return r
}

func addRoutes(r *chi.Mux, deviceHandler *DeviceHandler, authHandler *AuthHandler) {
	routeHandler := &CompositeHandler{DeviceHandler: deviceHandler, AuthHandler: authHandler}

	r.Get("/health", health.Handler)

	r.Route("/api/v1", func(r chi.Router) {

		swagger, _ := api.GetSwagger()

		validatorOptions := &nethttpmiddleware.Options{
			ErrorHandler: validationErrorHandler,
			Options: openapi3filter.Options{
				AuthenticationFunc: AuthenticationFunc(authHandler.UserAuthenticator(), deviceHandler.ApiKeyAuthenticator()),
			},
		}

		// Rate limit login: 5 requests per minute per IP; other endpoints not limited
		r.Use(LoginRateLimitMiddleware(5, time.Minute))

		// OpenApi request input validators
		r.Use(nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, validatorOptions))
		// Inject auth token into context if present
		r.Use(auth.PrincipalUserContextMiddleware(authHandler.UserAuthenticator()))
		// Inject auth token into context if present
		r.Use(device.PrincipalDeviceContextMiddleware(deviceHandler.ApiKeyAuthenticator()))

		strictHandler := api.NewStrictHandler(routeHandler, nil)
		api.HandlerFromMux(strictHandler, r)
	})

	// Any other path would go to the UI
	r.Handle("/*", ui.Handler())
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
