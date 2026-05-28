//go:build test

package integrationtest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestUserDelete_EvictsDeviceIPsFromPolicyCache is a cross-domain integration
// test that verifies the reactive pipeline triggered by user deletion:
//
//  1. A device owned by alice has an active address that is allowed through
//     the policy forward-auth.
//  2. Deleting alice via the HTTP API removes her host-access grants, which
//     triggers an async policy cache refresh.
//  3. The IP is denied after the refresh because the cache rebuilds with an
//     empty host set for alice's device.
//  4. The service layer confirms alice is no longer in the active user list.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestUserDelete_EvictsDeviceIPsFromPolicyCache(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithGroup(testutils.GroupFixture{Name: "backend"}).
			WithHost(testutils.HostFixture{FQDN: "api.internal", Groups: []string{"backend"}}).
			WithUser(testutils.UserFixture{Name: "alice"}).
			SetUserAccess("alice", false, "backend").
			WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: "10.0.0.1"}).
			WithPolicyInitialize(),
	)

	aliceID := seed.User("alice")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: alice's device address is hot in the policy cache.
	w := verifyIP(t, srv, "10.0.0.1", "api.internal")
	is.Equal(w.Code, http.StatusOK)

	// Delete alice via the HTTP API — this is the action under test.
	deleteResp, err := client.DeleteUserWithResponse(ctx, aliceID.Int64())
	is.NoErr(err)
	is.Equal(deleteResp.StatusCode(), http.StatusNoContent)

	// The policy cache refresh is async: DeleteUser → userAccessService removes
	// alice's host-access grants → policy.OnHostAccessChanged → RunListener
	// rebuilds cache with an empty host set for alice's device's IPs.
	time.Sleep(50 * time.Millisecond)

	// Policy cache assertion: the IP must now be denied.
	w = verifyIP(t, srv, "10.0.0.1", "api.internal")
	is.Equal(w.Code, http.StatusForbidden)

	// Service-layer assertion: alice is no longer in the active user list.
	users, err := srv.AuthService.ListUsers(ctx)
	is.NoErr(err)
	for _, u := range users {
		is.True(u.ID != aliceID)
	}
}
