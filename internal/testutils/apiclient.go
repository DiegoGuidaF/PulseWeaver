//go:build test

package testutils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
)

// handlerTransport is an http.RoundTripper that dispatches requests directly
// to an http.Handler without opening a real network connection. RemoteAddr is
// always set to the test trusted-proxy address so X-Real-IP headers are
// trusted by the middleware.
type handlerTransport struct {
	h http.Handler
}

func (t *handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.RemoteAddr = "127.0.0.1:0"
	// Strip scheme/host so the handler sees a path-only request, matching what
	// httptest.NewRequest produces. The OpenAPI validator panics on full URLs
	// when the spec defines a relative server URL (/api/v1).
	req.URL = &url.URL{Path: req.URL.Path, RawQuery: req.URL.RawQuery}
	req.Host = "example.com"
	// http.NewRequest sets Body=nil for bodyless requests; httptest.NewRequest
	// sets Body=http.NoBody. The slog-chi body-capture middleware panics when
	// it wraps a nil body and the OpenAPI validator reads it.
	if req.Body == nil {
		req.Body = http.NoBody
	}
	w := httptest.NewRecorder()
	t.h.ServeHTTP(w, req)
	return w.Result(), nil
}

// NewAPIClient returns a ClientWithResponses that dispatches requests directly
// to srv.HTTPServer, bypassing the network. Additional options (e.g.
// httpapi.WithRequestEditorFn for auth) can be appended.
func NewAPIClient(t *testing.T, srv *app.App, opts ...httpapi.ClientOption) *httpapi.ClientWithResponses {
	t.Helper()
	transport := &handlerTransport{h: srv.HTTPServer}
	allOpts := append(
		[]httpapi.ClientOption{httpapi.WithHTTPClient(&http.Client{Transport: transport})},
		opts...,
	)
	// Routes are mounted under /api/v1 in the production router.
	client, err := httpapi.NewClientWithResponses("http://localhost/api/v1", allOpts...)
	if err != nil {
		t.Fatalf("NewAPIClient: %v", err)
	}
	return client
}

// NewAdminAPIClient returns a ClientWithResponses authenticated as the
// bootstrap admin user via a session cookie.
func NewAdminAPIClient(t *testing.T, srv *app.App) *httpapi.ClientWithResponses {
	t.Helper()
	cookie := LoginCookie(t, srv.HTTPServer, "admin", TestAdminPassword)
	return NewAPIClient(t, srv, httpapi.WithRequestEditorFn(
		func(_ context.Context, req *http.Request) error {
			req.AddCookie(cookie)
			return nil
		},
	))
}
