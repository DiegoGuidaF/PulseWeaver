package httpserver

import (
	"context"
	"errors"

	"github.com/DiegoGuidaF/WallyDex/internal/auth"
	"github.com/DiegoGuidaF/WallyDex/internal/device"
	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/getkin/kin-openapi/openapi3filter"
)

type UserAuthenticator = auth.UserAuthenticator
type APIKeyAuthenticator = device.APIKeyAuthenticator

// AuthenticationFunc is called by the OapiRequestValidator to verify security schemes
func AuthenticationFunc(auth UserAuthenticator, apiKeyAuthenticator APIKeyAuthenticator) func(context.Context, *openapi3filter.AuthenticationInput) error {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		switch input.SecuritySchemeName {
		case httpapi.CookieAuthScope:
			// Extract from Cookie
			if input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil {
				return errors.New("no request context")
			}
			cookie, err := input.RequestValidationInput.Request.Cookie(httpapi.SessionCookieName)
			if err != nil {
				return errors.New("missing session cookie")
			}
			token := cookie.Value
			_, err = auth.Authenticate(ctx, token)
			if err != nil {
				return errors.New("invalid credentials")
			}

		case httpapi.APIKeyAuthScope:
			// Extract from X-API-Key header
			if input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil {
				return errors.New("no request context")
			}
			apiKey := input.RequestValidationInput.Request.Header.Get(httpapi.APIKeyHeaderName)
			if apiKey == "" {
				return errors.New("missing API key")
			}

			// Validate the API key
			if apiKeyAuthenticator == nil {
				return errors.New("API key validator not configured")
			}
			_, err := apiKeyAuthenticator.Authenticate(ctx, apiKey)
			if err != nil {
				return errors.New("invalid API key")
			}
			return nil

		default:
			return errors.New("unknown security scheme")
		}

		return nil
	}
}
