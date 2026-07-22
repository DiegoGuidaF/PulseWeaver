//go:build test

package device_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_ListDeviceRefs_ReturnsFlatRefsForAllDevices(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "frank"}).
		WithUser(testutils.UserFixture{Name: "grace"}).
		WithDevice(testutils.DeviceFixture{Name: "frank-desktop", OwnerUser: "frank"}).
		WithDevice(testutils.DeviceFixture{Name: "grace-tablet", OwnerUser: "grace"}).
		Build(srv)

	resp, err := client.ListDeviceRefsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	refs := *resp.JSON200

	frankDevice := findRef(refs, "frank-desktop")
	is.True(frankDevice != nil)
	is.Equal(frankDevice.OwnerId, seed.User("frank").Int64())
	is.Equal(frankDevice.Id, seed.Device("frank-desktop").Int64())

	graceDevice := findRef(refs, "grace-tablet")
	is.True(graceDevice != nil)
	is.Equal(graceDevice.OwnerId, seed.User("grace").Int64())
}

func findRef(refs []httpapi.DeviceRef, name string) *httpapi.DeviceRef {
	for i := range refs {
		if refs[i].Name == name {
			return &refs[i]
		}
	}
	return nil
}
