//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetAccessLog_EmptyRows(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.AccessLogResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	is.Equal(response.Total, 0)
	is.Equal(len(response.Rows), 0)
	is.True(response.NextCursor == nil)
}

func TestHandler_GetAccessLog_CorrectFields(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "access-fields-device")
	is.NoErr(err)
	addr, _, err := testServer.DeviceService.RegisterAddressActivity(t.Context(), dev.ID, "10.0.10.1", device.EventSourceManual)
	is.NoErr(err)

	reason := policy.DenyReasonIPNotRegistered
	targetHost := "example.com"
	targetURI := "/api/test"
	httpMethod := "GET"
	xffChain := "1.2.3.4, 10.0.0.1"
	createdAt := time.Now().UTC().Truncate(time.Second)
	devID := dev.ID
	addrID := addr.ID

	accessRepo := accesslog.NewRepository(testServer.Database.DB())
	err = accessRepo.BatchInsert(t.Context(), []policy.DecisionEvent{
		{
			ClientIP:   "1.2.3.4",
			Outcome:    false,
			DenyReason: &reason,
			DeviceID:   &devID,
			AddressID:  &addrID,
			CreatedAt:  createdAt,
			TargetHost: &targetHost,
			TargetURI:  &targetURI,
			HTTPMethod: &httpMethod,
			XFFChain:   &xffChain,
			Headers:    map[string][]string{"User-Agent": {"TestAgent/1.0"}},
		},
	})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.AccessLogResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	is.Equal(response.Total, 1)
	is.Equal(len(response.Rows), 1)
	is.True(response.NextCursor == nil)

	row := response.Rows[0]
	is.Equal(row.ClientIp, "1.2.3.4")
	is.Equal(row.Outcome, false)
	is.True(row.DenyReason != nil)
	is.Equal(*row.DenyReason, string(policy.DenyReasonIPNotRegistered))
	is.True(row.DeviceId != nil)
	is.Equal(*row.DeviceId, int64(dev.ID))
	is.True(row.DeviceName != nil)
	is.Equal(*row.DeviceName, dev.Name)
	is.True(row.AddressId != nil)
	is.Equal(*row.AddressId, int64(addr.ID))
	is.True(row.TargetHost != nil)
	is.Equal(*row.TargetHost, targetHost)
	is.True(row.TargetUri != nil)
	is.Equal(*row.TargetUri, targetURI)
	is.True(row.HttpMethod != nil)
	is.Equal(*row.HttpMethod, httpMethod)
	is.True(row.XffChain != nil)
	is.Equal(*row.XffChain, xffChain)
	is.True(time.Time(row.CreatedAt).UTC().Truncate(time.Second).Equal(createdAt))
}

func TestHandler_GetAccessLog_FilterByOutcome(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	reason := policy.DenyReasonIPNotRegistered
	accessRepo := accesslog.NewRepository(testServer.Database.DB())
	err := accessRepo.BatchInsert(t.Context(), []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &reason, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &reason, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "3.3.3.3", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log?outcome=false", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.AccessLogResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	is.Equal(response.Total, 2)
	is.Equal(len(response.Rows), 2)
	for _, row := range response.Rows {
		is.Equal(row.Outcome, false)
	}
}

func TestHandler_GetAccessLog_FilterByDeviceID(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	dev, _, err := testServer.DeviceService.CreateDevice(t.Context(), "access-device-filter")
	is.NoErr(err)
	devID := dev.ID

	reason := policy.DenyReasonIPNotRegistered
	accessRepo := accesslog.NewRepository(testServer.Database.DB())
	err = accessRepo.BatchInsert(t.Context(), []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &reason, DeviceID: &devID, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &reason, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // no device
	})
	is.NoErr(err)

	url := fmt.Sprintf("/api/v1/access-log?device_id=%d", devID)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.AccessLogResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	is.Equal(response.Total, 1)
	is.Equal(len(response.Rows), 1)
	is.True(response.Rows[0].DeviceId != nil)
	is.Equal(*response.Rows[0].DeviceId, int64(devID))
}

func TestHandler_GetAccessLog_FilterByIPPartialMatch(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	accessRepo := accesslog.NewRepository(testServer.Database.DB())
	err := accessRepo.BatchInsert(t.Context(), []policy.DecisionEvent{
		{ClientIP: "123.222.234.1", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "234.111.222.111", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "111.111.111.1", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	})
	is.NoErr(err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log?ip=234", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var response httpapi.AccessLogResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	is.NoErr(err)
	is.Equal(response.Total, 2)
	is.Equal(len(response.Rows), 2)
	for _, row := range response.Rows {
		is.True(strings.Contains(row.ClientIp, "234"))
	}
}

func TestHandler_GetAccessLog_Pagination(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	adminCookie := testutils.LoginCookie(t, testServer.HTTPServer, "admin", testutils.TestAdminPassword)

	reason := policy.DenyReasonIPNotRegistered
	accessRepo := accesslog.NewRepository(testServer.Database.DB())
	err := accessRepo.BatchInsert(t.Context(), []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &reason, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &reason, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "3.3.3.3", Outcome: false, DenyReason: &reason, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	})
	is.NoErr(err)

	// First page: limit=2 → 2 rows returned, next_cursor set, total reflects all 3
	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log?limit=2", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var page1 httpapi.AccessLogResponse
	err = json.NewDecoder(rec.Body).Decode(&page1)
	is.NoErr(err)
	is.Equal(page1.Total, 3)
	is.Equal(len(page1.Rows), 2)
	is.True(page1.NextCursor != nil)

	// Second page: before_id=cursor → 1 remaining row, no further cursor
	url := fmt.Sprintf("/api/v1/access-log?limit=2&before_id=%d", *page1.NextCursor)
	req = httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(adminCookie)
	rec = httptest.NewRecorder()
	testServer.HTTPServer.ServeHTTP(rec, req)

	is.Equal(rec.Code, http.StatusOK)

	var page2 httpapi.AccessLogResponse
	err = json.NewDecoder(rec.Body).Decode(&page2)
	is.NoErr(err)
	is.Equal(len(page2.Rows), 1)
	is.True(page2.NextCursor == nil)
}

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
