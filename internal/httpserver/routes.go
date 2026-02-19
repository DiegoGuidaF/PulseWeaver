package httpserver

import (
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/ui"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type CompositeHandler struct {
	*DeviceHandler
	*AuthHandler
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

		// Create custom error handlers with logging
		errorOptions := api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  createRequestErrorHandler(),
			ResponseErrorHandlerFunc: createResponseErrorHandler(),
		}

		strictHandler := api.NewStrictHandlerWithOptions(routeHandler, nil, errorOptions)
		api.HandlerFromMux(strictHandler, r)
	})

	// Any other path would go to the UI
	r.Handle("/*", ui.Handler())
}
