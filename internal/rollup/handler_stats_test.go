//go:build test

package rollup_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func dashboardStats(t *testing.T, srv http.Handler, cookie *http.Cookie) httpapi.DashboardStats {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard stats: status %d", rec.Code)
	}
	var stats httpapi.DashboardStats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return stats
}

// TestHandler_GetDashboardStats_DenyByReason seeds denials across both mapped
// reasons plus an unmapped one (and a nil reason) at "now", so the default 24h
// window hits the raw path, and asserts the deny_by_reason split reconciles to
// deny_count.
func TestHandler_GetDashboardStats_DenyByReason(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.NewSeeder(t).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "10.0.0.1", Outcome: true}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "10.0.0.2", Outcome: true}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.1", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered)}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.2", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered)}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "9.9.9.3", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered)}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "10.5.0.1", Outcome: false, DenyReason: new(policy.DenyReasonHostNotAllowed)}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "10.5.0.2", Outcome: false, DenyReason: new(policy.DenyReasonHostNotAllowed)}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "10.6.0.1", Outcome: false, DenyReason: new(policy.DenyReasonNoDeviceMatch)}).
		WithAccessLogEntry(testutils.AccessLogEntryFixture{ClientIP: "10.6.0.2", Outcome: false}).
		Build(testServer)

	stats := dashboardStats(t, testServer.HTTPServer, adminCookie)

	is.Equal(stats.AllowCount, int64(2))
	is.Equal(stats.DenyCount, int64(7))
	is.Equal(stats.DenyByReason.IpNotRegistered, int64(3))
	is.Equal(stats.DenyByReason.HostNotAllowed, int64(2))
	is.Equal(stats.DenyByReason.Other, int64(2)) // no_device_match + the nil-reason deny

	// The split partitions deny_count.
	is.Equal(stats.DenyByReason.IpNotRegistered+stats.DenyByReason.HostNotAllowed+stats.DenyByReason.Other, stats.DenyCount)
}
