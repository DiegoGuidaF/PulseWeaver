//go:build test

package integrationtest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
)

// verifyIP sends a forward-auth check for clientIP against targetHost.
// This endpoint is not in the OpenAPI spec so it is exercised manually.
// RemoteAddr is pinned to the trusted proxy so X-Real-IP is trusted by middleware.
func verifyIP(t *testing.T, srv *app.App, clientIP, targetHost string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("Authorization", "Bearer "+testutils.TestPolicySecret)
	req.Header.Set("X-Real-IP", clientIP)
	if targetHost != "" {
		req.Header.Set("X-Forwarded-Host", targetHost)
	}
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)
	return w
}
