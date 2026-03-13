//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDeviceAddresses_EmptyArray(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "empty-addresses-device")
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

func TestHandler_GetDeviceAddresses_CorrectFields(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "fields-device")
	is.NoErr(err)

	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.1.1", device.EventSourceManual)
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
	is.Equal(len(addresses), 1)

	got := addresses[0]
	is.Equal(got.Ip, "10.0.1.1")
	is.True(got.IsEnabled)
	is.Equal(got.DeviceId, int64(dev.ID))
	is.True(got.Id != 0)
	is.True(!time.Time(got.CreatedAt).IsZero())
	is.True(!time.Time(got.UpdatedAt).IsZero())
}

func TestHandler_GetDeviceAddresses_ExpiresAtPopulatedWithLease(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "lease-handler-device")
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.2.1", device.EventSourceManual)
	is.NoErr(err)

	// Insert a lease directly via the lease repository sharing the same DB.
	leaseRepo := lease.NewRepository(testServer.Database.DB())
	futureExpiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	addressLease := &lease.AddressLease{
		AddressID: addr.ID,
		DeviceID:  dev.ID,
		ExpiresAt: &futureExpiry,
	}
	_, err = leaseRepo.UpsertAddressLease(t.Context(), addressLease)
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
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "no-lease-handler-device")
	is.NoErr(err)

	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.3.1", device.EventSourceManual)
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

func TestHandler_GetDevices_CorrectFields(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "list-fields-device")
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var devices []httpapi.Device
	err = json.NewDecoder(rec.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.Equal(devices[0].Id, int64(dev.ID))
	is.Equal(devices[0].Name, dev.Name)
	is.Equal(devices[0].ApiKeyPrefix, dev.KeyPrefix)
	is.True(devices[0].AddressCount != nil)
	is.Equal(*devices[0].AddressCount, 0)
	is.True(!time.Time(devices[0].CreatedAt).IsZero())
}

func TestHandler_GetDevices_AddressCountReflectsEnabledAddresses(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	sessionCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", "AdminPass123!")

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "list-count-device")
	is.NoErr(err)

	addrToDisable, _, err := testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.4.1", device.EventSourceManual)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.4.2", device.EventSourceManual)
	is.NoErr(err)
	_, err = testServer.DeviceService.DisableAddress(t.Context(), dev.ID, addrToDisable.ID)
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var devices []httpapi.Device
	err = json.NewDecoder(rec.Body).Decode(&devices)
	is.NoErr(err)
	is.Equal(len(devices), 1)
	is.True(devices[0].AddressCount != nil)
	is.Equal(*devices[0].AddressCount, 1)
}
