//go:build test

package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

// fakeUserAuthenticator implements auth.UserAuthenticator for middleware tests.
type fakeUserAuthenticator struct {
	principal *auth.Principal
	err       error
}

func (f *fakeUserAuthenticator) Authenticate(_ context.Context, _ string) (*auth.Principal, error) {
	return f.principal, f.err
}

var _ auth.UserAuthenticator = (*fakeUserAuthenticator)(nil)

// TokenFromRequest

func TestTokenFromRequest_ValidCookie_ReturnsToken(t *testing.T) {
	is := is.New(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "tok-abc"})

	token, err := auth.TokenFromRequest(req)

	is.NoErr(err)
	is.Equal(token, "tok-abc")
}

func TestTokenFromRequest_MissingCookie_ReturnsError(t *testing.T) {
	is := is.New(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := auth.TokenFromRequest(req)

	is.True(err != nil)
}

func TestTokenFromRequest_EmptyCookieValue_ReturnsError(t *testing.T) {
	is := is.New(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: ""})

	_, err := auth.TokenFromRequest(req)

	is.True(err != nil)
}

// PrincipalUserContextMiddleware

func TestPrincipalUserContextMiddleware_ValidToken_InjectsPrincipal(t *testing.T) {
	is := is.New(t)
	principal := auth.NewPrincipal(ids.UserID(1), ids.SessionID(1), auth.UserRole)
	authenticator := &fakeUserAuthenticator{principal: principal}

	var captured *auth.Principal
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := auth.PrincipalFromContext(r.Context())
		if ok {
			captured = p
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.PrincipalUserContextMiddleware(authenticator)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "valid-token"})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	is.True(captured != nil)
	is.Equal(captured.UserID, ids.UserID(1))
}

func TestPrincipalUserContextMiddleware_MissingCookie_PassesThrough(t *testing.T) {
	is := is.New(t)
	authenticator := &fakeUserAuthenticator{principal: auth.NewPrincipal(ids.UserID(1), ids.SessionID(1), auth.UserRole)}

	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		_, ok := auth.PrincipalFromContext(r.Context())
		is.True(!ok)
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.PrincipalUserContextMiddleware(authenticator)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil) // no cookie
	handler.ServeHTTP(httptest.NewRecorder(), req)

	is.True(reached)
}

func TestPrincipalUserContextMiddleware_InvalidToken_PassesThrough(t *testing.T) {
	is := is.New(t)
	authenticator := &fakeUserAuthenticator{err: errors.New("invalid token")}

	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		_, ok := auth.PrincipalFromContext(r.Context())
		is.True(!ok)
		w.WriteHeader(http.StatusOK)
	})

	handler := auth.PrincipalUserContextMiddleware(authenticator)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: "bad-token"})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	is.True(reached)
}
