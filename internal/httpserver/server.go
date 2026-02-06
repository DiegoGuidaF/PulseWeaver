package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/ui"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oapi-codegen/nethttp-middleware"
	"github.com/samber/slog-chi"
	httpSwagger "github.com/swaggo/http-swagger"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
)

func NewServer(openApiHandler *device.OpenApiHandler, logger *slog.Logger) http.Handler {
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

	addRoutes(r, openApiHandler)

	return r
}

func addRoutes(r *chi.Mux, openApiHandler *device.OpenApiHandler) {
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
		}

		r.Use(nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, validatorOptions))

		// Middleware to extract client IP and add to context
		clientIPMiddleware := func(next api.StrictHandlerFunc, operationName string) api.StrictHandlerFunc {
			return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
				// Extract client IP from request (RealIP middleware sets RemoteAddr)
				clientIP := r.RemoteAddr
				ctx = context.WithValue(ctx, "client_ip", clientIP)
				return next(ctx, w, r, request)
			}
		}

		strictHandler := api.NewStrictHandler(openApiHandler, []api.StrictMiddlewareFunc{clientIPMiddleware})
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
