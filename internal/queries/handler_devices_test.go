//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDeviceAddresses_EmptyArray(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, err := testServer.DeviceService.CreateDevice(t.Context(), testutils.AdminPrincipal(t, testServer), "empty-addresses-device", nil)
	is.NoErr(err)

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", dev.ID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var addresses []httpapi.Address
	err = json.NewDecoder(rec.Body).Decode(&addresses)
	is.NoErr(err)
	is.Equal(len(addresses), 0)
}

// TestHandler_GetDeviceAddresses_ReturnsDeviceAddresses verifies the happy path
// for listing a device's addresses: correct IP, ownership, enabled status, and
// no expiry when no lease is registered.
func TestHandler_GetDeviceAddresses_ReturnsDeviceAddresses(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t).Build(testServer)
	deviceID := seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name) // james-laptop

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", deviceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var addresses []httpapi.Address
	is.NoErr(json.NewDecoder(rec.Body).Decode(&addresses))
	is.Equal(len(addresses), 2) // FixtureAddressAlice

	for _, address := range addresses {
		if address.Id == seed.Address(testutils.FixtureAddressAlice.Device, testutils.FixtureAddressAlice.IP).Int64() {
			is.Equal(address.Ip, testutils.FixtureAddressAlice.IP)
			is.True(address.IsEnabled)
			is.Equal(address.DeviceId, deviceID.Int64())
			is.True(address.Id != 0)
			is.True(!time.Time(address.CreatedAt).IsZero())
			is.True(address.ExpiresAt == nil) // no lease seeded
		}
	}

}

func TestHandler_GetDeviceAddresses_ExpiresAtPopulatedWithLease(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	futureExpiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "lease-user"}).
		WithDevice(testutils.DeviceFixture{Name: "lease-device", OwnerUser: "lease-user"}).
		WithAddress(testutils.AddressFixture{Device: "lease-device", IP: "10.0.2.1", ExpiresAt: &futureExpiry}).
		Build(testServer)

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", seed.Device("lease-device"))
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var addresses []httpapi.Address
	is.NoErr(json.NewDecoder(rec.Body).Decode(&addresses))
	is.Equal(len(addresses), 1)
	is.True(addresses[0].ExpiresAt != nil)
	is.True(time.Time(*addresses[0].ExpiresAt).UTC().Truncate(time.Second).Equal(futureExpiry))
}

// TestHandler_GetDeviceAddresses_SourceFieldPopulated verifies that the source field
// is present and non-empty in address responses.
func TestHandler_GetDeviceAddresses_SourceFieldPopulated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "source-user"}).
		WithDevice(testutils.DeviceFixture{Name: "source-device", OwnerUser: "source-user"}).
		WithAddress(testutils.AddressFixture{Device: "source-device", IP: "10.0.9.1"}).
		Build(testServer)

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", seed.Device("source-device"))
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var addresses []httpapi.Address
	is.NoErr(json.NewDecoder(rec.Body).Decode(&addresses))
	is.Equal(len(addresses), 1)
	is.True(string(addresses[0].Source) != "") // source must not be the zero value
}

func TestHandler_GetDeviceAddresses_DeviceNotFound(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", 99999)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusNotFound)
}

func TestHandler_GetDeviceAddresses_ExpiresAtNullWhenNoLease(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "no-lease-user"}).
		WithDevice(testutils.DeviceFixture{Name: "no-lease-device", OwnerUser: "no-lease-user"}).
		WithAddress(testutils.AddressFixture{Device: "no-lease-device", IP: "10.0.3.1"}).
		Build(testServer)

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", seed.Device("no-lease-device"))
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var addresses []httpapi.Address
	is.NoErr(json.NewDecoder(rec.Body).Decode(&addresses))
	is.Equal(len(addresses), 1)
	is.True(addresses[0].ExpiresAt == nil)
}

func TestHandler_GetDevices_EmptyArray(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var groups []httpapi.DeviceOwnerGroup
	err := json.NewDecoder(rec.Body).Decode(&groups)
	is.NoErr(err)
	// The endpoint now returns all users; only the bootstrap admin exists with no devices.
	is.Equal(len(groups), 1)
	is.Equal(len(groups[0].Devices), 0)
}

// TestHandler_GetDevices_GroupsDevicesByOwner is the primary happy-path test for the
// owner-grouped device list. Verifies owner metadata (host groups, bypass flag, counts),
// per-device state derivation, live-address count, and rule summaries.
func TestHandler_GetDevices_GroupsDevicesByOwner(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.SeedFullWorld(t).Build(testServer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var groups []httpapi.DeviceOwnerGroup
	is.NoErr(json.NewDecoder(rec.Body).Decode(&groups))
	is.Equal(len(groups), 8) // james + noah + maria + liam + sarah + tom + priya + bootstrap admin (superadmin)

	// ── james ───────────────────────────────────────────────────────────────────
	james := findOwnerGroup(groups, testutils.FixtureUserWithAccess.Name)
	is.True(james != nil)
	is.Equal(james.Owner.BypassHostCheck, false)
	is.Equal(len(james.Owner.HostGroups), 2) // backend + frontend
	is.Equal(james.Owner.DeviceCount, 1)
	is.Equal(james.Owner.LiveAddressCount, 1) // FixtureAddressAlice

	jamesLaptop := findDeviceEntry(james.Devices, testutils.FixtureDeviceWithOwnerAccess.Name)
	is.True(jamesLaptop != nil)
	is.Equal(jamesLaptop.LiveAddressCount, 1)
	is.Equal(string(jamesLaptop.State), string(httpapi.Healthy))
	is.True(jamesLaptop.Pairing == nil)

	// james-laptop has lease (1h) and max-active (2) rules from SeedFullWorld
	is.Equal(len(jamesLaptop.Rules), 2) // FixtureLeaseRuleAliceLaptop + FixtureMaxActiveRuleAliceLaptop
	leaseRule := findRule(jamesLaptop.Rules, httpapi.AutoExpiry)
	is.True(leaseRule != nil)
	is.True(leaseRule.Enabled)
	is.True(leaseRule.TtlSeconds != nil)
	is.Equal(*leaseRule.TtlSeconds, testutils.FixtureLeaseRuleAliceLaptop.TTLSeconds) // 3600s
	maxRule := findRule(jamesLaptop.Rules, httpapi.MaxActive)
	is.True(maxRule != nil)
	is.True(maxRule.Enabled)
	is.True(maxRule.Limit != nil)
	is.Equal(*maxRule.Limit, testutils.FixtureMaxActiveRuleAliceLaptop.MaxAddresses) // 2

	// ── noah ─────────────────────────────────────────────────────────────────────
	noah := findOwnerGroup(groups, testutils.FixtureUserNoAccess.Name)
	is.True(noah != nil)
	is.Equal(noah.Owner.BypassHostCheck, false)
	is.Equal(len(noah.Owner.HostGroups), 0) // no groups assigned
	is.Equal(noah.Owner.DeviceCount, 1)
	is.Equal(noah.Owner.LiveAddressCount, 1) // FixtureAddressBob

	noahPhone := findDeviceEntry(noah.Devices, testutils.FixtureDeviceWithoutOwnerAccess.Name)
	is.True(noahPhone != nil)
	is.Equal(noahPhone.LiveAddressCount, 1)
	is.Equal(string(noahPhone.State), string(httpapi.Healthy))
	is.Equal(len(noahPhone.Rules), 0) // no rules seeded

	// ── maria ──────────────────────────────────────────────────────────────────
	maria := findOwnerGroup(groups, testutils.FixtureUserBypassAccess.Name)
	is.True(maria != nil)
	is.Equal(maria.Owner.BypassHostCheck, true)
	// host groups are still returned even for bypass users; frontend decides rendering
	is.Equal(len(maria.Owner.HostGroups), 1) // backend
	is.Equal(maria.Owner.DeviceCount, 1)
	is.Equal(maria.Owner.LiveAddressCount, 1) // FixtureAddressShared

	mariaDesktop := findDeviceEntry(maria.Devices, testutils.FixtureDeviceBypassAccess.Name)
	is.True(mariaDesktop != nil)
	is.Equal(mariaDesktop.LiveAddressCount, 1)
	is.Equal(string(mariaDesktop.State), string(httpapi.Healthy))
}

// TestHandler_GetDevices_StaleState verifies that a device with no live addresses
// is reported as stale.
func TestHandler_GetDevices_StaleState(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "stale-user"}).
		WithDevice(testutils.DeviceFixture{Name: "stale-device", OwnerUser: "stale-user"}).
		WithAddress(testutils.AddressFixture{Device: "stale-device", IP: "10.0.5.1", Disabled: true}).
		Build(testServer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var groups []httpapi.DeviceOwnerGroup
	is.NoErr(json.NewDecoder(rec.Body).Decode(&groups))
	// Two users exist: bootstrap admin (no devices) and stale-user.
	is.Equal(len(groups), 2)

	staleGroup := findOwnerGroup(groups, "stale-user")
	is.True(staleGroup != nil)
	d := findDeviceEntry(staleGroup.Devices, "stale-device")
	is.True(d != nil)
	is.Equal(d.LiveAddressCount, 0)
	is.Equal(string(d.State), string(httpapi.Stale))
}

// TestHandler_GetDevices_PairingSummary verifies that the last_pairing field on
// DeviceListEntry reflects the correct derived status for each possible pairing case
// seeded by SeedFullWorld.
func TestHandler_GetDevices_PairingSummary(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.SeedFullWorld(t).Build(testServer)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)

	var groups []httpapi.DeviceOwnerGroup
	is.NoErr(json.NewDecoder(rec.Body).Decode(&groups))

	james := findOwnerGroup(groups, testutils.FixtureUserWithAccess.Name)
	jamesLaptop := findDeviceEntry(james.Devices, testutils.FixtureDeviceWithOwnerAccess.Name)
	is.True(jamesLaptop.Pairing == nil) // no pairing seeded for james-laptop

	noah := findOwnerGroup(groups, testutils.FixtureUserNoAccess.Name)
	noahPhone := findDeviceEntry(noah.Devices, testutils.FixtureDeviceWithoutOwnerAccess.Name)
	is.True(noahPhone.Pairing != nil)
	is.Equal(string(noahPhone.Pairing.Status), "pending")
	is.True(time.Time(noahPhone.Pairing.ExpiresAt).After(time.Now()))

	maria := findOwnerGroup(groups, testutils.FixtureUserBypassAccess.Name)
	mariaDesktop := findDeviceEntry(maria.Devices, testutils.FixtureDeviceBypassAccess.Name)
	is.True(mariaDesktop.Pairing != nil)
	is.Equal(string(mariaDesktop.Pairing.Status), "invalidated")

	liam := findOwnerGroup(groups, testutils.FixtureUserPairing.Name)
	is.True(liam != nil)

	liamUsed := findDeviceEntry(liam.Devices, testutils.FixtureDevicePairingUsed.Name)
	is.True(liamUsed.Pairing != nil)
	is.Equal(string(liamUsed.Pairing.Status), "used")

	liamExpired := findDeviceEntry(liam.Devices, testutils.FixtureDevicePairingExpired.Name)
	is.True(liamExpired.Pairing != nil)
	is.Equal(string(liamExpired.Pairing.Status), "expired")
	is.True(time.Time(liamExpired.Pairing.ExpiresAt).Before(time.Now()))
}

// findOwnerGroup returns the DeviceOwnerGroup whose owner username matches, or nil.
func findOwnerGroup(groups []httpapi.DeviceOwnerGroup, username string) *httpapi.DeviceOwnerGroup {
	for i := range groups {
		if groups[i].Owner.Username == username {
			return &groups[i]
		}
	}
	return nil
}

// findDeviceEntry returns the DeviceListEntry with the given name, or nil.
func findDeviceEntry(devices []httpapi.DeviceListEntry, name string) *httpapi.DeviceListEntry {
	for i := range devices {
		if devices[i].Name == name {
			return &devices[i]
		}
	}
	return nil
}

// findRule returns the DeviceRuleSummary with the given type, or nil.
func findRule(rules []httpapi.DeviceRuleSummary, ruleType httpapi.DeviceRuleSummaryType) *httpapi.DeviceRuleSummary {
	for i := range rules {
		if rules[i].Type == ruleType {
			return &rules[i]
		}
	}
	return nil
}
