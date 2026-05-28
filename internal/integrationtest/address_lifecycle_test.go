//go:build test

package integrationtest_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestAddressDisable_EvictsIPFromPolicyCache is a cross-domain integration test
// that verifies disabling an address removes it from the policy cache:
//
//  1. A device with an active address is allowed through the policy forward-auth.
//  2. Disabling the address via the HTTP API fires an AddressDisabled event that
//     triggers an async policy cache refresh.
//  3. The IP is denied after the refresh.
//  4. The service layer confirms no enabled addresses remain for the device.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestAddressDisable_EvictsIPFromPolicyCache(t *testing.T) {
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
	addressID := seed.Address("alice-laptop", deviceIP)
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: address is hot in the policy cache.
	w := verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusOK)

	// Disable the address via the HTTP API — this is the action under test.
	before := srv.PolicyService.LastRefreshedAt()
	disableResp, err := client.DisableAddressWithResponse(ctx, deviceID.Int64(), addressID.Int64())
	is.NoErr(err)
	is.Equal(disableResp.StatusCode(), http.StatusOK)

	// The policy cache refresh is async (AddressDisabled event → RunListener → refreshCache).
	testutils.WaitForPolicyRefresh(ctx, t, srv, before)

	// Policy cache assertion: the IP must now be denied.
	w = verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusForbidden)

	// Service-layer assertion: no enabled addresses remain for the device.
	addrs, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(addrs), 0)
}

// TestAddressEnable_AddsIPToPolicyCache is a cross-domain integration test that
// verifies adding a new address makes it accessible through the policy forward-auth:
//
//  1. A device with no registered address is denied at the policy forward-auth.
//  2. Adding an address via the HTTP API fires an AddressCreated event that
//     triggers an async policy cache refresh.
//  3. The IP is allowed after the refresh.
//  4. The service layer confirms the enabled address exists for the device.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestAddressEnable_AddsIPToPolicyCache(t *testing.T) {
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
			WithPolicyInitialize(),
	)

	deviceID := seed.Device("alice-laptop")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: IP is unknown to the cache (no address registered).
	w := verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusForbidden)

	// Add the address via the HTTP API — this is the action under test.
	before := srv.PolicyService.LastRefreshedAt()
	addResp, err := client.AddAddressWithResponse(ctx, deviceID.Int64(), httpapi.AddAddressJSONRequestBody{
		Ip: deviceIP,
	})
	is.NoErr(err)
	is.True(addResp.StatusCode() == http.StatusCreated || addResp.StatusCode() == http.StatusOK)

	// The policy cache refresh is async (AddressCreated event → RunListener → refreshCache).
	testutils.WaitForPolicyRefresh(ctx, t, srv, before)

	// Policy cache assertion: the IP must now be allowed.
	w = verifyIP(t, srv, deviceIP, backendHost)
	is.Equal(w.Code, http.StatusOK)

	// Service-layer assertion: the address is enabled for the device.
	addrs, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(addrs), 1)
	is.Equal(addrs[0].IP, deviceIP)
}
