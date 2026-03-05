package httpserver

import (
	"log/slog"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/health"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpapi"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/rule"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/ui"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type CompositeHandler struct {
	*DeviceHandler
	*AuthHandler
	*RuleHandler
}

type RuleHandler = rule.HTTPHandler

func addRoutes(r *chi.Mux, deviceHandler *DeviceHandler, authHandler *AuthHandler, ruleHandler *RuleHandler, authzHandler *AuthzHandler, logger *slog.Logger) {
	routeHandler := &CompositeHandler{
		DeviceHandler: deviceHandler,
		AuthHandler:   authHandler,
		RuleHandler:   ruleHandler,
	}

	r.Get("/health", health.Handler)
	r.Get("/api/authz/verify-ip", authzHandler.HandleForwardAuthIP)

	r.Route("/api/v1", func(r chi.Router) {

		swagger, _ := httpapi.GetSwagger()

		validatorOptions := &nethttpmiddleware.Options{
			ErrorHandler: validationErrorHandler,
			Options: openapi3filter.Options{
				AuthenticationFunc: AuthenticationFunc(authHandler.UserAuthenticator(), deviceHandler.APIKeyAuthenticator()),
			},
		}

		// Rate limit login: 5 requests per minute per IP; other endpoints not limited
		r.Use(LoginRateLimitMiddleware(5, time.Minute))

		// OpenApi request input validators
		r.Use(nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, validatorOptions))
		// Inject auth token into context if present
		r.Use(auth.PrincipalUserContextMiddleware(authHandler.UserAuthenticator()))
		// Inject auth token into context if present
		r.Use(device.PrincipalDeviceContextMiddleware(deviceHandler.APIKeyAuthenticator()))

		// Create custom error handlers with logging
		errorOptions := httpapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  createRequestErrorHandler(logger),
			ResponseErrorHandlerFunc: createResponseErrorHandler(logger),
		}

		strictHandler := httpapi.NewStrictHandlerWithOptions(routeHandler, nil, errorOptions)
		httpapi.HandlerFromMux(strictHandler, r)
	})

	// Any other path would go to the UI
	r.Handle("/*", ui.Handler())
}
