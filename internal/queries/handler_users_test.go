//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_ListUsersWithAccess_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t).Build(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var rows []httpapi.UserListItem
	is.NoErr(json.NewDecoder(w.Body).Decode(&rows))
	is.Equal(len(rows), 8) // admin (superadmin) + james + noah + maria + liam + sarah + tom + priya

	james := findUserRow(rows, seed.User(testutils.FixtureUserWithAccess.Name).Int64())
	is.True(james != nil)
	is.Equal(james.BypassHostCheck, false)
	is.Equal(james.HostCount, 4)        // GroupMedia(2) + GroupProductivity(2)
	is.Equal(james.DeviceCount, 1)      // FixtureDeviceWithOwnerAccess
	is.Equal(james.LiveAddressCount, 1) // FixtureAddressAlice. Disabled address is not counted
	is.Equal(len(james.Groups), 2)      // GroupMedia + GroupProductivity

	noah := findUserRow(rows, seed.User(testutils.FixtureUserNoAccess.Name).Int64())
	is.True(noah != nil)
	is.Equal(noah.BypassHostCheck, false)
	is.Equal(noah.HostCount, 0)
	is.Equal(noah.DeviceCount, 1)      // FixtureDeviceWithoutOwnerAccess
	is.Equal(noah.LiveAddressCount, 1) // FixtureAddressBob
	is.Equal(len(noah.Groups), 0)

	maria := findUserRow(rows, seed.User(testutils.FixtureUserBypassAccess.Name).Int64())
	is.True(maria != nil)
	is.Equal(maria.BypassHostCheck, true)
	is.Equal(maria.HostCount, 4) // bypass = all 4 hosts
	is.Equal(maria.DeviceCount, 1)
	is.Equal(len(maria.Groups), 1) // GroupMedia only
}

func TestHandler_ListUsersWithAccess_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}

func TestHandler_GetUserAccessDetail_HappyPath(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t).Build(srv)
	jamesID := seed.User(testutils.FixtureUserWithAccess.Name)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/access/users/%d", jamesID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.UserAccessDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.Id, jamesID.Int64())
	is.Equal(resp.Username, testutils.FixtureUserWithAccess.Name)
	is.Equal(resp.BypassHostCheck, false)

	// all 4 groups are returned; only backend and frontend are granted
	is.Equal(len(resp.Groups), 4)

	backend := findGroup(resp.Groups, testutils.GroupMedia.Name)
	is.True(backend != nil)
	is.Equal(backend.Granted, true)
	is.Equal(len(backend.Hosts), 2) // FixtureHostBackend1+2

	frontend := findGroup(resp.Groups, testutils.GroupProductivity.Name)
	is.True(frontend != nil)
	is.Equal(frontend.Granted, true)
	is.Equal(len(frontend.Hosts), 2) // FixtureHostFrontend1+2

	emptyGroup := findGroup(resp.Groups, testutils.GroupInfrastructure.Name)
	is.True(emptyGroup != nil)
	is.Equal(emptyGroup.Granted, false)
	is.Equal(len(emptyGroup.Hosts), 0)

	// james owns 1 device
	is.Equal(len(resp.Devices), 1)
	is.Equal(resp.Devices[0].Name, testutils.FixtureDeviceWithOwnerAccess.Name)
}

func TestHandler_GetUserAccessDetail_NotFound(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users/99999", nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusNotFound)
}

func TestHandler_GetUserAccessDetail_Unauthenticated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/access/users/1", nil)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusUnauthorized)
}
