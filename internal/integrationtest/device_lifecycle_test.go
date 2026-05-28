//go:build test

package integrationtest_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestDeviceDelete_EvictsIPFromPolicyCache is a cross-domain integration test
// that verifies the full reactive pipeline:
//
//  1. A device with an active address is allowed through the policy forward-auth.
//  2. Deleting the device via the HTTP API fires AddressDisabled events that
//     trigger an async policy cache refresh.
//  3. The IP is denied after the refresh, and service-layer state reflects the
//     full cleanup: device soft-deleted, addresses disabled, API key revoked.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestDeviceDelete_EvictsIPFromPolicyCache(t *testing.T) {
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

	deviceID := seed.Device("alice-laptop")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: device address is hot in the policy cache.
	w := verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusOK)

	// Give the device an API key so we can verify it is revoked after deletion.
	keyResp, err := client.RegenerateDeviceAPIKeyWithResponse(ctx, deviceID.Int64())
	is.NoErr(err)
	is.Equal(keyResp.StatusCode(), http.StatusOK)
	rawKey := keyResp.JSON200.ApiKey

	// Delete the device via the HTTP API — this is the action under test.
	before := srv.PolicyService.LastRefreshedAt()
	deleteResp, err := client.DeleteDeviceWithResponse(ctx, deviceID.Int64())
	is.NoErr(err)
	is.Equal(deleteResp.StatusCode(), http.StatusNoContent)

	// The policy cache refresh is async (event → RunListener → refreshCache).
	testutils.WaitForPolicyRefresh(ctx, t, srv, before)

	// Policy cache assertion: the IP must now be denied.
	w = verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusForbidden)

	// Service-layer assertions — verify the full cleanup cascade.

	// Device is soft-deleted: GetDevice filters WHERE deleted_at IS NULL.
	_, err = srv.DeviceService.GetDevice(ctx, deviceID)
	is.True(errors.Is(err, device.ErrDeviceNotFound))

	// All addresses were disabled by the delete transaction.
	addrs, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(addrs), 0)

	// API key was revoked: Authenticate hashes the raw key and looks it up;
	// with the key row deleted, it returns ErrDeviceNotFound.
	_, err = srv.DeviceService.Authenticate(ctx, rawKey)
	is.True(errors.Is(err, device.ErrDeviceNotFound))
}

// TestDeviceOwnershipChange_RefreshesHostAccessInPolicyCache is a cross-domain
// integration test that verifies ownership reassignment correctly swaps which
// hosts the device's IP may reach:
//
//  1. alice-laptop (owned by alice, backend access) has IP 10.0.0.1, which is
//     allowed for api.internal (backend) and denied for web.internal (frontend).
//  2. Reassigning ownership to bob (frontend access only) via the HTTP API fires
//     an EventTypeDeviceOwnershipChanged event that triggers an async cache refresh.
//  3. After the refresh the IP is denied for api.internal and allowed for web.internal.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestDeviceOwnershipChange_RefreshesHostAccessInPolicyCache(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	const (
		deviceIP     = "10.0.0.1"
		backendHost  = "api.internal"
		frontendHost = "web.internal"
	)

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithGroup(testutils.GroupFixture{Name: "backend"}).
			WithGroup(testutils.GroupFixture{Name: "frontend"}).
			WithHost(testutils.HostFixture{FQDN: backendHost, Groups: []string{"backend"}}).
			WithHost(testutils.HostFixture{FQDN: frontendHost, Groups: []string{"frontend"}}).
			WithUser(testutils.UserFixture{Name: "alice"}).
			WithUser(testutils.UserFixture{Name: "bob"}).
			SetUserAccess("alice", false, "backend").
			SetUserAccess("bob", false, "frontend").
			WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: deviceIP}).
			WithPolicyInitialize(),
	)

	deviceID := seed.Device("alice-laptop")
	bobID := seed.User("bob")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: IP is routed by alice's grants — backend allowed, frontend denied.
	w := verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusOK)
	w = verifyIP(t, srv, deviceIP, frontendHost)
	is.Equal(w.Code, http.StatusForbidden)

	// Reassign ownership to bob via the HTTP API — this is the action under test.
	before := srv.PolicyService.LastRefreshedAt()
	updateResp, err := client.UpdateDeviceWithResponse(ctx, deviceID.Int64(), httpapi.UpdateDeviceJSONRequestBody{
		OwnerId: new(int(bobID.Int64())),
	})
	is.NoErr(err)
	is.Equal(updateResp.StatusCode(), http.StatusOK)

	// The policy cache refresh is async (EventTypeDeviceOwnershipChanged →
	// policy.OnAddressEvent → triggerRefresh → RunListener → refreshCache).
	testutils.WaitForPolicyRefresh(ctx, t, srv, before)

	// Policy cache assertion: access is now determined by bob's grants.
	w = verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusForbidden)
	w = verifyIP(t, srv, deviceIP, frontendHost)
	is.Equal(w.Code, http.StatusOK)

	// Service-layer assertion: device owner reflects the update.
	dev, err := srv.DeviceService.GetDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(dev.OwnerID, bobID)
}
