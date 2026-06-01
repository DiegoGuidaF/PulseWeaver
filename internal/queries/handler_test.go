//go:build test

package queries_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// SimulatePolicyAccess is implemented in the policy package; these integration
// tests live here because they rely on the same SetupIntegrationServer harness
// used by the rest of the queries handler tests.

func TestHandler_SimulatePolicyAccess_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	resp, err := testutils.NewAPIClient(t, testServer).SimulatePolicyAccessWithResponse(ctx, &httpapi.SimulatePolicyAccessParams{
		Ip:   "1.2.3.4",
		Host: "example.com",
	})
	is.NoErr(err)
	is.True(resp.StatusCode() == http.StatusUnauthorized || resp.StatusCode() == http.StatusForbidden)
}

func TestHandler_SimulatePolicyAccess_IPNotRegistered(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.SimulatePolicyAccessWithResponse(ctx, &httpapi.SimulatePolicyAccessParams{
		Ip:   "9.9.9.9",
		Host: "example.com",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	result := *resp.JSON200
	is.True(!result.Allowed)
	is.True(result.DenyReason != nil)
	is.Equal(string(*result.DenyReason), "ip_not_registered")
}

func TestHandler_SimulatePolicyAccess_HostNotAllowed(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "sim-device", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "5.6.7.9", device.EventSourceManual)
	is.NoErr(err)
	is.NoErr(testServer.PolicyService.Initialize(ctx))

	resp, err := client.SimulatePolicyAccessWithResponse(ctx, &httpapi.SimulatePolicyAccessParams{
		Ip:   "5.6.7.9",
		Host: "denied.example.com",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	result := *resp.JSON200
	is.True(!result.Allowed)
	is.True(result.DenyReason != nil)
	is.Equal(string(*result.DenyReason), "host_not_allowed")
}

func TestHandler_SimulatePolicyAccess_AllowedBypass(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	principal := testutils.AdminPrincipal(t, testServer)
	is.NoErr(testServer.UserAccessService.SetUserAccess(ctx, principal.UserID, true, nil))

	dev, err := testServer.DeviceService.CreateDevice(ctx, principal, "sim-bypass-device", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "5.6.7.10", device.EventSourceManual)
	is.NoErr(err)
	is.NoErr(testServer.PolicyService.Initialize(ctx))

	resp, err := client.SimulatePolicyAccessWithResponse(ctx, &httpapi.SimulatePolicyAccessParams{
		Ip:   "5.6.7.10",
		Host: "anything.example.com",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	result := *resp.JSON200
	is.True(result.Allowed)
	is.True(result.DenyReason == nil)
}

func TestHandler_SimulatePolicyAccess_DoesNotWriteAccessLog(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.SimulatePolicyAccessWithResponse(ctx, &httpapi.SimulatePolicyAccessParams{
		Ip:   "9.9.9.9",
		Host: "example.com",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	// Access log must remain empty.
	logResp, err := client.GetAccessLogWithResponse(ctx, &httpapi.GetAccessLogParams{})
	is.NoErr(err)
	is.Equal(logResp.StatusCode(), http.StatusOK)
	is.Equal((*logResp.JSON200).Total, 0)
}
