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

	seed := testutils.SeedFullWorld(t, testServer).Build()
	deviceID := seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name) // alice-laptop

	url := fmt.Sprintf("/api/v1/devices/%d/addresses", deviceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var addresses []httpapi.Address
	is.NoErr(json.NewDecoder(rec.Body).Decode(&addresses))
	is.Equal(len(addresses), 1) // FixtureAddressAlice

	got := addresses[0]
	is.Equal(got.Ip, testutils.FixtureAddressAlice.IP)
	is.True(got.IsEnabled)
	is.Equal(got.DeviceId, deviceID.Int64())
	is.True(got.Id != 0)
	is.True(!time.Time(got.CreatedAt).IsZero())
	is.True(got.ExpiresAt == nil) // no lease seeded
}

func TestHandler_GetDeviceAddresses_ExpiresAtPopulatedWithLease(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	futureExpiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	seed := testutils.NewSeeder(t, testServer).
		WithUser(testutils.UserFixture{Name: "lease-user"}).
		WithDevice(testutils.DeviceFixture{Name: "lease-device", OwnerUser: "lease-user"}).
		WithAddress(testutils.AddressFixture{Device: "lease-device", IP: "10.0.2.1", ExpiresAt: &futureExpiry}).
		Build()

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

	seed := testutils.NewSeeder(t, testServer).
		WithUser(testutils.UserFixture{Name: "source-user"}).
		WithDevice(testutils.DeviceFixture{Name: "source-device", OwnerUser: "source-user"}).
		WithAddress(testutils.AddressFixture{Device: "source-device", IP: "10.0.9.1"}).
		Build()

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

	seed := testutils.NewSeeder(t, testServer).
		WithUser(testutils.UserFixture{Name: "no-lease-user"}).
		WithDevice(testutils.DeviceFixture{Name: "no-lease-device", OwnerUser: "no-lease-user"}).
		WithAddress(testutils.AddressFixture{Device: "no-lease-device", IP: "10.0.3.1"}).
		Build()

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
	is.Equal(len(groups), 0)
}

// TestHandler_GetDevices_GroupsDevicesByOwner is the primary happy-path test for the
// owner-grouped device list. Verifies owner metadata (host groups, bypass flag, counts),
// per-device state derivation, live-address count, and rule summaries.
func TestHandler_GetDevices_GroupsDevicesByOwner(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.SeedFullWorld(t, testServer).Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var groups []httpapi.DeviceOwnerGroup
	is.NoErr(json.NewDecoder(rec.Body).Decode(&groups))
	is.Equal(len(groups), 3) // alice + bob + charlie

	// ── alice ───────────────────────────────────────────────────────────────────
	alice := findOwnerGroup(groups, testutils.FixtureUserWithAccess.Name)
	is.True(alice != nil)
	is.Equal(alice.Owner.BypassHostCheck, false)
	is.Equal(len(alice.Owner.HostGroups), 2) // backend + frontend
	is.Equal(alice.Owner.DeviceCount, 1)
	is.Equal(alice.Owner.LiveAddressCount, 1) // FixtureAddressAlice

	aliceLaptop := findDeviceEntry(alice.Devices, testutils.FixtureDeviceWithOwnerAccess.Name)
	is.True(aliceLaptop != nil)
	is.Equal(aliceLaptop.LiveAddressCount, 1)
	is.Equal(string(aliceLaptop.State), string(httpapi.Healthy))
	is.True(aliceLaptop.Pairing == nil)

	// alice-laptop has lease (1h) and max-active (2) rules from SeedFullWorld
	is.Equal(len(aliceLaptop.Rules), 2) // FixtureLeaseRuleAliceLaptop + FixtureMaxActiveRuleAliceLaptop
	leaseRule := findRule(aliceLaptop.Rules, httpapi.AutoExpiry)
	is.True(leaseRule != nil)
	is.True(leaseRule.Enabled)
	is.True(leaseRule.TtlSeconds != nil)
	is.Equal(*leaseRule.TtlSeconds, testutils.FixtureLeaseRuleAliceLaptop.TTLSeconds) // 3600s
	maxRule := findRule(aliceLaptop.Rules, httpapi.MaxActive)
	is.True(maxRule != nil)
	is.True(maxRule.Enabled)
	is.True(maxRule.Limit != nil)
	is.Equal(*maxRule.Limit, testutils.FixtureMaxActiveRuleAliceLaptop.MaxAddresses) // 2

	// ── bob ─────────────────────────────────────────────────────────────────────
	bob := findOwnerGroup(groups, testutils.FixtureUserNoAccess.Name)
	is.True(bob != nil)
	is.Equal(bob.Owner.BypassHostCheck, false)
	is.Equal(len(bob.Owner.HostGroups), 0) // no groups assigned
	is.Equal(bob.Owner.DeviceCount, 1)
	is.Equal(bob.Owner.LiveAddressCount, 1) // FixtureAddressBob

	bobPhone := findDeviceEntry(bob.Devices, testutils.FixtureDeviceWithoutOwnerAccess.Name)
	is.True(bobPhone != nil)
	is.Equal(bobPhone.LiveAddressCount, 1)
	is.Equal(string(bobPhone.State), string(httpapi.Healthy))
	is.Equal(len(bobPhone.Rules), 0) // no rules seeded

	// ── charlie ──────────────────────────────────────────────────────────────────
	charlie := findOwnerGroup(groups, testutils.FixtureUserBypassAccess.Name)
	is.True(charlie != nil)
	is.Equal(charlie.Owner.BypassHostCheck, true)
	// host groups are still returned even for bypass users; frontend decides rendering
	is.Equal(len(charlie.Owner.HostGroups), 1) // backend
	is.Equal(charlie.Owner.DeviceCount, 1)
	is.Equal(charlie.Owner.LiveAddressCount, 1) // FixtureAddressShared

	charlieDesktop := findDeviceEntry(charlie.Devices, testutils.FixtureDeviceBypassAccess.Name)
	is.True(charlieDesktop != nil)
	is.Equal(charlieDesktop.LiveAddressCount, 1)
	is.Equal(string(charlieDesktop.State), string(httpapi.Healthy))
}

// TestHandler_GetDevices_StaleState verifies that a device with no live addresses
// is reported as stale.
func TestHandler_GetDevices_StaleState(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.NewSeeder(t, testServer).
		WithUser(testutils.UserFixture{Name: "stale-user"}).
		WithDevice(testutils.DeviceFixture{Name: "stale-device", OwnerUser: "stale-user"}).
		WithAddress(testutils.AddressFixture{Device: "stale-device", IP: "10.0.5.1", Disabled: true}).
		Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var groups []httpapi.DeviceOwnerGroup
	is.NoErr(json.NewDecoder(rec.Body).Decode(&groups))
	is.Equal(len(groups), 1)

	d := findDeviceEntry(groups[0].Devices, "stale-device")
	is.True(d != nil)
	is.Equal(d.LiveAddressCount, 0)
	is.Equal(string(d.State), string(httpapi.Stale))
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
