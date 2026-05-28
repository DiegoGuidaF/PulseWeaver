//go:build test

package integrationtest_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestMaxActiveAddressesRule_EvictsOldestAddressFromPolicyCache is a cross-domain
// integration test that verifies the max-active-addresses rule correctly evicts
// excess addresses and updates the policy cache:
//
//  1. A device has three active addresses with no max-active rule configured.
//     All three IPs are allowed through the policy forward-auth.
//  2. Enabling the rule with max_addresses=2 via the HTTP API fires a
//     RuleEventTypeEnabled event → maxaddr.RunListener enforces the limit →
//     DisableAddresses(…, EventSourceLimitExceeded) → AddressDisabled event →
//     policy cache refresh.
//  3. Exactly two addresses remain enabled; the evicted IP is denied.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestMaxActiveAddressesRule_EvictsOldestAddressFromPolicyCache(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	const (
		ip1         = "10.0.0.1"
		ip2         = "10.0.0.2"
		ip3         = "10.0.0.3"
		backendHost = "api.internal"
	)

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithGroup(testutils.GroupFixture{Name: "backend"}).
			WithHost(testutils.HostFixture{FQDN: backendHost, Groups: []string{"backend"}}).
			WithUser(testutils.UserFixture{Name: "alice"}).
			SetUserAccess("alice", false, "backend").
			WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip1}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip2}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip3}).
			WithPolicyInitialize(),
	)

	deviceID := seed.Device("alice-laptop")
	client := testutils.NewAdminAPIClient(t, srv)

	// Pre-condition: all three IPs are in the cache and allowed.
	for _, ip := range []string{ip1, ip2, ip3} {
		w := verifyIP(t, srv, ip, backendHost)
		is.Equal(w.Code, http.StatusOK)
	}

	// Enable max-active-addresses rule with max=2 — this is the action under test.
	// Flow: RuleEventTypeEnabled → maxaddr.OnRuleEvent → channel →
	//       RunListener.enforce → DisableAddresses(EventSourceLimitExceeded) →
	//       AddressDisabled → policy.OnAddressEvent → triggerRefresh → refreshCache.
	maxAddresses := 2
	before := srv.PolicyService.LastRefreshedAt()
	ruleResp, err := client.PutMaxActiveAddressesRuleWithResponse(ctx, deviceID.Int64(), httpapi.PutMaxActiveAddressesRuleJSONRequestBody{
		MaxAddresses: maxAddresses,
	})
	is.NoErr(err)
	is.Equal(ruleResp.StatusCode(), http.StatusOK)

	// Two async hops: rule event → enforce → address event → cache refresh.
	testutils.WaitForPolicyRefresh(ctx, t, srv, before)

	// Service-layer assertion: exactly two addresses remain enabled.
	enabledAfter, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(enabledAfter), maxAddresses)

	// Determine the evicted IP by diffing the original set against what remains.
	enabledIPs := make(map[string]struct{}, len(enabledAfter))
	for _, a := range enabledAfter {
		enabledIPs[a.IP] = struct{}{}
	}
	var evictedIP string
	for _, ip := range []string{ip1, ip2, ip3} {
		if _, ok := enabledIPs[ip]; !ok {
			evictedIP = ip
			break
		}
	}
	is.True(evictedIP != "") // exactly one address must have been evicted

	// Policy cache assertions: evicted IP denied; remaining two allowed.
	w := verifyIP(t, srv, evictedIP, backendHost)
	is.Equal(w.Code, http.StatusForbidden)

	for ip := range enabledIPs {
		w := verifyIP(t, srv, ip, backendHost)
		is.Equal(w.Code, http.StatusOK)
	}

	// Address-history assertion: the evicted address was disabled with
	// EventSourceLimitExceeded, confirming the maxaddr enforcer was responsible.
	historyQuery := device.AddressHistoryQuery{
		DeviceIDs: []ids.DeviceID{deviceID},
		Source:    strPtr(string(device.EventSourceLimitExceeded)),
	}
	is.NoErr(historyQuery.Validate())
	history, err := srv.DeviceService.GetAddressHistory(ctx, historyQuery)
	is.NoErr(err)
	is.Equal(len(history.Events), 1)
	is.Equal(history.Events[0].IP, evictedIP)
	is.Equal(history.Events[0].Source, device.EventSourceLimitExceeded)
}

func strPtr(s string) *string { return &s }
