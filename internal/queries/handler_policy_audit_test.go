//go:build test

package queries_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetPolicyUserMap_Unauthenticated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-map", nil)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.True(rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden)
}

func TestHandler_GetPolicyUserMap_EmptyCache(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-map", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.PolicyUserMapAudit
	err := json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	// Empty cache: admin user should appear as a no-access user.
	is.True(len(response.Users) >= 1)
	is.True(response.RefreshDurationMs >= 0)
	// All users should have empty ips (no cache entries).
	for _, u := range response.Users {
		is.Equal(len(u.Ips), 0)
		is.True(u.LastSeenAt == nil)
	}
}

// TestHandler_GetPolicyUserMap verifies the happy path with a fully
// seeded database: three users with distinct access profiles, two active devices
// at separate IPs, and no shared IP.
func TestHandler_GetPolicyUserMap(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	// Seed groups, hosts, users (alice/bob/charlie), policies, devices,
	// addresses, policy cache, and access log entries.
	testutils.SeedFullWorld(t).Build(testServer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/policy-map", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)
	var response httpapi.PolicyUserMapAudit
	is.NoErr(json.NewDecoder(rec.Body).Decode(&response))

	// ─── top-level aggregates ─────────────────────────────────────────────────
	is.Equal(response.TotalIpCount, 2)     // 10.1.0.1 (alice+charlie shared) + 10.2.0.1 (bob)
	is.Equal(response.TotalDeviceCount, 3) // alice-laptop + bob-phone + charlie-desktop
	is.Equal(response.SharedIpCount, 1)    // 10.1.0.1 is shared by alice and charlie
	// alice contributes api1+api2+web1+web2 via backend+frontend groups; charlie
	// contributes api1+api2 via backend (bypass, but counted before override);
	// union is the same 4 hosts.
	is.Equal(response.TotalHostCount, 4) // FixtureHostBackend1+2 + FixtureHostFrontend1+2

	// ─── alice: backend+frontend access, active device ────────────────────────
	alice := findUser(response.Users, testutils.FixtureUserWithAccess.Name)
	is.True(alice != nil)
	is.Equal(alice.BypassAllowlist, false)
	is.Equal(alice.IsAdmin, false)
	is.Equal(alice.IpCount, 1)
	is.Equal(alice.DeviceCount, 1)
	is.Equal(alice.AllowedHostCount, 4) // FixtureHostBackend1+2 + FixtureHostFrontend1+2
	is.True(alice.LastSeenAt != nil)
	is.Equal(len(alice.Ips), 1)
	is.Equal(alice.Ips[0].Ip, "10.1.0.1")
	is.Equal(len(alice.Ips[0].Addresses), 1)
	is.Equal(alice.Ips[0].Addresses[0].DeviceName, testutils.FixtureDeviceWithOwnerAccess.Name)

	// ─── bob: no group access, active device ──────────────────────────────────
	bob := findUser(response.Users, testutils.FixtureUserNoAccess.Name)
	is.True(bob != nil)
	is.Equal(bob.BypassAllowlist, false)
	is.Equal(bob.IsAdmin, false)
	is.Equal(bob.IpCount, 1)
	is.Equal(bob.DeviceCount, 1)
	is.Equal(bob.AllowedHostCount, 0) // no group memberships
	is.True(bob.LastSeenAt != nil)
	is.Equal(len(bob.Ips), 1)
	is.Equal(bob.Ips[0].Ip, "10.2.0.1")

	// ─── charlie: backend access with bypass, shares alice's IP ─────────────
	charlie := findUser(response.Users, testutils.FixtureUserBypassAccess.Name)
	is.True(charlie != nil)
	is.Equal(charlie.BypassAllowlist, true)
	is.Equal(charlie.IsAdmin, false)
	is.Equal(charlie.IpCount, 1)          // 10.1.0.1 shared with alice
	is.Equal(charlie.DeviceCount, 1)      // charlie-desktop
	is.Equal(charlie.AllowedHostCount, 0) // bypass overrides host count to 0
	is.True(charlie.LastSeenAt != nil)
	is.Equal(len(charlie.Ips), 1)
	is.Equal(charlie.Ips[0].Ip, testutils.FixtureAddressShared.IP)
}

// ── BuildPolicyUserMap integration tests ─────────────────────────────────────
//
// These tests exercise the handler's orchestration method (BuildPolicyUserMap)
// directly — bypassing HTTP — to verify the DB query paths and assembly logic
// that the thin HTTP handler delegates to.

// stubPolicyMapReader is a minimal queries.PolicyMapReader implementation for
// tests that need to control the snapshot without running the full policy service.
type stubPolicyMapReader struct {
	snap policy.PolicyMapSnapshot
}

func (s *stubPolicyMapReader) GetPolicyMap() policy.PolicyMapSnapshot {
	return s.snap
}

// TestBuildPolicyUserMap_NoAccessUser verifies that non-deleted users absent
// from the snapshot appear with empty ips and populated bypass/host fields.
func TestBuildPolicyUserMap_NoAccessUser(t *testing.T) {
	is := is.New(t)

	srv := testutils.SetupIntegrationServer(t)
	repo := queries.NewRepository(srv.Database.DB())

	// The admin user exists from the seed. The snapshot is empty.
	reader := &stubPolicyMapReader{snap: policy.PolicyMapSnapshot{
		LastRefreshedAt:       time.Now().UTC(),
		LastRefreshDurationMs: 0,
	}}

	result, err := repo.BuildPolicyUserMap(t.Context(), reader, nil)
	is.NoErr(err)

	// At least the seeded admin user must appear.
	is.True(len(result.Users) >= 1)
	adminFound := false
	for _, u := range result.Users {
		is.Equal(len(u.Ips), 0)
		is.True(u.LastSeenAt == nil)
		// Required slices must not be nil (JSON must serialize as []).
		is.True(u.Ips != nil)
		is.True(u.UserAllowedHosts != nil)
		if u.IsAdmin {
			adminFound = true
		}
	}
	// The seeded superadmin must be flagged as admin.
	is.True(adminFound)

	// Aggregates must be present (empty snapshot → all zero/empty).
	is.Equal(result.TotalIpCount, 0)
	is.Equal(result.TotalDeviceCount, 0)
	is.Equal(result.SharedIpCount, 0)
}

// findUser returns a pointer to the first PolicyUserEntry whose UserName matches,
// or nil if not found. Mirrors findPolicy/findGroup helpers in the package.
func findUser(users []httpapi.PolicyUserEntry, name string) *httpapi.PolicyUserEntry {
	for i := range users {
		if users[i].DisplayName == name {
			return &users[i]
		}
	}
	return nil
}
