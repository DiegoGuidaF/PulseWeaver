//go:build test

package authz_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/WallyDex/internal/authz"
	"github.com/matryer/is"
)

func TestHandler_MissingAuthHeader_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("X-Real-IP", "1.2.3.4")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_WrongToken_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer wrongtoken")
	r.Header.Set("X-Real-IP", "1.2.3.4")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_EmptySecret_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "") // empty secret = fail-closed
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer anything")
	r.Header.Set("X-Real-IP", "1.2.3.4")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_MissingXRealIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_InvalidIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r.Header.Set("X-Real-IP", "not-an-ip")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_AllowedIP_Returns200(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r.Header.Set("X-Real-IP", "1.2.3.4")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)
}

func TestHandler_DisabledIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r.Header.Set("X-Real-IP", "9.9.9.9")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_IPv6Normalisation(t *testing.T) {
	is := is.New(t)
	// "::1" is the normalized form; the cache should store the normalized form
	h := newTestHandler([]string{"::1"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r.Header.Set("X-Real-IP", "::1")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)
}

// newTestHandler creates a Handler pre-populated with the given IPs in its cache.
func newTestHandler(enabledIPs []string, secret string) *authz.Handler {
	provider := &mockProvider{ips: enabledIPs}
	svc := authz.NewService(provider, secret, noopLogger())
	_ = svc.Initialize(context.Background())
	return authz.NewHandler(svc, noopLogger())
}
