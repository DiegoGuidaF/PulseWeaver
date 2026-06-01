//go:build test

package hosts_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_ReconcileHosts(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ReconcileHostsWithResponse(ctx, httpapi.ReconcileHostsJSONRequestBody{
		Hosts: []httpapi.HostInput{
			{Fqdn: "router.example.com", GroupIds: []httpapi.ID{}},
		},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_ReconcileHostGroups(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ReconcileHostGroupsWithResponse(ctx, httpapi.ReconcileHostGroupsJSONRequestBody{
		Groups: []httpapi.GroupWrite{
			{Name: "infra", Color: "#000000", Icon: "server", HostIds: []httpapi.ID{}},
		},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_ReconcileHostGroups_InvalidColor_Returns400(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	// "notahex" triggers the color validation → 400.
	resp, err := client.ReconcileHostGroupsWithResponse(ctx, httpapi.ReconcileHostGroupsJSONRequestBody{
		Groups: []httpapi.GroupWrite{
			{Name: "infra", Color: "notahex", Icon: "server", HostIds: []httpapi.ID{}},
		},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_IgnoreSuggestion(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.IgnoreSuggestionWithResponse(ctx, httpapi.IgnoreSuggestionJSONRequestBody{
		Fqdn: "ignored.example.com",
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusCreated)
	is.Equal(resp.JSON201.Fqdn, "ignored.example.com")
	is.True(resp.JSON201.Id != 0)
}

func TestHandler_UnignoreSuggestion(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	_, err := srv.HostsService.AddIgnoredSuggestion(ctx, "ignored.example.com")
	is.NoErr(err)

	resp, err := client.UnignoreSuggestionWithResponse(ctx, "ignored.example.com")
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}
