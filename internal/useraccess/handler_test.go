//go:build test

package useraccess_test

import (
	"net/http"
	"slices"
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

func TestHandler_ListOwnerRefs(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "yara", DisplayName: "Yara"}).
		WithUser(testutils.UserFixture{Name: "bruno", DisplayName: "Bruno"}).
		Build(srv)

	resp, err := client.ListOwnerRefsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	refs := *resp.JSON200

	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.DisplayName
	}
	is.True(slices.IsSorted(names))

	bruno := findOwnerRef(refs, seed.User("bruno").Int64())
	is.True(bruno != nil)
	is.Equal(bruno.DisplayName, "Bruno")
	is.True(findOwnerRef(refs, seed.User("yara").Int64()) != nil)
}

func findOwnerRef(refs []httpapi.OwnerRef, id int64) *httpapi.OwnerRef {
	for i := range refs {
		if refs[i].Id == id {
			return &refs[i]
		}
	}
	return nil
}
