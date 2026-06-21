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
	is.Equal(len(rows), 8) // admin (superadmin) + alice + bob + charlie + diana + erin + grace + frank

	alice := findUserRow(rows, seed.User(testutils.FixtureUserWithAccess.Name).Int64())
	is.True(alice != nil)
	is.Equal(alice.BypassHostCheck, false)
	is.Equal(alice.HostCount, 4)        // GroupMedia(2) + GroupProductivity(2)
	is.Equal(alice.DeviceCount, 1)      // FixtureDeviceWithOwnerAccess
	is.Equal(alice.LiveAddressCount, 1) // FixtureAddressAlice. Disabled address is not counted
	is.Equal(len(alice.Groups), 2)      // GroupMedia + GroupProductivity

	bob := findUserRow(rows, seed.User(testutils.FixtureUserNoAccess.Name).Int64())
	is.True(bob != nil)
	is.Equal(bob.BypassHostCheck, false)
	is.Equal(bob.HostCount, 0)
	is.Equal(bob.DeviceCount, 1)      // FixtureDeviceWithoutOwnerAccess
	is.Equal(bob.LiveAddressCount, 1) // FixtureAddressBob
	is.Equal(len(bob.Groups), 0)

	charlie := findUserRow(rows, seed.User(testutils.FixtureUserBypassAccess.Name).Int64())
	is.True(charlie != nil)
	is.Equal(charlie.BypassHostCheck, true)
	is.Equal(charlie.HostCount, 4) // bypass = all 4 hosts
	is.Equal(charlie.DeviceCount, 1)
	is.Equal(len(charlie.Groups), 1) // GroupMedia only
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
	aliceID := seed.User(testutils.FixtureUserWithAccess.Name)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/admin/access/users/%d", aliceID), nil)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, req)

	is.Equal(w.Code, http.StatusOK)
	var resp httpapi.UserAccessDetail
	is.NoErr(json.NewDecoder(w.Body).Decode(&resp))
	is.Equal(resp.Id, aliceID.Int64())
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

	// alice owns 1 device
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
