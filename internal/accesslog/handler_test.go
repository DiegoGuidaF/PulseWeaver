//go:build test

package accesslog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDenyReasons_Empty(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.GetAccessLogDenyReasonsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(*resp.JSON200), 0)
}

func TestHandler_GetDenyReasons_WithData(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	// Seed decision events directly: there is no Seeder fixture for raw access-log
	// events, and they are the Given here, not the request under test.
	repo := accesslog.NewRepository(testServer.Database.DB())
	r1 := policy.DenyReasonIPNotRegistered
	r2 := policy.DenyReasonNoDeviceMatch
	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // duplicate — deduplicated
		{ClientIP: "3.3.3.3", Outcome: false, DenyReason: &r2, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "4.4.4.4", Outcome: true, DenyReason: nil, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // allow — excluded
	}
	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	resp, err := client.GetAccessLogDenyReasonsWithResponse(ctx)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	reasons := *resp.JSON200
	is.Equal(len(reasons), 2)
	is.Equal(reasons[0], string(r1))
	is.Equal(reasons[1], string(r2))
}
