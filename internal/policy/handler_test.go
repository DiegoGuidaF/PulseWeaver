//go:build test

package policy_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// testMockProvider is a local EnabledIPsProvider for handler tests.
type testMockProvider struct {
	entries []device.IPEntry
	err     error
}

func (m *testMockProvider) GetEnabledIPEntries(_ context.Context) ([]device.IPEntry, error) {
	return m.entries, m.err
}

// testBypassAllHostProvider grants bypass access to all users.
type testBypassAllHostProvider struct{}

func (b *testBypassAllHostProvider) GetAllUserHostAccess(_ context.Context) ([]policy.UserHostAccess, error) {
	return []policy.UserHostAccess{{UserID: 0, BypassAllowlist: true}}, nil
}

// testFakeObserver records every DecisionEvent it receives.
type testFakeObserver struct {
	mu     sync.Mutex
	events []policy.DecisionEvent
}

func (f *testFakeObserver) OnDecision(_ context.Context, e policy.DecisionEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func (f *testFakeObserver) received() []policy.DecisionEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]policy.DecisionEvent, len(f.events))
	copy(out, f.events)
	return out
}

func TestHandler_EarlyData_Returns425(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Early-Data", "1")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusTooEarly)
}

func TestHandler_SimulatePolicyAccess_DeviceAllow(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	resp, err := h.SimulatePolicyAccess(context.Background(), httpapi.SimulatePolicyAccessRequestObject{
		Params: httpapi.SimulatePolicyAccessParams{Ip: "1.2.3.4", Host: "example.com"},
	})
	is.NoErr(err)
	result, ok := resp.(httpapi.SimulatePolicyAccess200JSONResponse)
	is.True(ok)
	is.True(result.Allowed)
}

func TestHandler_MissingAuthHeader_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_WrongToken_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer wrongtoken")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestNewService_EmptySecret_ReturnsError(t *testing.T) {
	is := is.New(t)
	provider := &testMockProvider{entries: []device.IPEntry{{IP: "1.2.3.4", DeviceID: ids.DeviceID(1), AddressID: ids.AddressID(1)}}}

	_, err := policy.NewService(provider, &testBypassAllHostProvider{}, &geoip.Lookup{}, nil, "", slog.New(slog.DiscardHandler), netip.Addr{})
	is.True(errors.Is(err, policy.ErrSecretNotConfigured))
}

func TestHandler_MissingClientIPInContext_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_InvalidIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "not-an-ip"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_AllowedIP_Returns200(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)
}

func TestHandler_DisabledIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandler([]string{"1.2.3.4"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "9.9.9.9"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_IPv6Normalisation(t *testing.T) {
	is := is.New(t)
	// "::1" is the normalized form; the cache should store the normalized form
	h := newTestHandler([]string{"::1"})
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "::1"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)
}

func TestHandler_ProxyIP_Returns403(t *testing.T) {
	is := is.New(t)
	h := newTestHandlerWithProxy([]string{"127.0.0.1"}, "mysecret", "127.0.0.1")
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "127.0.0.1"))
	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)
}

func TestHandler_EnrichmentHeaders_PassedToVerifyAccess(t *testing.T) {
	is := is.New(t)
	obs := &testFakeObserver{}
	h := newTestHandlerWithObserver([]string{"1.2.3.4"}, obs)

	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r.Header.Set("User-Agent", "TestAgent/1.0")
	r.Header.Set("X-Forwarded-Host", "myhost.example.com")
	r.Header.Set("X-Forwarded-Uri", "/protected")
	r.Header.Set("X-Forwarded-Method", "GET")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))

	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusOK)

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.Equal(e.ClientIP, "1.2.3.4")
	is.True(e.Outcome)
	is.True(e.TargetHost != nil)
	is.Equal(*e.TargetHost, "myhost.example.com")
	is.True(e.TargetURI != nil)
	is.Equal(*e.TargetURI, "/protected")
	is.True(len(e.Headers["User-Agent"]) > 0)
	is.Equal(e.Headers["User-Agent"][0], "TestAgent/1.0")
	is.Equal(len(e.Headers["Authorization"]), 0)
}

func TestHandler_DenyEvent_StoresMinimalHeaders(t *testing.T) {
	is := is.New(t)
	obs := &testFakeObserver{}
	// 9.9.9.9 is not registered, so the decision is a deny.
	h := newTestHandlerWithObserver([]string{"1.2.3.4"}, obs)

	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "Bearer mysecret")
	r.Header.Set("User-Agent", "TestAgent/1.0")
	r.Header.Set("Accept-Encoding", "gzip")
	r.Header.Set("X-Forwarded-Host", "myhost.example.com")
	r.Header.Set("X-Forwarded-Uri", "/protected")
	r.Header.Set("X-Forwarded-Method", "GET")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "9.9.9.9"))

	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)

	events := obs.received()
	is.Equal(len(events), 1)
	e := events[0]
	is.True(!e.Outcome)
	// Forwarding context is retained.
	is.Equal(e.Headers["X-Forwarded-Host"][0], "myhost.example.com")
	is.Equal(e.Headers["X-Forwarded-Method"][0], "GET")
	// Arbitrary client headers are not stored on a deny.
	is.Equal(len(e.Headers["User-Agent"]), 0)
	is.Equal(len(e.Headers["Accept-Encoding"]), 0)
	// Sensitive headers are never stored.
	is.Equal(len(e.Headers["Authorization"]), 0)
}

// A malformed Authorization header (no "Bearer " prefix) hits the reject branch
// that once logged the whole header map — including the credential it carries and
// any Cookie. The log must never echo those values.
func TestHandler_MalformedAuth_DoesNotLogCredentials(t *testing.T) {
	is := is.New(t)

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := newTestHandlerWithLogger([]string{"1.2.3.4"}, logger)

	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.Header.Set("Authorization", "malformed-secret-value") // no "Bearer " prefix
	r.Header.Set("Cookie", "session=topsecret")
	r = r.WithContext(httpapi.WithClientIP(r.Context(), "1.2.3.4"))

	w := httptest.NewRecorder()
	h.HandleForwardAuthIP(w, r)
	is.Equal(w.Code, http.StatusForbidden)

	logged := buf.String()
	is.True(!strings.Contains(logged, "malformed-secret-value")) // credential must not leak
	is.True(!strings.Contains(logged, "topsecret"))              // cookie must not leak
}

// newTestHandler creates an HTTPHandler pre-populated with the given IPs in its cache.
func newTestHandler(enabledIPs []string) *policy.HTTPHandler {
	return newTestHandlerWithProxy(enabledIPs, "mysecret", "")
}

// newTestHandlerWithLogger builds a handler whose own logger is the supplied one,
// so tests can assert on what the handler writes to its log sink.
func newTestHandlerWithLogger(enabledIPs []string, logger *slog.Logger) *policy.HTTPHandler {
	entries := make([]device.IPEntry, len(enabledIPs))
	for i, ip := range enabledIPs {
		entries[i] = device.IPEntry{IP: ip, DeviceID: ids.DeviceID(int64(i + 1)), AddressID: ids.AddressID(int64(i + 1))}
	}
	provider := &testMockProvider{entries: entries}
	svc, err := policy.NewService(provider, &testBypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", slog.New(slog.DiscardHandler), netip.Addr{})
	if err != nil {
		panic(err)
	}
	_ = svc.Initialize(context.Background())
	return policy.NewHTTPHandler(svc, logger)
}

func newTestHandlerWithProxy(enabledIPs []string, secret, trustedProxy string) *policy.HTTPHandler {
	entries := make([]device.IPEntry, len(enabledIPs))
	for i, ip := range enabledIPs {
		entries[i] = device.IPEntry{IP: ip, DeviceID: ids.DeviceID(int64(i + 1)), AddressID: ids.AddressID(int64(i + 1))}
	}
	provider := &testMockProvider{entries: entries}
	var proxyAddr netip.Addr
	if trustedProxy != "" {
		proxyAddr = netip.MustParseAddr(trustedProxy)
	}
	svc, err := policy.NewService(provider, &testBypassAllHostProvider{}, &geoip.Lookup{}, nil, secret, slog.New(slog.DiscardHandler), proxyAddr)
	if err != nil {
		panic(err)
	}
	_ = svc.Initialize(context.Background())
	return policy.NewHTTPHandler(svc, slog.New(slog.DiscardHandler))
}

func newTestHandlerWithObserver(enabledIPs []string, obs policy.DecisionObserver) *policy.HTTPHandler {
	entries := make([]device.IPEntry, len(enabledIPs))
	for i, ip := range enabledIPs {
		entries[i] = device.IPEntry{IP: ip, DeviceID: ids.DeviceID(int64(i + 1)), AddressID: ids.AddressID(int64(i + 1))}
	}
	provider := &testMockProvider{entries: entries}
	svc, err := policy.NewService(provider, &testBypassAllHostProvider{}, &geoip.Lookup{}, nil, "mysecret", slog.New(slog.DiscardHandler), netip.Addr{})
	if err != nil {
		panic(err)
	}
	if obs != nil {
		svc.AddDecisionObserver(obs)
	}
	_ = svc.Initialize(context.Background())
	return policy.NewHTTPHandler(svc, slog.New(slog.DiscardHandler))
}
