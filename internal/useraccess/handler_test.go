//go:build test

package useraccess_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_SetUserHostGrants(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	adminID := testutils.AdminPrincipal(t, srv).UserID

	resp, err := client.SetUserAccessWithResponse(ctx, adminID.Int64(), httpapi.SetUserAccessJSONRequestBody{
		BypassHostCheck: false,
		GroupIds:        []httpapi.ID{},
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNoContent)
}
