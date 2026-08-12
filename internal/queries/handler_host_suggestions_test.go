//go:build test

package queries_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestHandler_ListHostSuggestions_HappyPath covers all three filter branches:
// unknown host (→ suggestion), known host (→ excluded), ignored host (→ ignored list).
func TestHandler_ListHostSuggestions_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	knownHost := "known-app.internal"
	suggestedHost := "new-app.internal"
	ignoredHost := "ignored-app.internal"

	testutils.NewSeeder(t).
		WithHost(testutils.HostFixture{FQDN: knownHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: true, TargetHost: &knownHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &ignoredHost}).
		Build(srv)

	_, err := srv.HostsService.AddIgnoredSuggestion(t.Context(), ignoredHost)
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostSuggestionsPage
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))

	// only new-app.internal appears; known and ignored are filtered out
	is.Equal(len(resp.Suggestions), 1)
	is.Equal(resp.Suggestions[0].Fqdn, suggestedHost)
	is.Equal(resp.Suggestions[0].DeniedHits, 1)
	is.Equal(resp.Suggestions[0].AllowedHits, 0)

	is.Equal(len(resp.Ignored), 1)
	is.Equal(resp.Ignored[0].Fqdn, ignoredHost)
}

// TestHandler_ListHostSuggestions_PortNormalised proves that a host observed with
// a port suffix (the shape produced when a proxy is fronted on a non-default port)
// surfaces as a suggestion for its bare FQDN, aggregates with bare-host hits, and
// is excluded once the bare host is granted.
func TestHandler_ListHostSuggestions_PortNormalised(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	knownHost := "known-app.internal"
	knownHostPort := "known-app.internal:8443"
	suggestedHost := "new-app.internal"
	suggestedHostPort := "new-app.internal:8443"

	testutils.NewSeeder(t).
		WithHost(testutils.HostFixture{FQDN: knownHost}).
		// granted host seen with a port suffix must still be excluded from suggestions
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: true, TargetHost: &knownHostPort}).
		// same unknown host seen bare and with a port → one suggestion, hits summed
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHost}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.9", Outcome: false, TargetHost: &suggestedHostPort}).
		Build(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostSuggestionsPage
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))

	is.Equal(len(resp.Suggestions), 1)
	is.Equal(resp.Suggestions[0].Fqdn, suggestedHost) // bare FQDN, port stripped
	is.Equal(resp.Suggestions[0].DeniedHits, 2)       // bare + port-suffixed hits merged
	is.Equal(resp.Suggestions[0].AllowedHits, 0)
}

func TestHandler_ListHostSuggestions_Empty(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.HostSuggestionsPage
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(len(resp.Suggestions), 0)
	is.Equal(len(resp.Ignored), 0)
}

// TestHandler_ListHostSuggestions_AggregateBackedWindow proves the real
// endpoint's fixed 7-day window — always wider than rollup.RawWindowThreshold
// — answers correctly end-to-end: a host observed several days ago is rolled
// up and has its raw rows pruned (so it can only come from
// hourly_traffic_aggregates), while a host observed moments ago is merged in
// from the still-in-flight hour of access_log.
func TestHandler_ListHostSuggestions_AggregateBackedWindow(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)
	ctx := t.Context()

	accessLogRepo := accesslog.NewRepository(srv.Database.DB(), nil)
	rollupRepo := rollup.NewRepository(srv.Database.DB(), nil)

	oldHost := "old-suggestion.internal"
	freshHost := "fresh-suggestion.internal"
	currentHourStart := time.Now().UTC().Truncate(time.Hour)
	oldHour := currentHourStart.Add(-3 * 24 * time.Hour)

	is.NoErr(accessLogRepo.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "5.5.5.5", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &oldHost, CreatedAt: oldHour.Add(5 * time.Minute), Headers: map[string][]string{}},
		{ClientIP: "5.5.5.5", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &oldHost, CreatedAt: oldHour.Add(10 * time.Minute), Headers: map[string][]string{}},
	}))
	is.NoErr(rollupRepo.RunRollup(ctx, oldHour.Add(-time.Hour), currentHourStart))
	// Prune the raw rows rollup just covered — the old host can now only
	// surface via hourly_traffic_aggregates.
	_, err := accessLogRepo.DeleteOlderThan(ctx, currentHourStart)
	is.NoErr(err)

	is.NoErr(accessLogRepo.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "4.4.4.4", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &freshHost, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}))

	resp, err := client.ListHostSuggestionsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	old := findSuggestion(resp.JSON200.Suggestions, oldHost)
	is.True(old != nil) // answered from hourly_traffic_aggregates; raw rows were pruned
	is.Equal(old.DeniedHits, 2)

	fresh := findSuggestion(resp.JSON200.Suggestions, freshHost)
	is.True(fresh != nil) // merged in from the not-yet-rolled current hour
	is.Equal(fresh.DeniedHits, 1)
}

func findSuggestion(suggestions []httpapi.HostSuggestion, fqdn string) *httpapi.HostSuggestion {
	for i := range suggestions {
		if suggestions[i].Fqdn == fqdn {
			return &suggestions[i]
		}
	}
	return nil
}

func TestHandler_ListHostSuggestions_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/host-suggestions", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}
