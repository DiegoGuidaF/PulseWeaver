//go:build test

package dashboard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDashboardAttributionSplit_Unauthenticated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attribution-split?kind=policy", nil)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.True(rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden)
}

// TestHandler_GetDashboardAttributionSplit_MissingKind: kind is required, so the
// OpenAPI validator rejects a request without it before the handler runs.
func TestHandler_GetDashboardAttributionSplit_MissingKind(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attribution-split", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusBadRequest)
}

func attributionSplit(t *testing.T, srv http.Handler, cookie *http.Cookie, kind string) map[string]httpapi.DashboardAttributionCount {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/attribution-split?kind="+kind, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("attribution-split kind=%s: status %d", kind, rec.Code)
	}
	var resp httpapi.DashboardAttributionSplitResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byName := make(map[string]httpapi.DashboardAttributionCount, len(resp.Entities))
	for _, e := range resp.Entities {
		byName[e.EntityName] = e
	}
	return byName
}

// TestHandler_GetDashboardAttributionSplit seeds the full world (fixtures are
// written at "now", so the default 24h window hits the raw path) and asserts the
// per-entity allow/deny split for each kind. The shared-IP fixture
// (FixtureAccessLogSharedIPAllow) lists two devices of two different users, so it
// counts once for each.
func TestHandler_GetDashboardAttributionSplit(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.SeedFullWorld(t).
		// A second request against the group-policy, denied, so a policy shows a mix.
		WithAccessLogEntry(testutils.AccessLogEntryFixture{
			ClientIP:   "10.3.0.9",
			Outcome:    false,
			PolicyName: testutils.FixturePolicyWithGroups.Name,
			TargetHost: new(testutils.FixtureHostBackend1.FQDN),
		}).
		Build(testServer)

	// ── policy ──
	policies := attributionSplit(t, testServer.HTTPServer, adminCookie, "policy")
	withGroups := policies[testutils.FixturePolicyWithGroups.Name]
	is.Equal(withGroups.AllowCount, int64(1)) // FixtureAccessLogNetworkPolicyAllow
	is.Equal(withGroups.DenyCount, int64(1))  // the extra deny seeded above
	is.True(withGroups.EntityId != nil)
	bypass := policies[testutils.FixturePolicyBypassHostCheck.Name]
	is.Equal(bypass.AllowCount, int64(1)) // FixtureAccessLogBypassAllow
	is.Equal(bypass.DenyCount, int64(0))

	// ── user ── (display_name == fixture Name)
	users := attributionSplit(t, testServer.HTTPServer, adminCookie, "user")
	// alice: AliceAllow + SharedIPAllow, two distinct requests.
	is.Equal(users[testutils.FixtureUserWithAccess.Name].AllowCount, int64(2))
	is.Equal(users[testutils.FixtureUserWithAccess.Name].DenyCount, int64(0))
	// charlie: only the shared-IP request.
	is.Equal(users[testutils.FixtureUserBypassAccess.Name].AllowCount, int64(1))
	// bob: a single host-denied request.
	is.Equal(users[testutils.FixtureUserNoAccess.Name].DenyCount, int64(1))

	// ── device ──
	devices := attributionSplit(t, testServer.HTTPServer, adminCookie, "device")
	// alice-laptop appears in both AliceAllow and the shared-IP request.
	is.Equal(devices[testutils.FixtureDeviceWithOwnerAccess.Name].AllowCount, int64(2))
	is.Equal(devices[testutils.FixtureDeviceBypassAccess.Name].AllowCount, int64(1))
	is.Equal(devices[testutils.FixtureDeviceWithoutOwnerAccess.Name].DenyCount, int64(1))
}
