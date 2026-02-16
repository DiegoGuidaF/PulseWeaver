package httpserver

import (
	"context"
	"errors"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"github.com/getkin/kin-openapi/openapi3filter"
)

type UserAuthenticator = auth.UserAuthenticator

type ApiKeyAuthenticator = device.ApiKeyAuthenticator

// SessionCookieName is defined in auth package to avoid import cycles
const SessionCookieName = auth.SessionCookieName
const ApiKeyHeaderName = device.ApiKeyHeaderName

// AuthenticationFunc is called by the OapiRequestValidator to verify security schemes
func AuthenticationFunc(auth UserAuthenticator, apiKeyAuthenticator ApiKeyAuthenticator) func(context.Context, *openapi3filter.AuthenticationInput) error {
	return func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
		switch input.SecuritySchemeName {
		case "cookieAuth":
			// Extract from Cookie
			if input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil {
				return errors.New("no request context")
			}
			cookie, err := input.RequestValidationInput.Request.Cookie(SessionCookieName)
			if err != nil {
				return errors.New("missing session cookie")
			}
			token := cookie.Value
			_, err = auth.Authenticate(ctx, token)
			if err != nil {
				return errors.New("invalid credentials")
			}

		case "apiKeyAuth":
			// Extract from X-API-Key header
			if input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil {
				return errors.New("no request context")
			}
			apiKey := input.RequestValidationInput.Request.Header.Get(ApiKeyHeaderName)
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
