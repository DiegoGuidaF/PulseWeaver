//go:build test

package audit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/audit"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDenyReasons_Empty(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/request-audit-log/deny-reasons", nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var reasons []string
	err := json.NewDecoder(res.Body).Decode(&reasons)
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}

func TestHandler_GetDenyReasons_WithData(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", testutils.TestAdminPassword)

	repo := audit.NewRepository(testServer.Database.DB())
	r1 := policy.DenyReasonIPNotRegistered
	r2 := policy.DenyReasonNoDeviceMatch
	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // duplicate — deduplicated
		{ClientIP: "3.3.3.3", Outcome: false, DenyReason: &r2, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "4.4.4.4", Outcome: true, DenyReason: nil, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // allow — excluded
	}
	err := repo.BatchInsert(t.Context(), events)
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/request-audit-log/deny-reasons", nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var reasons []string
	err = json.NewDecoder(res.Body).Decode(&reasons)
	is.NoErr(err)
	is.Equal(len(reasons), 2)
	is.Equal(reasons[0], string(r1))
	is.Equal(reasons[1], string(r2))
}
