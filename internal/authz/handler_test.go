//go:build test

package authz_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/WallyDex/internal/authz"
	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/matryer/is"
)

func TestHandler_MissingAuthHeader_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_WrongToken_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer wrongtoken")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_EmptySecret_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "") // empty secret = fail-closed
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer anything")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_MissingClientIPInContext_Returns403(t *testing.T) {
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
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "not-an-ip"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_AllowedIP_Returns200(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)
}

func TestHandler_DisabledIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"}, "mysecret")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "9.9.9.9"))
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
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "::1"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)
}

func TestHandler_ProxyIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandlerWithProxy([]string{"127.0.0.1"}, "mysecret", "127.0.0.1")
	r := httptest.NewRequest(http.MethodGet, "/api/authz/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "127.0.0.1"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

// newTestHandler creates a Handler pre-populated with the given IPs in its cache.
func newTestHandler(enabledIPs []string, secret string) *authz.Handler {
	return newTestHandlerWithProxy(enabledIPs, secret, "")
}

func newTestHandlerWithProxy(enabledIPs []string, secret, trustedProxy string) *authz.Handler {
	provider := &mockProvider{ips: enabledIPs}
	var proxyAddr netip.Addr
	if trustedProxy != "" {
		proxyAddr = netip.MustParseAddr(trustedProxy)
	}
	svc := authz.NewService(provider, secret, noopLogger(), proxyAddr)
	_ = svc.Initialize(context.Background())
	return authz.NewHandler(svc, noopLogger())
}
