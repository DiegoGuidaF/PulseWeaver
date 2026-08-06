//go:build test

package buildmeta_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetVersion(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.GetVersionWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	// A test binary is never link-stamped, so it must report the honest defaults.
	is.Equal(resp.JSON200.Version, "dev")
	is.Equal(resp.JSON200.Commit, "unknown")
	is.True(resp.JSON200.BuildTime == nil)
}

// The public /health endpoint carries no version on purpose; this endpoint is
// the one that discloses it, so it must stay behind authentication.
func TestHandler_GetVersion_Unauthenticated_Returns401(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)

	resp, err := testutils.NewAPIClient(t, srv).GetVersionWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}
