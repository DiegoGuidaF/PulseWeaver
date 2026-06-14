//go:build test

package integrationtest_test

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/lease"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestLeaseRuleSave_DoesNotReExpireMaxRuleDisabledAddresses reproduces a
// production bug in the interaction between the max-active-addresses rule and
// the device-lease TTL rule:
//
//  1. A device with a lease TTL has four enabled addresses. Enabling the
//     max-active rule (max=2) evicts the two oldest (source=limit_exceeded).
//  2. The lease rule is saved (its TTL is changed). Saving runs the device-wide
//     re-arm SetDeviceAddressLeasesExpiry — `UPDATE address_leases SET
//     expires_at = ? WHERE device_id = ?` — over every lease row of the device.
//  3. Time advances past the TTL and the ExpiryJob runs.
//
// Only the two still-enabled addresses may expire. The bug was that disabling an
// address merely nulled its lease's expiry, leaving the row in place, so the
// re-arm in step 2 resurrected the leases of the already-disabled addresses and
// they were "expired" again in a batch. Disabling now deletes the lease row, so
// the re-arm has nothing to touch on disabled addresses.
//
// Addresses are seeded before background services start, and the eviction is
// driven by a single rule-enable API call, to avoid the SQLite shared-cache
// write contention that rapid concurrent address writes would cause.
func TestLeaseRuleSave_DoesNotReExpireMaxRuleDisabledAddresses(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	const (
		maxAddresses  = 2
		initialTTLSec = 300
		newTTLSec     = 600
		ip1           = "10.0.0.1"
		ip2           = "10.0.0.2"
		ip3           = "10.0.0.3"
		ip4           = "10.0.0.4"
	)

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithUser(testutils.UserFixture{Name: "alice"}).
			WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
			WithDeviceLeaseRule(testutils.DeviceLeaseRuleFixture{Device: "alice-laptop", TTLSeconds: initialTTLSec}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip1}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip2}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip3}).
			WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: ip4}).
			WithPolicyInitialize(),
	)

	deviceID := seed.Device("alice-laptop")
	client := testutils.NewAdminAPIClient(t, srv)
	db := srv.Database.DB()

	waitUntil := func(timeout time.Duration, cond func() bool) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if cond() {
				return true
			}
			time.Sleep(2 * time.Millisecond)
		}
		return cond()
	}

	type leaseRow struct {
		AddressID ids.AddressID `db:"address_id"`
		ExpiresAt *time.Time    `db:"expires_at"`
		UpdatedAt time.Time     `db:"updated_at"`
	}
	readLeases := func() []leaseRow {
		var rows []leaseRow
		err := db.SelectContext(ctx, &rows,
			`SELECT address_id, expires_at, updated_at FROM address_leases WHERE device_id = ?`, deviceID)
		is.NoErr(err)
		return rows
	}

	// Enable the max-active rule — this evicts the two oldest addresses.
	maxResp, err := client.PutMaxActiveAddressesRuleWithResponse(ctx, deviceID.Int64(),
		httpapi.PutMaxActiveAddressesRuleJSONRequestBody{MaxAddresses: maxAddresses})
	is.NoErr(err)
	is.Equal(maxResp.StatusCode(), http.StatusOK)

	settled := waitUntil(3*time.Second, func() bool {
		enabled, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
		is.NoErr(err)
		return len(enabled) == maxAddresses
	})
	is.True(settled) // max-active rule must converge to two enabled addresses

	enabledAddrs, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	enabledSet := make(map[ids.AddressID]struct{}, len(enabledAddrs))
	for _, a := range enabledAddrs {
		enabledSet[a.ID] = struct{}{}
	}

	// Core invariant: once the eviction has drained, every remaining lease row
	// belongs to a still-enabled address. A disabled address must hold no lease.
	noLeaseOnDisabled := waitUntil(3*time.Second, func() bool {
		for _, r := range readLeases() {
			if _, enabled := enabledSet[r.AddressID]; !enabled {
				return false
			}
		}
		return true
	})
	is.True(noLeaseOnDisabled) // disabling an address must remove its lease row

	// Save the lease rule with a new TTL — the production trigger. This re-arms
	// the device's leases device-wide; it must not reach the disabled addresses.
	beforeSave := time.Now().UTC()
	ruleResp, err := client.PutDeviceAddressLeaseRuleWithResponse(ctx, deviceID.Int64(),
		httpapi.PutDeviceAddressLeaseRuleJSONRequestBody{TtlSeconds: newTTLSec})
	is.NoErr(err)
	is.Equal(ruleResp.StatusCode(), http.StatusOK)

	// Wait for the async re-arm to land: the enabled leases get a fresh updated_at.
	rearmed := waitUntil(3*time.Second, func() bool {
		rows := readLeases()
		if len(rows) < maxAddresses {
			return false
		}
		for _, r := range rows {
			if !r.UpdatedAt.After(beforeSave) {
				return false
			}
		}
		return true
	})
	is.True(rearmed) // the lease-rule save must have re-armed the enabled leases

	// Advance time past the TTL for every lease that still has an expiry.
	past := time.Now().UTC().Add(-time.Hour)
	_, err = db.ExecContext(ctx,
		`UPDATE address_leases SET expires_at = ? WHERE device_id = ? AND expires_at IS NOT NULL`,
		past, deviceID)
	is.NoErr(err)

	// The expiry pass must select exactly the two still-enabled addresses.
	leaseRepo := lease.NewRepository(db)
	leaseSvc := lease.NewService(leaseRepo, srv.RuleService, slog.New(slog.DiscardHandler))
	expiredIDs, err := leaseSvc.GetExpiredAddressIDs(ctx)
	is.NoErr(err)
	is.Equal(len(expiredIDs), maxAddresses) // only the two enabled addresses, never the disabled ones
	for _, id := range expiredIDs {
		_, stillEnabled := enabledSet[id]
		is.True(stillEnabled)
	}

	// Run the real expiry job and confirm the downstream effect: exactly two
	// expiry events recorded, and no enabled addresses remain.
	expiryJob := leaseSvc.NewExpiryJob(srv.DeviceService)
	is.NoErr(expiryJob.Run(ctx))

	queriesRepo := queries.NewRepository(db)
	historyQuery := queries.AddressHistoryQuery{
		DeviceIDs: []ids.DeviceID{deviceID},
		Source:    new(string(device.EventSourceExpiry)),
	}
	is.NoErr(historyQuery.Validate())
	history, err := queriesRepo.GetAddressHistory(ctx, historyQuery)
	is.NoErr(err)
	is.Equal(len(history.Events), maxAddresses) // exactly two expiry events, one per still-enabled address

	enabledAfter, err := srv.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(enabledAfter), 0)
}
