package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
	httpSwagger "github.com/swaggo/http-swagger"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

type DeviceHandler = device.HTTPHandler
type AuthHandler = auth.HTTPHandler

type CompositeHandler struct {
	*DeviceHandler
	*AuthHandler
}

func NewServer(deviceHandler *DeviceHandler, authHandler *AuthHandler, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	loggerConfig := slogchi.Config{
		WithRequestID: true,
	}
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP) // Get real IP when behind proxy
	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.SetHeader("X-Content-Type-Options", "nosniff"))
	r.Use(middleware.SetHeader("X-Frame-Options", "DENY"))

	addRoutes(r, deviceHandler, authHandler)

	return r
}

func addRoutes(r *chi.Mux, deviceHandler *DeviceHandler, authHandler *AuthHandler) {
	routeHandler := &CompositeHandler{DeviceHandler: deviceHandler, AuthHandler: authHandler}

	r.Get("/health", health.Handler)

	r.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		swagger, _ := api.GetSwagger()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swagger)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger.json"),
	))

	r.Route("/api/v1", func(r chi.Router) {
		swagger, _ := api.GetSwagger()

		validatorOptions := &nethttpmiddleware.Options{
			ErrorHandler: validationErrorHandler,
			Options: openapi3filter.Options{
				AuthenticationFunc: auth.AuthenticationFunc(authHandler.Authenticator()),
			},
		}

		// OpenApi request input validators
		r.Use(nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, validatorOptions))
		// Inject auth token into context if present
		r.Use(auth.PrincipalContextMiddleware(authHandler.Authenticator()))
		// Inject request client IP into context
		r.Use(device.ClientIPContextMiddleware())

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
