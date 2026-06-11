//go:build test

package integrationtest_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
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

	const (
		deviceIP    = "10.0.0.1"
		backendHost = "api.internal"
	)

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithGroup(testutils.GroupFixture{Name: "backend"}).
			WithHost(testutils.HostFixture{FQDN: backendHost, Groups: []string{"backend"}}).
			WithUser(testutils.UserFixture{Name: "alice"}).
			SetUserAccess("alice", false, "backend").
			WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: deviceIP}).
			WithPolicyInitialize(),
	)

	aliceID := seed.User("alice")
	deviceID := seed.Device("alice-laptop")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: alice's device address is hot in the policy cache.
	w := verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusOK)

	// Delete alice via the HTTP API — this is the action under test.
	before := srv.PolicyService.LastRefreshedAt()
	deleteResp, err := client.DeleteUserWithResponse(ctx, aliceID.Int64())
	is.NoErr(err)
	is.Equal(deleteResp.StatusCode(), http.StatusNoContent)

	// The policy cache refresh is async: DeleteUser → userAccessService removes
	// alice's host-access grants → policy.OnHostAccessChanged → RunListener
	// rebuilds cache with an empty host set for alice's device's IPs.
	testutils.WaitForPolicyRefresh(ctx, t, srv, before)

	// Policy cache assertion: the IP must now be denied.
	w = verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusForbidden)

	// Service-layer assertion: alice is no longer in the active user list.
	_, err = srv.AuthService.GetUserFromPrincipal(ctx, new(auth.Principal{UserID: aliceID}))
	errors.Is(err, auth.ErrUserNotFound)

	// Service-layer assertions: alice's device must be soft-deleted and its
	// addresses disabled — deleting a user should cascade to owned devices.
	_, err = srv.DeviceService.GetDevice(ctx, deviceID)
	is.True(errors.Is(err, device.ErrDeviceNotFound))

	addrs, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(addrs), 0)
}
