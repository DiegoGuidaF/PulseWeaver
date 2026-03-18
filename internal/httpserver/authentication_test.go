//go:build test

package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpserver"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/matryer/is"
)

// fakeSessionAuthenticator implements auth.UserAuthenticator.
type fakeSessionAuthenticator struct {
	principal *auth.Principal
	err       error
}

func (f *fakeSessionAuthenticator) Authenticate(_ context.Context, _ string) (*auth.Principal, error) {
	return f.principal, f.err
}

var _ httpserver.UserAuthenticator = (*fakeSessionAuthenticator)(nil)

// fakeAPIKeyAuthenticator implements device.APIKeyAuthenticator.
type fakeAPIKeyAuthenticator struct {
	principal *device.Principal
	err       error
}

func (f *fakeAPIKeyAuthenticator) Authenticate(_ context.Context, _ string) (*device.Principal, error) {
	return f.principal, f.err
}

var _ httpserver.APIKeyAuthenticator = (*fakeAPIKeyAuthenticator)(nil)

// authInput builds a minimal AuthenticationInput for a given scheme and request.
func authInput(scheme string, r *http.Request) *openapi3filter.AuthenticationInput {
	return &openapi3filter.AuthenticationInput{
		SecuritySchemeName: scheme,
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: r,
		},
	}
}

// Cookie scheme

func TestAuthenticationFunc_CookieScheme_ValidToken_ReturnsNil(t *testing.T) {
	is := is.New(t)
	sessionAuth := &fakeSessionAuthenticator{principal: auth.NewPrincipal(auth.UserID(1), auth.SessionID(1), auth.UserRole)}
	fn := httpserver.AuthenticationFunc(sessionAuth, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "valid-token"})

	err := fn(context.Background(), authInput(httpapi.CookieAuthScope, req))
	is.NoErr(err)
}

func TestAuthenticationFunc_CookieScheme_MissingCookie_ReturnsError(t *testing.T) {
	is := is.New(t)
	sessionAuth := &fakeSessionAuthenticator{}
	fn := httpserver.AuthenticationFunc(sessionAuth, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := fn(context.Background(), authInput(httpapi.CookieAuthScope, req))
	is.True(err != nil)
}

func TestAuthenticationFunc_CookieScheme_InvalidToken_ReturnsError(t *testing.T) {
	is := is.New(t)
	sessionAuth := &fakeSessionAuthenticator{err: errors.New("invalid session")}
	fn := httpserver.AuthenticationFunc(sessionAuth, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "bad-token"})

	err := fn(context.Background(), authInput(httpapi.CookieAuthScope, req))
	is.True(err != nil)
}

// API key scheme

func TestAuthenticationFunc_APIKeyScheme_ValidKey_ReturnsNil(t *testing.T) {
	is := is.New(t)
	apiKeyAuth := &fakeAPIKeyAuthenticator{principal: &device.Principal{DeviceID: device.DeviceID(1)}}
	fn := httpserver.AuthenticationFunc(nil, apiKeyAuth)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpapi.APIKeyHeaderName, "wdk_validkey")

	err := fn(context.Background(), authInput(httpapi.APIKeyAuthScope, req))
	is.NoErr(err)
}

func TestAuthenticationFunc_APIKeyScheme_MissingHeader_ReturnsError(t *testing.T) {
	is := is.New(t)
	apiKeyAuth := &fakeAPIKeyAuthenticator{}
	fn := httpserver.AuthenticationFunc(nil, apiKeyAuth)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := fn(context.Background(), authInput(httpapi.APIKeyAuthScope, req))
	is.True(err != nil)
}

func TestAuthenticationFunc_APIKeyScheme_InvalidKey_ReturnsError(t *testing.T) {
	is := is.New(t)
	apiKeyAuth := &fakeAPIKeyAuthenticator{err: errors.New("bad key")}
	fn := httpserver.AuthenticationFunc(nil, apiKeyAuth)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpapi.APIKeyHeaderName, "wdk_badkey")

	err := fn(context.Background(), authInput(httpapi.APIKeyAuthScope, req))
	is.True(err != nil)
}

func TestAuthenticationFunc_APIKeyScheme_NilAuthenticator_ReturnsError(t *testing.T) {
	is := is.New(t)
	fn := httpserver.AuthenticationFunc(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(httpapi.APIKeyHeaderName, "wdk_somekey")

	err := fn(context.Background(), authInput(httpapi.APIKeyAuthScope, req))
	is.True(err != nil)
}

// Unknown scheme

func TestAuthenticationFunc_UnknownScheme_ReturnsError(t *testing.T) {
	is := is.New(t)
	fn := httpserver.AuthenticationFunc(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := fn(context.Background(), authInput("unknownScheme", req))
	is.True(err != nil)
}
