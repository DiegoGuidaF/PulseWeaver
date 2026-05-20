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

	var devices []httpapi.Device
	err := json.NewDecoder(rec.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 0)
}

func TestHandler_GetDevices_AddressCountReflectsEnabledAddresses(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	testutils.NewSeeder(t, testServer).
		WithUser(testutils.UserFixture{Name: "count-user"}).
		WithDevice(testutils.DeviceFixture{Name: "count-device", OwnerUser: "count-user"}).
		WithAddress(testutils.AddressFixture{Device: "count-device", IP: "10.0.4.1", Disabled: true}).
		WithAddress(testutils.AddressFixture{Device: "count-device", IP: "10.0.4.2"}).
		Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var devices []httpapi.Device
	is.NoErr(json.NewDecoder(rec.Body).Decode(&devices))
	is.Equal(len(devices), 1)
	is.True(devices[0].AddressCount != nil)
	is.Equal(*devices[0].AddressCount, 1)
}

// TestHandler_GetDevices_ReturnsWorldDevices verifies that the admin device list
// returns all seeded devices with populated owner names and per-device address counts.
// This exercises the LEFT JOIN with users and addresses in the underlying query.
func TestHandler_GetDevices_ReturnsWorldDevices(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t, testServer).Build()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var devices []httpapi.Device
	is.NoErr(json.NewDecoder(rec.Body).Decode(&devices))
	is.Equal(len(devices), 3) // alice-laptop + bob-phone + charlie-desktop

	aliceLaptop := findDevice(devices, testutils.FixtureDeviceWithOwnerAccess.Name)
	is.True(aliceLaptop != nil)
	is.Equal(aliceLaptop.OwnerId, seed.User(testutils.FixtureUserWithAccess.Name).Int64())
	is.True(aliceLaptop.OwnerName != nil)
	is.Equal(*aliceLaptop.OwnerName, testutils.FixtureUserWithAccess.Name)
	is.True(aliceLaptop.AddressCount != nil)
	is.Equal(*aliceLaptop.AddressCount, 1) // FixtureAddressAlice

	bobPhone := findDevice(devices, testutils.FixtureDeviceWithoutOwnerAccess.Name)
	is.True(bobPhone != nil)
	is.True(bobPhone.AddressCount != nil)
	is.Equal(*bobPhone.AddressCount, 1) // FixtureAddressBob

	charlieDesktop := findDevice(devices, testutils.FixtureDeviceBypassAccess.Name)
	is.True(charlieDesktop != nil)
	is.True(charlieDesktop.AddressCount != nil)
	is.Equal(*charlieDesktop.AddressCount, 1) // FixtureAddressShared
}

// ── GetDevice ────────────────────────────────────────────────────────────────

func TestHandler_GetDevice_ReturnsCorrectDetail(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t, testServer).Build()
	deviceID := seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name) // alice-laptop

	url := fmt.Sprintf("/api/v1/devices/%d", deviceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var d httpapi.Device
	is.NoErr(json.NewDecoder(rec.Body).Decode(&d))
	is.Equal(d.Id, deviceID.Int64())
	is.Equal(d.Name, testutils.FixtureDeviceWithOwnerAccess.Name)
	is.Equal(d.OwnerId, seed.User(testutils.FixtureUserWithAccess.Name).Int64())
	is.True(d.OwnerName != nil)
	is.Equal(*d.OwnerName, testutils.FixtureUserWithAccess.Name)
	is.True(d.AddressCount != nil)
	is.Equal(*d.AddressCount, 1) // FixtureAddressAlice
}

func TestHandler_GetDevice_NotFound(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	url := fmt.Sprintf("/api/v1/devices/%d", 99999)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusNotFound)
}

// ── GetDevicesByUser ─────────────────────────────────────────────────────────

func TestHandler_GetDevicesByUser_Unauthenticated(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/1/devices", nil)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.True(rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden)
}

// TestHandler_GetDevicesByUser_ReturnsUserDevices verifies that the per-user
// device list filters correctly to the requested owner.
func TestHandler_GetDevicesByUser_ReturnsUserDevices(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	seed := testutils.SeedFullWorld(t, testServer).Build()
	aliceID := seed.User(testutils.FixtureUserWithAccess.Name)

	url := fmt.Sprintf("/api/v1/admin/users/%d/devices", aliceID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var devices []httpapi.Device
	is.NoErr(json.NewDecoder(rec.Body).Decode(&devices))
	is.Equal(len(devices), 1) // alice-laptop only

	d := devices[0]
	is.Equal(d.Name, testutils.FixtureDeviceWithOwnerAccess.Name)
	is.Equal(d.OwnerId, aliceID.Int64())
	is.True(d.OwnerName != nil)
	is.Equal(*d.OwnerName, testutils.FixtureUserWithAccess.Name)
	is.True(d.AddressCount != nil)
	is.Equal(*d.AddressCount, 1) // FixtureAddressAlice
}

// findDevice returns a pointer to the first Device whose Name matches, or nil.
func findDevice(devices []httpapi.Device, name string) *httpapi.Device {
	for i := range devices {
		if devices[i].Name == name {
			return &devices[i]
		}
	}
	return nil
}
