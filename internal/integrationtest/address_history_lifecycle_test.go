//go:build test

package integrationtest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// histogramFor fetches the address-history histogram scoped to one device over
// the default window.
func histogramFor(t *testing.T, client *httpapi.ClientWithResponses, deviceID ids.DeviceID) *httpapi.AddressHistoryHistogramResponse {
	t.Helper()
	is := is.New(t)

	deviceIDFilter := []httpapi.ID{deviceID.Int64()}
	resp, err := client.GetAddressHistoryHistogramWithResponse(t.Context(), &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	return resp.JSON200
}

// tuningFor fetches the address-history tuning readout scoped to one device
// over the default window — device_id bypasses the ranking threshold, so the
// device's readout is returned whether or not it would qualify fleet-wide.
func tuningFor(t *testing.T, client *httpapi.ClientWithResponses, deviceID ids.DeviceID) *httpapi.AddressHistoryTuningResponse {
	t.Helper()
	is := is.New(t)

	id := deviceID.Int64()
	resp, err := client.GetAddressHistoryTuningWithResponse(t.Context(), &httpapi.GetAddressHistoryTuningParams{
		DeviceId: &id,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	return resp.JSON200
}

// eventsFor fetches the address-history events scoped to one device over the
// default window.
func eventsFor(t *testing.T, client *httpapi.ClientWithResponses, deviceID ids.DeviceID) []httpapi.AddressHistoryEvent {
	t.Helper()
	is := is.New(t)

	deviceIDFilter := []httpapi.ID{deviceID.Int64()}
	resp, err := client.GetAddressHistoryWithResponse(t.Context(), &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	return resp.JSON200.Events
}

// countRisk counts events at a given ttl_risk level.
func countRisk(events []httpapi.AddressHistoryEvent, risk httpapi.TTLRisk) int {
	n := 0
	for _, e := range events {
		if e.TtlRisk == risk {
			n++
		}
	}
	return n
}

// TestLeaseRuleChange_RetunesAddressHistoryRisk is a cross-domain integration
// test: the rule domain owns the lease TTL, the device domain owns the address
// events, and the address-history read model scores one against the other on
// every read. The events never change here — only the rule does — so the whole
// test measures the seam between the two domains:
//
//  1. With a generous TTL every renewal reads ok and nothing is ranked.
//  2. Tightening the TTL via the rules API reclassifies the same rows as
//     breached and puts the device into the tuning readout, with counts and a
//     p95 gap the events endpoint agrees with.
//  3. Disabling the rule removes the yardstick entirely: the rows fall back to
//     unknown, the ranking empties, and the buckets read all-zero.
//
// Step 3 is the one worth an integration test on its own — the read model
// reaches into device_rules with an `enabled = 1` join condition, so a rule
// that is disabled rather than deleted must stop counting.
func TestLeaseRuleChange_RetunesAddressHistoryRisk(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	const (
		deviceName = "james-pixel"
		deviceIP   = "10.0.0.42"
		// The seeded renewals are 30 minutes apart. A 4-hour TTL leaves them
		// comfortably ok; a 60-second TTL puts every one of them past it.
		generousTTLSeconds = int(4 * time.Hour / time.Second)
		tightTTLSeconds    = 60
		seededRenewals     = 3 // four events, the first of which has nothing to measure against
	)

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithUser(testutils.UserFixture{Name: "james"}).
			WithDevice(testutils.DeviceFixture{Name: deviceName, OwnerUser: "james"}).
			WithDeviceLeaseRule(testutils.DeviceLeaseRuleFixture{Device: deviceName, TTLSeconds: generousTTLSeconds}).
			WithAddress(testutils.AddressFixture{
				Device: deviceName,
				IP:     deviceIP,
				History: []testutils.AddressEventFixture{
					{Ago: 100 * time.Minute, Enabled: true, Source: device.EventSourceHeartbeat},
					{Ago: 70 * time.Minute, Enabled: true, Source: device.EventSourceHeartbeat},
					{Ago: 40 * time.Minute, Enabled: true, Source: device.EventSourceHeartbeat},
					{Ago: 10 * time.Minute, Enabled: true, Source: device.EventSourceHeartbeat},
				},
			}),
	)

	deviceID := seed.Device(deviceName)
	client := testutils.NewAdminAPIClient(t, srv)

	// 1 — a generous TTL: the same cadence reads as healthy. The device-scoped
	// tuning readout renders regardless — it is a comfortable reading here,
	// not a recommendation, since seededRenewals is well under the fleet
	// ranking's sample floor and device_id bypasses that floor entirely.
	events := eventsFor(t, client, deviceID)
	is.Equal(countRisk(events, httpapi.Ok), seededRenewals)
	tuning := tuningFor(t, client, deviceID)
	is.Equal(tuning.Total, 1)
	is.Equal(len(tuning.Devices), 1)
	is.True(tuning.Devices[0].P95GapSeconds <= tuning.Devices[0].TtlSeconds) // healthy — no misfit to report
	histogram := histogramFor(t, client, deviceID)
	okDevices := 0
	for _, b := range histogram.Buckets {
		okDevices += b.OkDeviceCount
	}
	is.True(okDevices > 0) // …but they are still charted, as the denominator

	// 2 — tighten the TTL through the rules API. The action under test.
	putResp, err := client.PutDeviceAddressLeaseRuleWithResponse(ctx, deviceID.Int64(),
		httpapi.PutDeviceAddressLeaseRuleJSONRequestBody{TtlSeconds: tightTTLSeconds},
	)
	is.NoErr(err)
	is.Equal(putResp.StatusCode(), http.StatusOK)

	events = eventsFor(t, client, deviceID)
	is.Equal(countRisk(events, httpapi.Breached), seededRenewals) // the identical rows, rescored
	is.Equal(countRisk(events, httpapi.Ok), 0)

	tuning = tuningFor(t, client, deviceID)
	is.Equal(tuning.Total, 1)
	is.Equal(len(tuning.Devices), 1)
	ranked := tuning.Devices[0]
	is.Equal(ranked.DeviceId, deviceID.Int64())
	is.Equal(ranked.TtlSeconds, int64(tightTTLSeconds)) // the TTL the rules API just stored
	is.Equal(ranked.RenewalCount, seededRenewals)
	is.Equal(ranked.LateRenewalCount, seededRenewals)

	// The tuning readout has to be actionable: a p95 gap the operator can size
	// a new TTL against, agreeing with the gaps the table beside it shows.
	var maxGap int64
	for _, e := range events {
		if e.RenewalGapSeconds != nil && *e.RenewalGapSeconds > maxGap {
			maxGap = *e.RenewalGapSeconds
		}
	}
	is.True(ranked.P95GapSeconds > 0)
	is.Equal(ranked.P95GapSeconds, maxGap)            // few renewals — nearest-rank is the maximum
	is.True(ranked.P95GapSeconds > ranked.TtlSeconds) // …and it says the TTL is too short

	// 3 — disable the rule. With no TTL to score against, nothing is measurable.
	disableResp, err := client.DisableDeviceAddressLeaseRuleWithResponse(ctx, deviceID.Int64())
	is.NoErr(err)
	is.Equal(disableResp.StatusCode(), http.StatusNoContent)

	events = eventsFor(t, client, deviceID)
	is.Equal(countRisk(events, httpapi.Unknown), len(events)) // every row, including the renewals
	for _, e := range events {
		is.True(e.TtlSeconds == nil) // a disabled rule is not a configured TTL
	}

	tuning = tuningFor(t, client, deviceID)
	is.Equal(tuning.Total, 0)
	is.Equal(len(tuning.Devices), 0) // no lease rule — empty, not an error
	histogram = histogramFor(t, client, deviceID)
	is.True(len(histogram.Buckets) > 0) // still contiguous across the window
	for _, b := range histogram.Buckets {
		is.Equal(b.OkDeviceCount, 0) // an unmeasurable device is not a healthy one
		is.Equal(b.ApproachingDeviceCount, 0)
		is.Equal(b.CriticalDeviceCount, 0)
		is.Equal(b.BreachedDeviceCount, 0)
	}
}

// TestAddressExpiry_CountsAsLateRenewalNotBreachedEvent is a cross-domain
// integration test covering the write path the read model is built to explain.
// Every other address-history test fabricates its events with direct inserts;
// this one lets the services write them — a real registration, the same
// DisableAddresses call the lease expiry job makes, and a real re-registration
// afterwards — then checks the read model tells the true story:
//
//   - the expiry row itself is never a breach; it terminates a lease rather
//     than renewing one, so it carries no gap and stays out of the denominator,
//   - the heartbeat that comes back after it is the breach, and
//   - the ranking's counters reflect exactly that split.
//
// Getting this backwards would report every routine nightly expiry as a TTL
// violation, which is the failure the renewal-gap model exists to prevent.
func TestAddressExpiry_CountsAsLateRenewalNotBreachedEvent(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	const (
		deviceName = "noah-phone"
		deviceIP   = "10.0.0.43"
		// Short enough that a one-second pause between heartbeats already
		// breaches it, so the flow needs no long wall-clock wait.
		ttlSeconds = 1
	)

	srv, seed := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithUser(testutils.UserFixture{Name: "noah"}).
			WithDevice(testutils.DeviceFixture{Name: deviceName, OwnerUser: "noah"}).
			WithDeviceLeaseRule(testutils.DeviceLeaseRuleFixture{Device: deviceName, TTLSeconds: ttlSeconds}),
	)

	deviceID := seed.Device(deviceName)
	client := testutils.NewAdminAPIClient(t, srv)

	// The device checks in for the first time — nothing to measure yet.
	addr, _, err := srv.DeviceService.RegisterAddressActivity(ctx, deviceID, deviceIP, device.EventSourceHeartbeat)
	is.NoErr(err)

	// The lease lapses. This is the call the lease expiry job makes when it
	// finds an expired address; driving it directly keeps the test off the
	// scheduler's interval without faking the event.
	time.Sleep(1100 * time.Millisecond)
	err = srv.DeviceService.DisableAddresses(ctx, []ids.AddressID{addr.ID}, device.EventSourceExpiry)
	is.NoErr(err)

	// …and the device comes back, late.
	time.Sleep(1100 * time.Millisecond)
	_, _, err = srv.DeviceService.RegisterAddressActivity(ctx, deviceID, deviceIP, device.EventSourceHeartbeat)
	is.NoErr(err)

	events := eventsFor(t, client, deviceID)
	is.Equal(len(events), 3) // registration, expiry, late re-registration

	var expiry, lateRenewal *httpapi.AddressHistoryEvent
	for i, e := range events {
		if e.Source == httpapi.Expiry {
			expiry = &events[i]
		}
		if e.Source == httpapi.Heartbeat && e.RenewalGapSeconds != nil {
			lateRenewal = &events[i]
		}
	}

	is.True(expiry != nil)
	is.True(expiry.RenewalGapSeconds == nil)  // an expiry renews nothing
	is.Equal(expiry.TtlRisk, httpapi.Unknown) // so it is never scored, never breached

	is.True(lateRenewal != nil)
	is.Equal(lateRenewal.TtlRisk, httpapi.Breached) // the late return is the breach
	// The gap is measured from the previous renewal, skipping the expiry that
	// sits between them — otherwise it would read as the shorter post-expiry
	// interval and under-report the outage.
	is.True(*lateRenewal.RenewalGapSeconds >= int64(2*ttlSeconds))

	tuning := tuningFor(t, client, deviceID)
	is.Equal(tuning.Total, 1)
	is.Equal(len(tuning.Devices), 1)
	ranked := tuning.Devices[0]
	is.Equal(ranked.DeviceId, deviceID.Int64())
	is.Equal(ranked.RenewalCount, 1)     // only the late heartbeat had a gap to measure
	is.Equal(ranked.LateRenewalCount, 1) // …and it was late
	is.Equal(ranked.TtlSeconds, int64(ttlSeconds))
	is.Equal(ranked.P95GapSeconds, *lateRenewal.RenewalGapSeconds)

	histogram := histogramFor(t, client, deviceID)
	breachedBuckets := 0
	for _, b := range histogram.Buckets {
		is.Equal(b.OkDeviceCount, 0)
		breachedBuckets += b.BreachedDeviceCount
	}
	is.Equal(breachedBuckets, 1) // the one bucket the late renewal landed in
}
