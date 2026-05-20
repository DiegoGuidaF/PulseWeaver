//go:build test

package testutils_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestSeedFullWorld_AllEntitiesCreated(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	seed := testutils.SeedFullWorld(t, srv).Build()

	// Groups
	is.True(seed.Group(testutils.FixtureGroupEmpty.Name) != 0)
	is.True(seed.Group(testutils.FixtureGroupBackend.Name) != 0)
	is.True(seed.Group(testutils.FixtureGroupFrontend.Name) != 0)

	// Hosts
	is.True(seed.Host(testutils.FixtureHostBackend1.FQDN) != 0)
	is.True(seed.Host(testutils.FixtureHostBackend2.FQDN) != 0)
	is.True(seed.Host(testutils.FixtureHostFrontend1.FQDN) != 0)
	is.True(seed.Host(testutils.FixtureHostFrontend2.FQDN) != 0)

	// Users
	is.True(seed.User(testutils.FixtureUserWithAccess.Name) != 0)
	is.True(seed.User(testutils.FixtureUserNoAccess.Name) != 0)
	is.True(seed.User(testutils.FixtureUserBypassAccess.Name) != 0)

	// Policies
	is.True(seed.Policy(testutils.FixturePolicyWithGroups.Name) != 0)
	is.True(seed.Policy(testutils.FixturePolicyNoGroups.Name) != 0)

	// Devices
	is.True(seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name) != 0)
	is.True(seed.Device(testutils.FixtureDeviceWithoutOwnerAccess.Name) != 0)
	is.True(seed.Device(testutils.FixtureDeviceBypassAccess.Name) != 0)

	// Addresses (including shared IP)
	is.True(seed.Address(testutils.FixtureAddressAlice.Device, testutils.FixtureAddressAlice.IP) != 0)
	is.True(seed.Address(testutils.FixtureAddressBob.Device, testutils.FixtureAddressBob.IP) != 0)
	is.True(seed.Address(testutils.FixtureAddressShared.Device, testutils.FixtureAddressShared.IP) != 0)

	// Access log rows: 5 entries seeded
	// access_log_contributors: 4 rows (alice allow:1, bob deny:1, shared allow:2)
	// access_log_network_policy_contributors: 1 row (network policy allow)
	var logCount, contribCount, policyContribCount int
	is.NoErr(srv.Database.DB().QueryRowxContext(t.Context(), `SELECT COUNT(*) FROM access_log`).Scan(&logCount))
	is.NoErr(srv.Database.DB().QueryRowxContext(t.Context(), `SELECT COUNT(*) FROM access_log_contributors`).Scan(&contribCount))
	is.NoErr(srv.Database.DB().QueryRowxContext(t.Context(), `SELECT COUNT(*) FROM access_log_network_policy_contributors`).Scan(&policyContribCount))
	is.Equal(logCount, 5)
	is.Equal(contribCount, 4)
	is.Equal(policyContribCount, 1)

	// IDs are distinct (no silent collision)
	is.True(seed.Group(testutils.FixtureGroupBackend.Name) != seed.Group(testutils.FixtureGroupFrontend.Name))
	is.True(seed.User(testutils.FixtureUserWithAccess.Name) != seed.User(testutils.FixtureUserNoAccess.Name))
	is.True(seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name) != seed.Device(testutils.FixtureDeviceWithoutOwnerAccess.Name))
}
