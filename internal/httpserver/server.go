package httpserver

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
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
		WithSpanID:  true,
		WithTraceID: true,
	}

	r.Use(slogchi.NewWithConfig(logger, loggerConfig))
	r.Use(middleware.Recoverer)

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

		strictHandler := api.NewStrictHandler(openApiHandler, nil)
		api.HandlerFromMux(strictHandler, r)
	})

	//// Devices
	//r.Get("/api/v1/devices", deviceHandler.GetDevicesv1)
	//r.Post("/api/v1/devices", deviceHandler.CreateDevicev1)

	//// IP routes
	//r.Get("/api/v1/devices/{id}/ips", deviceHandler.ListDeviceIPsv1)
	//r.Post("/api/v1/devices/{id}/ips", deviceHandler.AssignIP)
	//r.Patch("/api/v1/devices/{id}/ips/{ip_id}/disable", deviceHandler.DisableDeviceIP)
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
