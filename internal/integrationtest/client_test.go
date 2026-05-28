//go:build test

package integrationtest_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
)

// testClient wraps an app.App and a session cookie, providing typed HTTP helpers
// for the endpoints used in cross-domain integration tests. Requests go through
// HTTPServer directly (no real network), exercising the full middleware stack.
type testClient struct {
	t      *testing.T
	srv    *app.App
	cookie *http.Cookie
}

func newAdminClient(t *testing.T, srv *app.App) *testClient {
	t.Helper()
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	return &testClient{t: t, srv: srv, cookie: cookie}
}

func (c *testClient) do(method, path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	var b []byte
	if body != nil {
		var err error
		b, err = json.Marshal(body)
		if err != nil {
			c.t.Fatalf("testClient: marshal body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.AddCookie(c.cookie)
	w := httptest.NewRecorder()
	c.srv.HTTPServer.ServeHTTP(w, req)
	return w
}

func (c *testClient) deleteDevice(deviceID int64) *httptest.ResponseRecorder {
	return c.do(http.MethodDelete, fmt.Sprintf("/api/v1/devices/%d", deviceID), nil)
}

func (c *testClient) regenerateAPIKey(deviceID int64) *httptest.ResponseRecorder {
	return c.do(http.MethodPost, fmt.Sprintf("/api/v1/devices/%d/api-key/regenerate", deviceID), nil)
}

// verifyIP sends a forward-auth request. Uses the policy engine Bearer token,
// not the session cookie. RemoteAddr is pinned to the test trusted proxy so
// X-Real-IP is trusted by the middleware.
func (c *testClient) verifyIP(clientIP, targetHost string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("Authorization", "Bearer test-policy-secret")
	req.Header.Set("X-Real-IP", clientIP)
	if targetHost != "" {
		req.Header.Set("X-Forwarded-Host", targetHost)
	}
	w := httptest.NewRecorder()
	c.srv.HTTPServer.ServeHTTP(w, req)
	return w
}

func decodeJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Fatalf("decodeJSON: %v (body: %s)", err, w.Body.String())
	}
	return v
}
