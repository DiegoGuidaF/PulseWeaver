//go:build test

package integrationtest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

var wantSecurityHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
	"Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; font-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'",
	"Referrer-Policy":    "strict-origin-when-cross-origin",
	"Permissions-Policy": "camera=(), microphone=(), geolocation=(), interest-cohort=()",
}

// TestSecurityHeaders_PresentOnEveryResponseType asserts the full security header
// set, each with exactly one value, on every class of response the server emits.
// The SPA handler's own responses are covered in internal/ui under the prod tag.
func TestSecurityHeaders_PresentOnEveryResponseType(t *testing.T) {
	srv := testutils.SetupIntegrationServer(t)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{"public endpoint", http.MethodGet, "/health", http.StatusOK},
		{"API unauthorized", http.MethodGet, "/api/v1/version", http.StatusUnauthorized},
		{"API unknown route", http.MethodGet, "/api/v1/nope", http.StatusNotFound},
		{"forward-auth endpoint", http.MethodGet, "/api/policy-engine/verify-ip", http.StatusForbidden},
		{"non-API fallthrough", http.MethodGet, "/some/spa/route", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:1"
			w := httptest.NewRecorder()
			srv.HTTPServer.ServeHTTP(w, req)

			is.Equal(w.Code, tt.wantStatus)
			for name, want := range wantSecurityHeaders {
				is.Equal(w.Result().Header.Values(name), []string{want})
			}
		})
	}
}
