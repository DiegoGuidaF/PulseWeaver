//go:build test

package queries_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDashboardPosture_Unauthenticated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/posture", nil)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.True(rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden)
}

// TestHandler_GetDashboardPosture seeds one user per status bucket plus enabled
// network policies and an observed unknown host, then asserts the posture
// histogram and top-level counts.
func TestHandler_GetDashboardPosture(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "grants"}).
		WithHost(testutils.HostFixture{FQDN: "grant.example.com", Groups: []string{"grants"}}).
		// bypass: reaches everything regardless of IPs or grants.
		WithUser(testutils.UserFixture{Name: "bypass-user"}).
		SetUserAccess("bypass-user", true).
		// live_with_access: live IP in the cache + a host grant.
		WithUser(testutils.UserFixture{Name: "live-with"}).
		SetUserAccess("live-with", false, "grants").
		WithDevice(testutils.DeviceFixture{Name: "live-with-laptop", OwnerUser: "live-with"}).
		WithAddress(testutils.AddressFixture{Device: "live-with-laptop", IP: "10.1.0.1"}).
		// live_no_host_access: live IP but no host grants — a true lockout.
		WithUser(testutils.UserFixture{Name: "live-no-host"}).
		WithDevice(testutils.DeviceFixture{Name: "live-no-host-laptop", OwnerUser: "live-no-host"}).
		WithAddress(testutils.AddressFixture{Device: "live-no-host-laptop", IP: "10.2.0.1"}).
		// no_live_ips: host grants but no live IP (never ran the client).
		WithUser(testutils.UserFixture{Name: "no-live"}).
		SetUserAccess("no-live", false, "grants").
		// no_access: neither live IPs nor host grants.
		WithUser(testutils.UserFixture{Name: "no-access"}).
		// Two enabled network policies, one bypassing the host check.
		WithPolicy(testutils.PolicyFixture{Name: "np-normal", CIDR: "10.50.0.0/16"}).
		WithPolicy(testutils.PolicyFixture{Name: "np-bypass", CIDR: "10.60.0.0/16"}).
		WithPolicyBypassHostCheck("np-bypass").
		// An unknown host receiving traffic → one pending suggestion.
		WithObservedHost("unknown.example.com", 2).
		WithPolicyInitialize().
		Build(testServer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/posture", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)
	var resp httpapi.DashboardPosture
	is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))

	// ─── user status histogram ────────────────────────────────────────────────
	is.Equal(resp.Users.Bypass, 1)           // bypass-user
	is.Equal(resp.Users.LiveWithAccess, 1)   // live-with
	is.Equal(resp.Users.LiveNoHostAccess, 1) // live-no-host
	is.Equal(resp.Users.NoLiveIps, 1)        // no-live
	is.Equal(resp.Users.NoAccess, 2)         // no-access + bootstrap admin

	// ─── network policies ─────────────────────────────────────────────────────
	is.Equal(resp.NetworkPolicies.Enabled, 2)         // np-normal + np-bypass
	is.Equal(resp.NetworkPolicies.BypassHostCheck, 1) // np-bypass

	// ─── top-level counts ─────────────────────────────────────────────────────
	is.Equal(resp.SharedIpCount, 0)          // distinct IPs, none shared
	is.Equal(resp.KnownHostCount, 1)         // grant.example.com (via live-with + no-live)
	is.Equal(resp.PendingSuggestionCount, 1) // unknown.example.com

	is.True(!time.Time(resp.RefreshedAt).IsZero())
}
