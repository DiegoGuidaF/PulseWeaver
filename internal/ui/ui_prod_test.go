//go:build prod && test

package ui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpserver"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ui"
	"github.com/matryer/is"
)

// TestHandler_SecurityHeadersSingleValued checks that the embedded SPA, wrapped by
// the global middleware, carries each framing/sniffing header exactly once for the
// index, a static asset and the client-side-routing fallback.
func TestHandler_SecurityHeadersSingleValued(t *testing.T) {
	h := httpserver.SecurityHeadersMiddleware()(ui.Handler())

	for _, path := range []string{"/", "/favicon.svg", "/devices/42"} {
		t.Run(path, func(t *testing.T) {
			is := is.New(t)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

			is.Equal(w.Code, http.StatusOK)
			hdr := w.Result().Header
			is.Equal(hdr.Values("X-Frame-Options"), []string{"DENY"})
			is.Equal(hdr.Values("X-Content-Type-Options"), []string{"nosniff"})
			is.Equal(len(hdr.Values("Content-Security-Policy")), 1)
		})
	}
}
