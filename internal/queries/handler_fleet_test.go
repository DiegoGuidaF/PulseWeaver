//go:build test

package queries_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rule"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_GetDeviceFleet_NestsRulesAndPairingPerDevice(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "amara"}).
		WithDevice(testutils.DeviceFixture{Name: "amara-router", OwnerUser: "amara", GenerateAPIKey: true}).
		WithDevice(testutils.DeviceFixture{Name: "amara-tablet", OwnerUser: "amara"}).
		WithDeviceLeaseRule(testutils.DeviceLeaseRuleFixture{Device: "amara-router", TTLSeconds: 3600}).
		WithDeviceMaxActiveRule(testutils.DeviceMaxActiveRuleFixture{Device: "amara-router", MaxAddresses: 2}).
		WithPairing(testutils.PairingFixture{Device: "amara-router", Status: "pending"}).
		WithAddress(testutils.AddressFixture{Device: "amara-router", IP: "10.5.0.1"}).
		Build(srv)

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	group := findFleetGroup(*resp.JSON200, seed.User("amara").Int64())
	is.True(group != nil)
	is.Equal(group.Owner.DeviceCount, 2)
	is.Equal(group.Owner.LiveAddressCount, 1)

	router := findFleetDevice(group.Devices, "amara-router")
	is.True(router != nil)
	is.Equal(len(router.Rules), 2)
	is.True(router.Pairing != nil)
	is.Equal(router.Pairing.Status, httpapi.DevicePairingStatusPending)
	is.Equal(router.State, httpapi.DeviceStateHealthy)
	is.True(router.ApiKeyPrefix != nil)
	is.True(router.LastSeenAt != nil)

	// A device with neither rules nor a pairing renders as an empty list and a
	// null pairing, never as an absent device.
	tablet := findFleetDevice(group.Devices, "amara-tablet")
	is.True(tablet != nil)
	is.Equal(len(tablet.Rules), 0)
	is.True(tablet.Pairing == nil)
	is.Equal(tablet.State, httpapi.DeviceStateStale)
}

// TestHandler_GetDeviceFleet_OwnerWithNoDevices_IsAGroupWithEmptyDevices is the
// regression guard for the deleted ownerExists probes: the composite shape has
// to distinguish "owner with no devices" from "unknown owner" structurally.
func TestHandler_GetDeviceFleet_OwnerWithNoDevices_IsAGroupWithEmptyDevices(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "bilal"}).
		Build(srv)

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{
		OwnerId: new(seed.User("bilal").Int64()),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	groups := *resp.JSON200
	is.Equal(len(groups), 1)
	is.Equal(groups[0].Owner.Id, seed.User("bilal").Int64())
	is.Equal(len(groups[0].Devices), 0)
	is.Equal(groups[0].Owner.DeviceCount, 0)
	is.Equal(groups[0].Owner.LiveAddressCount, 0)
}

func TestHandler_GetDeviceFleet_UnknownOwner_EmptyArrayNot404(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{
		OwnerId: new(int64(99999)),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(*resp.JSON200), 0)
}

// TestHandler_GetDeviceFleet_FilteredMatchesUnfilteredEntry is the invariant the
// whole "one endpoint, two views" design rests on: the detail page must be able
// to consume exactly the list page's element type, with identical content.
func TestHandler_GetDeviceFleet_FilteredMatchesUnfilteredEntry(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "ops"}).
		WithUser(testutils.UserFixture{Name: "carmen"}).
		WithUser(testutils.UserFixture{Name: "dmitri"}).
		SetUserAccess("carmen", true, "ops").
		WithDevice(testutils.DeviceFixture{Name: "carmen-nas", OwnerUser: "carmen", GenerateAPIKey: true}).
		WithDevice(testutils.DeviceFixture{Name: "carmen-phone", OwnerUser: "carmen"}).
		WithDevice(testutils.DeviceFixture{Name: "dmitri-laptop", OwnerUser: "dmitri"}).
		WithDeviceMaxActiveRule(testutils.DeviceMaxActiveRuleFixture{Device: "carmen-nas", MaxAddresses: 3}).
		WithPairing(testutils.PairingFixture{Device: "carmen-phone", Status: "used"}).
		WithAddress(testutils.AddressFixture{Device: "carmen-nas", IP: "10.6.0.1"}).
		WithAddress(testutils.AddressFixture{Device: "carmen-nas", IP: "10.6.0.2"}).
		Build(srv)

	carmenID := seed.User("carmen").Int64()

	listResp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{})
	is.NoErr(err)
	is.Equal(listResp.StatusCode(), http.StatusOK)

	detailResp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{
		OwnerId: new(carmenID),
	})
	is.NoErr(err)
	is.Equal(detailResp.StatusCode(), http.StatusOK)

	detail := *detailResp.JSON200
	is.Equal(len(detail), 1)
	fromList := findFleetGroup(*listResp.JSON200, carmenID)
	is.True(fromList != nil)
	is.Equal(detail[0], *fromList)
}

func TestHandler_GetDeviceFleet_SkipsUnparseableRuleConfig(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "elif"}).
		WithDevice(testutils.DeviceFixture{Name: "elif-hub", OwnerUser: "elif"}).
		WithDeviceMaxActiveRule(testutils.DeviceMaxActiveRuleFixture{Device: "elif-hub", MaxAddresses: 2}).
		Build(srv)

	// Only raw SQL can produce a rule whose config no longer parses — the
	// service validates every config it writes.
	insertBrokenRule(t, srv, seed.Device("elif-hub"))

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{
		OwnerId: new(seed.User("elif").Int64()),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	devices := (*resp.JSON200)[0].Devices
	is.Equal(len(devices), 1)
	is.Equal(len(devices[0].Rules), 1)
	is.Equal(devices[0].Rules[0].Type, httpapi.MaxActive)
}

func TestHandler_GetDeviceFleet_ExcludesSoftDeletedDevicesFromListAndAggregates(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "farida"}).
		WithDevice(testutils.DeviceFixture{Name: "farida-live", OwnerUser: "farida"}).
		WithDevice(testutils.DeviceFixture{Name: "farida-gone", OwnerUser: "farida", Deleted: true}).
		WithAddress(testutils.AddressFixture{Device: "farida-live", IP: "10.7.0.1"}).
		WithAddress(testutils.AddressFixture{Device: "farida-gone", IP: "10.7.0.2"}).
		Build(srv)

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{
		OwnerId: new(seed.User("farida").Int64()),
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	group := (*resp.JSON200)[0]
	is.Equal(len(group.Devices), 1)
	is.Equal(group.Devices[0].Name, "farida-live")
	is.Equal(group.Owner.DeviceCount, 1)
	is.Equal(group.Owner.LiveAddressCount, 1)
}

// TestHandler_GetDeviceFleet_OrdersOwnersByDisplayNameThenID pins the id
// tiebreak: display names are not unique, so without it two owners sharing one
// can swap places between refetches.
func TestHandler_GetDeviceFleet_OrdersOwnersByDisplayNameThenID(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "zoe_second", DisplayName: "Zoe"}).
		WithUser(testutils.UserFixture{Name: "zoe_first", DisplayName: "Zoe"}).
		WithUser(testutils.UserFixture{Name: "aiko", DisplayName: "Aiko"}).
		WithDevice(testutils.DeviceFixture{Name: "aiko-zeta", OwnerUser: "aiko"}).
		WithDevice(testutils.DeviceFixture{Name: "aiko-alpha", OwnerUser: "aiko"}).
		Build(srv)

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	groups := *resp.JSON200

	names := make([]string, len(groups))
	zoeOrder := []int64{}
	for i, g := range groups {
		names[i] = g.Owner.DisplayName
		if g.Owner.DisplayName == "Zoe" {
			zoeOrder = append(zoeOrder, g.Owner.Id)
		}
	}
	is.True(slices.IsSorted(names))
	is.Equal(len(zoeOrder), 2)
	is.Equal(zoeOrder[0], seed.User("zoe_second").Int64()) // lower id, seeded first
	is.True(zoeOrder[0] < zoeOrder[1])

	aiko := findFleetGroup(groups, seed.User("aiko").Int64())
	is.True(aiko != nil)
	is.Equal(aiko.Devices[0].Name, "aiko-alpha")
	is.Equal(aiko.Devices[1].Name, "aiko-zeta")
}

// TestHandler_GetDeviceFleet_DoesNotCrossContaminateOwners is the id-keyed
// readers' characteristic failure mode: a rule or pairing attached to the wrong
// owner's device.
func TestHandler_GetDeviceFleet_DoesNotCrossContaminateOwners(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	seed := testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "goro"}).
		WithUser(testutils.UserFixture{Name: "hana"}).
		WithDevice(testutils.DeviceFixture{Name: "goro-pi", OwnerUser: "goro"}).
		WithDevice(testutils.DeviceFixture{Name: "hana-pi", OwnerUser: "hana"}).
		WithDeviceMaxActiveRule(testutils.DeviceMaxActiveRuleFixture{Device: "goro-pi", MaxAddresses: 1}).
		WithPairing(testutils.PairingFixture{Device: "hana-pi", Status: "pending"}).
		Build(srv)

	resp, err := client.GetDeviceFleetWithResponse(ctx, &httpapi.GetDeviceFleetParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	groups := *resp.JSON200

	goro := findFleetGroup(groups, seed.User("goro").Int64())
	is.True(goro != nil)
	is.Equal(len(goro.Devices), 1)
	is.Equal(len(goro.Devices[0].Rules), 1)
	is.True(goro.Devices[0].Pairing == nil)

	hana := findFleetGroup(groups, seed.User("hana").Int64())
	is.True(hana != nil)
	is.Equal(len(hana.Devices), 1)
	is.Equal(len(hana.Devices[0].Rules), 0)
	is.True(hana.Devices[0].Pairing != nil)
}

func insertBrokenRule(t *testing.T, srv *app.App, deviceID ids.DeviceID) {
	t.Helper()
	// The truncated config is inserted as a []byte because json.RawMessage
	// cannot scan a TEXT value.
	_, err := srv.Database.DB().ExecContext(t.Context(),
		`INSERT INTO device_rules (device_id, rule_type, enabled, config) VALUES (?, ?, 1, ?)`,
		deviceID, rule.RuleTypeDeviceAddressLease, []byte(`{"ttl_seconds":`))
	if err != nil {
		t.Fatalf("insert broken rule: %v", err)
	}
}

func findFleetGroup(groups []httpapi.OwnerFleetGroup, ownerID int64) *httpapi.OwnerFleetGroup {
	for i := range groups {
		if groups[i].Owner.Id == ownerID {
			return &groups[i]
		}
	}
	return nil
}

func findFleetDevice(devices []httpapi.FleetDevice, name string) *httpapi.FleetDevice {
	for i := range devices {
		if devices[i].Name == name {
			return &devices[i]
		}
	}
	return nil
}
