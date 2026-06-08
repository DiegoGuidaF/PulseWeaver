//go:build test

package httpserver_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpserver"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

var testLogger = slog.New(slog.DiscardHandler)

func TestClientIPFromRequest_ExtractsFromRemoteAddr(t *testing.T) {
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

	// Request with X-Real-IP header (should be ignored)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(httpapi.XRealIP, "192.0.2.100")
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

func TestClientIPFromRealIP_TrustedProxyExtractsClientIP(t *testing.T) {
	is := is.New(t)

	// Setup trusted proxy prefix
	trustedProxy := netip.MustParseAddr("127.0.0.1")

	// Create a handler that captures client IP from context
	var capturedIP string
	handler := httpserver.ClientIPFromRealIPMiddleware(trustedProxy, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request from trusted proxy with X-Real-IP
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(httpapi.XRealIP, "192.0.2.100")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain the client IP from X-Real-IP
	is.Equal(capturedIP, "192.0.2.100")
}

func TestClientIPFromRealIP_MappedV4HeaderCanonicalized(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromRealIPMiddleware(trustedProxy, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// X-Real-IP arrives as an IPv4-mapped IPv6 address; the context must hold the
	// canonical (unmapped) plain IPv4 form so policy lookups are representation-stable.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set(httpapi.XRealIP, "::ffff:192.0.2.100")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	is.Equal(capturedIP, "192.0.2.100")
}

func TestClientIPFromRealIP_UntrustedProxyIgnoresHeader(t *testing.T) {
	is := is.New(t)

	// Setup trusted proxy prefix (only 127.0.0.1)
	trustedProxy := netip.MustParseAddr("127.0.0.1")

	// Create a handler that captures client IP from context
	var capturedIP string
	handler := httpserver.ClientIPFromRealIPMiddleware(trustedProxy, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request from untrusted IP with X-Real-IP
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.0.2.200:12345"        // Not the trusted proxy
	req.Header.Set(httpapi.XRealIP, "10.0.0.1") // Should be ignored
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain the untrusted peer IP (X-Real-IP ignored)
	is.Equal(capturedIP, "192.0.2.200")
}

func TestClientIPFromRealIP_InvalidHeaderFallsBackToPeer(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromRealIPMiddleware(trustedProxy, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		xRealIP    string
		expectedIP string
	}{
		{
			name:       "invalid IP format",
			xRealIP:    "not.an.ip",
			expectedIP: "127.0.0.1",
		},
		{
			name:       "empty header",
			xRealIP:    "",
			expectedIP: "127.0.0.1",
		},
		{
			name:       "valid IPv4",
			xRealIP:    "192.0.2.100",
			expectedIP: "192.0.2.100",
		},
		{
			name:       "valid IPv6",
			xRealIP:    "2001:db8::1",
			expectedIP: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			capturedIP = "" // Reset
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			if tt.xRealIP != "" {
				req.Header.Set(httpapi.XRealIP, tt.xRealIP)
			}
			res := httptest.NewRecorder()

			handler.ServeHTTP(res, req)

			is.Equal(res.Code, http.StatusOK)

			is.Equal(capturedIP, tt.expectedIP)
		})
	}
}

func TestClientIPFromRealIP_NoHeaderUsesPeerIP(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromRealIPMiddleware(trustedProxy, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	// No X-Real-IP header
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Context should contain peer IP (no X-Real-IP header)
	is.Equal(capturedIP, "127.0.0.1")
}

func TestClientIPFromRealIP_InvalidPeerIPFallback(t *testing.T) {
	is := is.New(t)

	trustedProxy := netip.MustParseAddr("127.0.0.1")

	var capturedIP string
	handler := httpserver.ClientIPFromRealIPMiddleware(trustedProxy, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, ok := httpapi.ClientIPFromContext(r.Context())
		if ok {
			capturedIP = ip
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Request with invalid RemoteAddr format
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "invalid-address" // Invalid format
	req.Header.Set(httpapi.XRealIP, "192.0.2.100")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	// Should store original RemoteAddr since peer IP is invalid
	is.Equal(capturedIP, "invalid-address")
}

func TestHeartbeatRateLimit_429AfterLimit(t *testing.T) {
	is := is.New(t)

	limit := 2
	handler := httpserver.HeartbeatRateLimitMiddleware(limit, time.Minute, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusOK)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusTooManyRequests)

	var errorResp httpapi.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&errorResp)
	is.NoErr(err)
	is.True(errorResp.Error != nil)
	is.Equal(*errorResp.Error, "Too many heartbeat requests. Try again later.")
}

func TestHeartbeatRateLimit_OnlyAffectsHeartbeatEndpoint(t *testing.T) {
	is := is.New(t)

	limit := 2
	called := 0
	handler := httpserver.HeartbeatRateLimitMiddleware(limit, time.Minute, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the heartbeat budget
	for i := 0; i < limit+1; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
	}

	calledAfterHeartbeat := called

	// Requests to a different path must never be rate limited
	for i := 0; i < limit+5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/other", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusOK)
	}

	_ = calledAfterHeartbeat
}

func TestHeartbeatRateLimit_DifferentIPsIndependent(t *testing.T) {
	is := is.New(t)

	limit := 2
	handler := httpserver.HeartbeatRateLimitMiddleware(limit, time.Minute, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the budget for IP1
	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
		req.RemoteAddr = "192.0.2.10:12345"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusOK)
	}

	// IP1 is now rate limited
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
	req1.RemoteAddr = "192.0.2.10:12345"
	res1 := httptest.NewRecorder()
	handler.ServeHTTP(res1, req1)
	is.Equal(res1.Code, http.StatusTooManyRequests)

	// IP2 still has its own budget
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat", nil)
	req2.RemoteAddr = "192.0.2.20:12345"
	res2 := httptest.NewRecorder()
	handler.ServeHTTP(res2, req2)
	is.Equal(res2.Code, http.StatusOK)
}

func TestVerifyIPRateLimit_429AfterLimit(t *testing.T) {
	is := is.New(t)

	limit := 3
	handler := httpserver.VerifyIPRateLimitMiddleware(limit, time.Minute, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden) // mimic forward-auth fail-closed
	}))

	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodGet, httpapi.VerifyIPEndpoint, nil)
		req.RemoteAddr = "192.0.2.10:12345"
		req.Header.Set("Authorization", "Bearer badtoken123")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusForbidden)
	}

	// The next request from the same IP must be throttled with 429.
	req := httptest.NewRequest(http.MethodGet, httpapi.VerifyIPEndpoint, nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("Authorization", "Bearer badtoken123")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusTooManyRequests)

	var errorResp httpapi.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&errorResp)
	is.NoErr(err)
	is.True(errorResp.Error != nil)
	is.Equal(*errorResp.Error, "Too many verification requests. Try again later.")
}

func TestVerifyIPRateLimit_DifferentIPsIndependent(t *testing.T) {
	is := is.New(t)

	limit := 2
	handler := httpserver.VerifyIPRateLimitMiddleware(limit, time.Minute, testLogger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))

	for i := 0; i < limit; i++ {
		req := httptest.NewRequest(http.MethodGet, httpapi.VerifyIPEndpoint, nil)
		req.RemoteAddr = "192.0.2.10:12345"
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		is.Equal(res.Code, http.StatusForbidden)
	}

	// IP1 is now throttled.
	req1 := httptest.NewRequest(http.MethodGet, httpapi.VerifyIPEndpoint, nil)
	req1.RemoteAddr = "192.0.2.10:12345"
	res1 := httptest.NewRecorder()
	handler.ServeHTTP(res1, req1)
	is.Equal(res1.Code, http.StatusTooManyRequests)

	// IP2 retains its own budget.
	req2 := httptest.NewRequest(http.MethodGet, httpapi.VerifyIPEndpoint, nil)
	req2.RemoteAddr = "192.0.2.20:12345"
	res2 := httptest.NewRecorder()
	handler.ServeHTTP(res2, req2)
	is.Equal(res2.Code, http.StatusForbidden)
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
