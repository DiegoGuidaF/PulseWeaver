package httpserver

import (
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/devicepairing"
	"github.com/DiegoGuidaF/PulseWeaver/internal/health"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ui"
	"github.com/DiegoGuidaF/PulseWeaver/internal/useraccess"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

type CompositeHandler struct {
	*DeviceHandler
	*AuthHandler
	*RuleHandler
	*QueriesHandler
	*RollupHandler
	*DevicePairingHandler
	*HostsHandler
	*UserAccessHandler
	*PolicyHandler
	*NetworkPoliciesHandler
}

type RuleHandler = rule.HTTPHandler
type QueriesHandler = queries.HTTPHandler
type DeviceHandler = device.HTTPHandler
type AuthHandler = auth.HTTPHandler
type PolicyHandler = policy.HTTPHandler
type RollupHandler = rollup.HTTPHandler
type DevicePairingHandler = devicepairing.HTTPHandler
type HostsHandler = hosts.HTTPHandler
type UserAccessHandler = useraccess.HTTPHandler
type NetworkPoliciesHandler = networkpolicies.HTTPHandler

func addRoutes(
	r *chi.Mux,
	deviceHandler *DeviceHandler,
	authHandler *AuthHandler,
	ruleHandler *RuleHandler,
	queriesHandler *QueriesHandler,
	policyHandler *PolicyHandler,
	rollupHandler *RollupHandler,
	pairingHandler *DevicePairingHandler,
	hostsHandler *HostsHandler,
	userAccessHandler *UserAccessHandler,
	networkPoliciesHandler *NetworkPoliciesHandler,
	logger *slog.Logger,
) {
	routeHandler := &CompositeHandler{
		DeviceHandler:          deviceHandler,
		AuthHandler:            authHandler,
		RuleHandler:            ruleHandler,
		QueriesHandler:         queriesHandler,
		RollupHandler:          rollupHandler,
		DevicePairingHandler:   pairingHandler,
		HostsHandler:           hostsHandler,
		UserAccessHandler:      userAccessHandler,
		PolicyHandler:          policyHandler,
		NetworkPoliciesHandler: networkPoliciesHandler,
	}

	r.Get("/health", health.Handler)
	// verify-ip is the forward-auth endpoint registered outside /api/v1. The
	// proxy authenticates to it with a static bearer token; the token check is
	// constant-time, so it is not rate limited here.
	r.Get(httpapi.VerifyIPEndpoint, policyHandler.HandleForwardAuthIP)

	r.Route("/api/v1", func(r chi.Router) {

		swagger, _ := httpapi.GetSwagger()

		validatorOptions := &nethttpmiddleware.Options{
			ErrorHandler: createValidationErrorHandler(logger),
			// The spec's only server URL is the path-only "/api/v1" prefix, which
			// carries no host for the validator to match against, so its Host-header
			// validation never rejects a request. Silence the blanket warning.
			SilenceServersWarning: true,
			Options: openapi3filter.Options{
				AuthenticationFunc: AuthenticationFunc(authHandler.UserAuthenticator(), deviceHandler.APIKeyAuthenticator()),
			},
		}

		// Rate limit unauthenticated endpoints by IP
		r.Use(LoginRateLimitMiddleware(5, time.Minute, logger))
		r.Use(HeartbeatRateLimitMiddleware(30, time.Minute, logger))
		r.Use(DevicePairingRateLimitMiddleware(10, time.Minute, logger))

		r.Use(nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, validatorOptions))
		// Inject auth token into context if present
		r.Use(auth.PrincipalUserContextMiddleware(authHandler.UserAuthenticator()))
		// Inject device API key into context if present
		r.Use(device.PrincipalDeviceContextMiddleware(deviceHandler.APIKeyAuthenticator()))
		// Enforce admin invariant: any user session principal must be an admin
		r.Use(auth.RequireAdmin)

		errorOptions := httpapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  createRequestErrorHandler(logger),
			ResponseErrorHandlerFunc: createResponseErrorHandler(logger),
		}

		middlewares := []httpapi.StrictMiddlewareFunc{contentionMiddleware}
		strictHandler := httpapi.NewStrictHandlerWithOptions(routeHandler, middlewares, errorOptions)
		httpapi.HandlerFromMux(strictHandler, r)
	})

	// Any other path would go to the UI
	r.Handle("/*", ui.Handler())
}
