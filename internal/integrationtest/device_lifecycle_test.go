//go:build test

package integrationtest_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
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
// This test uses SetupRunningIntegrationServer, which starts all background
// services (including policy RunListener) so the event pipeline runs exactly
// as it does in production.
func TestDeviceDelete_EvictsIPFromPolicyCache(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	srv := testutils.SetupRunningIntegrationServer(t)

	seed := testutils.NewSeeder(t, srv).
		WithGroup(testutils.GroupFixture{Name: "backend"}).
		WithHost(testutils.HostFixture{FQDN: "api.internal", Groups: []string{"backend"}}).
		WithUser(testutils.UserFixture{Name: "alice"}).
		SetUserAccess("alice", false, "backend").
		WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
		WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: "10.0.0.1"}).
		WithPolicyInitialize().
		Build()

	deviceID := seed.Device("alice-laptop")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: device address is hot in the policy cache.
	w := verifyIP(t, srv, "10.0.0.1", "api.internal")
	is.Equal(w.Code, http.StatusOK)

	// Give the device an API key so we can verify it is revoked after deletion.
	keyResp, err := client.RegenerateDeviceAPIKeyWithResponse(ctx, deviceID.Int64())
	is.NoErr(err)
	is.Equal(keyResp.StatusCode(), http.StatusOK)
	rawKey := keyResp.JSON200.ApiKey

	// Delete the device via the HTTP API — this is the action under test.
	deleteResp, err := client.DeleteDeviceWithResponse(ctx, deviceID.Int64())
	is.NoErr(err)
	is.Equal(deleteResp.StatusCode(), http.StatusNoContent)

	// The policy cache refresh is async (event → RunListener → refreshCache).
	// 50 ms is consistent with the unit-level lifecycle tests and sufficient
	// for an in-process SQLite refresh.
	time.Sleep(50 * time.Millisecond)

	// Policy cache assertion: the IP must now be denied.
	w = verifyIP(t, srv, "10.0.0.1", "api.internal")
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
