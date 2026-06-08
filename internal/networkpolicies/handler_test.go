//go:build test

package networkpolicies_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// ── CreateNetworkPolicy ───────────────────────────────────────────────────────

func TestHandler_CreateNetworkPolicy(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.CreateNetworkPolicyWithResponse(ctx, httpapi.CreateNetworkPolicyJSONRequestBody{
		Name: "home",
		Cidr: "192.168.1.5/24",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	is.Equal(resp.JSON201.Name, "home")
	is.Equal(resp.JSON201.Cidr, "192.168.1.0/24") // host bits zeroed
	is.True(resp.JSON201.Enabled)
}

func TestHandler_CreateNetworkPolicy_InvalidCIDR(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.CreateNetworkPolicyWithResponse(ctx, httpapi.CreateNetworkPolicyJSONRequestBody{
		Name: "bad",
		Cidr: "not-a-cidr",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_CreateNetworkPolicy_TooBroadCIDR(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.CreateNetworkPolicyWithResponse(ctx, httpapi.CreateNetworkPolicyJSONRequestBody{
		Name: "allow-all",
		Cidr: "0.0.0.0/0",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_CreateNetworkPolicy_DuplicateCIDR(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	_, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "first", "10.0.0.0/16", nil)
	is.NoErr(err)

	resp, err := client.CreateNetworkPolicyWithResponse(ctx, httpapi.CreateNetworkPolicyJSONRequestBody{
		Name: "second",
		Cidr: "10.0.0.0/16",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusConflict)
}

func TestHandler_CreateNetworkPolicy_Unauthenticated(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)

	resp, err := testutils.NewAPIClient(t, srv).CreateNetworkPolicyWithResponse(ctx, httpapi.CreateNetworkPolicyJSONRequestBody{
		Name: "home",
		Cidr: "10.0.0.0/16",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

// ── UpdateNetworkPolicy ───────────────────────────────────────────────────────

func TestHandler_UpdateNetworkPolicy(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	p, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "original", "10.1.0.0/16", nil)
	is.NoErr(err)

	descEmpty := ""
	resp, err := client.UpdateNetworkPolicyWithResponse(ctx, p.ID.Int64(), httpapi.UpdateNetworkPolicyJSONRequestBody{
		Name:        "renamed",
		Cidr:        "10.1.0.0/16",
		Enabled:     false,
		Description: &descEmpty,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_UpdateNetworkPolicy_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	descEmpty := ""
	resp, err := client.UpdateNetworkPolicyWithResponse(ctx, httpapi.ID(99999), httpapi.UpdateNetworkPolicyJSONRequestBody{
		Name:        "ghost",
		Cidr:        "10.2.0.0/16",
		Enabled:     true,
		Description: &descEmpty,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestHandler_UpdateNetworkPolicy_InvalidCIDR(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	p, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "p", "10.3.0.0/16", nil)
	is.NoErr(err)

	descEmpty := ""
	resp, err := client.UpdateNetworkPolicyWithResponse(ctx, p.ID.Int64(), httpapi.UpdateNetworkPolicyJSONRequestBody{
		Name:        "p",
		Cidr:        "bad-cidr",
		Enabled:     true,
		Description: &descEmpty,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_UpdateNetworkPolicy_DuplicateCIDR(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	_, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "taken", "10.4.0.0/16", nil)
	is.NoErr(err)
	p, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "target", "10.5.0.0/16", nil)
	is.NoErr(err)

	descEmpty := ""
	resp, err := client.UpdateNetworkPolicyWithResponse(ctx, p.ID.Int64(), httpapi.UpdateNetworkPolicyJSONRequestBody{
		Name:        "target",
		Cidr:        "10.4.0.0/16",
		Enabled:     true,
		Description: &descEmpty,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusConflict)
}

// ── DeleteNetworkPolicy ───────────────────────────────────────────────────────

func TestHandler_DeleteNetworkPolicy(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	p, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "to-delete", "10.6.0.0/16", nil)
	is.NoErr(err)

	resp, err := client.DeleteNetworkPolicyWithResponse(ctx, p.ID.Int64())
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_DeleteNetworkPolicy_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.DeleteNetworkPolicyWithResponse(ctx, httpapi.ID(99999))
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

// ── UpdateNetworkPolicyAccess ─────────────────────────────────────────────────

func TestHandler_UpdateNetworkPolicyAccess(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	p, err := srv.NetworkPoliciesService.CreatePolicy(ctx, "access-target", "10.7.0.0/16", nil)
	is.NoErr(err)

	resp, err := client.UpdateNetworkPolicyAccessWithResponse(ctx, p.ID.Int64(), httpapi.UpdateNetworkPolicyAccessJSONRequestBody{
		BypassHostCheck: true,
		GroupIds:        []httpapi.ID{},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_UpdateNetworkPolicyAccess_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.UpdateNetworkPolicyAccessWithResponse(ctx, httpapi.ID(99999), httpapi.UpdateNetworkPolicyAccessJSONRequestBody{
		BypassHostCheck: false,
		GroupIds:        []httpapi.ID{},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}
