//go:build test

package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/httpserver"
	"github.com/DiegoGuidaF/WallyDex/internal/testutils"
	"github.com/matryer/is"
)

func TestClientIpFromRequest_ExtractsFromRemoteAddr(t *testing.T) {
	is := is.New(t)

	// Create a handler that captures client IP from context
	var capturedIP string
	handler := httpserver.ClientIPFromRequestMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request with X-Forwarded-For header (should be ignored)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(httpapi.XForwardedFor, "192.0.2.100")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain the client IP from RemoteAddr, ignoring XFF header
	is.Equal(capturedIP, "127.0.0.1")
}

func TestClientIpFromRequest_HandlesPlainAddress(t *testing.T) {
	is := is.New(t)

	var capturedIP string
	handler := httpserver.ClientIPFromRequestMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request with plain address format (no port)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.200"
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	is.Equal(capturedIP, "192.0.2.200")
}

func TestClientIPFromXFFHeader_TrustedProxyExtractsClientIP(t *testing.T) {
	is := is.New(t)

	// Setup trusted proxy IP
	trustedProxy := netip.MustParseAddr("127.0.0.1")

	// Create a handler that captures client IP from context
	var capturedIP string
	handler := httpserver.ClientIPFromXFFHeaderMiddleware(trustedProxy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request from trusted proxy with X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(httpapi.XForwardedFor, "192.0.2.100")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain the client IP from X-Forwarded-For
	is.Equal(capturedIP, "192.0.2.100")
}

func TestClientIPFromXFFHeader_UntrustedProxyIgnoresXFF(t *testing.T) {
	is := is.New(t)

	// Setup trusted proxy IP (only 127.0.0.1)
	trustedProxy := netip.MustParseAddr("127.0.0.1")

	// Create a handler that captures client IP from context
	var capturedIP string
	handler := httpserver.ClientIPFromXFFHeaderMiddleware(trustedProxy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request from untrusted IP with X-Forwarded-For
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.200:12345"              // Not the trusted proxy
	req.Header.Set(httpapi.XForwardedFor, "10.0.0.1") // Should be ignored
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain the untrusted proxy IP (XFF ignored)
	is.Equal(capturedIP, "192.0.2.200")
}

func TestClientIPFromXFFHeader_InvalidEntriesIgnored(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromXFFHeaderMiddleware(trustedProxy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name              string
		xff               string
		expectedIP        string
		expectedUnchanged bool
	}{
		{
			name:              "invalid IP format",
			xff:               "not.an.ip",
			expectedUnchanged: true, // Should keep original peer IP
		},
		{
			name:              "empty XFF",
			xff:               "",
			expectedUnchanged: true,
		},
		{
			name:              "multiple invalid entries",
			xff:               "invalid1, invalid2",
			expectedUnchanged: true,
		},
		{
			name:       "mixed valid and invalid - uses rightmost valid",
			xff:        "invalid, 192.0.2.100, also-invalid",
			expectedIP: "192.0.2.100",
		},
		{
			name:       "valid IPv4",
			xff:        "192.0.2.100",
			expectedIP: "192.0.2.100",
		},
		{
			name:       "valid IPv6",
			xff:        "2001:db8::1",
			expectedIP: "2001:db8::1",
		},
		{
			name:       "multiple valid IPs - uses rightmost",
			xff:        "192.0.2.100, 192.0.2.200",
			expectedIP: "192.0.2.200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			capturedIP = "" // Reset
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			if tt.xff != "" {
				req.Header.Set(httpapi.XForwardedFor, tt.xff)
			}
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			is.Equal(res.Code, http.StatusOK)

			if tt.expectedUnchanged {
				is.Equal(capturedIP, "127.0.0.1")
			} else {
				is.Equal(capturedIP, tt.expectedIP)
			}
		})
	}
}

func TestClientIPFromXFFHeader_NoHeaderUsesPeerIP(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromXFFHeaderMiddleware(trustedProxy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	// No X-Forwarded-For header
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain peer IP (no XFF header)
	is.Equal(capturedIP, "127.0.0.1")
}

func TestClientIPFromXFFHeader_InvalidPeerIPFallback(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromXFFHeaderMiddleware(trustedProxy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request with invalid RemoteAddr format
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "invalid-address" // Invalid format
	req.Header.Set(httpapi.XForwardedFor, "192.0.2.100")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Should store original RemoteAddr since peer IP is invalid
	is.Equal(capturedIP, "invalid-address")
}

func TestLoginRateLimit_429AfterLimit(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer

	// Create a request body
	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "AdminPass123!",
	})

	// Make 5 requests (the limit) - all should succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.10:12345" // Same IP for all requests
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusOK)
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.10:12345" // Same IP
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusTooManyRequests)

	var errorResp httpapi.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&errorResp)
	is.NoErr(err)
	is.True(errorResp.Error != nil)
	is.Equal(*errorResp.Error, "Too many login attempts. Try again later.")
}

func TestLoginRateLimit_OtherEndpointsUnaffected(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	sessionCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	// Make many requests to a non-login endpoint from the same IP
	// These should not be rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		req.AddCookie(sessionCookie)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusOK)
	}

	// Make many POST requests to a non-login endpoint
	// Use unique names to avoid conflicts
	for i := 0; i < 10; i++ {
		createBody, _ := json.Marshal(map[string]string{"name": "test-device-" + string(rune('a'+i))})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", bytes.NewReader(createBody))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.10:12345"
		req.AddCookie(sessionCookie)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusCreated)
	}
}

func TestLoginRateLimit_DifferentIPsIndependent(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "AdminPass123!",
	})

	// Make 5 requests from IP1 (limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.10:12345"
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusOK)
	}

	// IP1 should be rate limited
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "192.0.2.10:12345"
	res1 := httptest.NewRecorder()
	server.ServeHTTP(res1, req1)
	is.Equal(res1.Code, http.StatusTooManyRequests)

	// IP2 should still be able to make requests
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "192.0.2.20:12345" // Different IP
	res2 := httptest.NewRecorder()
	server.ServeHTTP(res2, req2)
	is.Equal(res2.Code, http.StatusOK)
}
