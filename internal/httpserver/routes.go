package httpserver

import (
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/audit"
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/dashboard"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/health"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ui"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type CompositeHandler struct {
	*DeviceHandler
	*AuthHandler
	*RuleHandler
	*QueriesHandler
	*AuditHandler
	*DashboardHandler
}

type RuleHandler = rule.HTTPHandler
type QueriesHandler = queries.HTTPHandler
type DeviceHandler = device.HTTPHandler
type AuthHandler = auth.HTTPHandler
type PolicyHandler = policy.HTTPHandler
type AuditHandler = audit.HTTPHandler
type DashboardHandler = dashboard.HTTPHandler

func addRoutes(r *chi.Mux, deviceHandler *DeviceHandler, authHandler *AuthHandler, ruleHandler *RuleHandler, queriesHandler *QueriesHandler, policyHandler *PolicyHandler, auditHandler *AuditHandler, dashboardHandler *DashboardHandler, logger *slog.Logger) {
	routeHandler := &CompositeHandler{
		DeviceHandler:    deviceHandler,
		AuthHandler:      authHandler,
		RuleHandler:      ruleHandler,
		QueriesHandler:   queriesHandler,
		AuditHandler:     auditHandler,
		DashboardHandler: dashboardHandler,
	}

	r.Get("/health", health.Handler)
	r.Get("/api/policy-engine/verify-ip", policyHandler.HandleForwardAuthIP)

	r.Route("/api/v1", func(r chi.Router) {

		swagger, _ := httpapi.GetSwagger()

		validatorOptions := &nethttpmiddleware.Options{
			ErrorHandler: createValidationErrorHandler(logger),
			Options: openapi3filter.Options{
				AuthenticationFunc: AuthenticationFunc(authHandler.UserAuthenticator(), deviceHandler.APIKeyAuthenticator()),
			},
		}

		// Rate limit unauthenticated endpoints by IP
		r.Use(LoginRateLimitMiddleware(5, time.Minute))
		r.Use(HeartbeatRateLimitMiddleware(30, time.Minute))

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
