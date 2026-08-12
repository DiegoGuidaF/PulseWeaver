//go:build test

package device_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// tuningFloor mirrors device.tuningMinRenewals — the sample floor a device
// must clear to qualify for the fleet-wide tuning ranking.
const tuningFloor = 10

// tuningCap mirrors device.addressHistoryTuningLimit — the ranking's page size.
const tuningCap = 5

// seedRenewalTTL is the lease TTL every seedRenewalGaps device is given. The
// gaps those tests seed are chosen as ratios of it, so the band each renewal
// lands in is readable from the call site.
const seedRenewalTTL = 100

// seedRenewalGaps creates a device with an address-lease rule and one address,
// then backdates its event history so the gap between each consecutive pair of
// renewals is exactly the corresponding entry in gaps. The registration event
// anchors the sequence at start and never classifies — it has no previous
// renewal to measure against — so len(gaps) renewals are measurable.
func seedRenewalGaps(t *testing.T, srv *app.App, name, ip string, start time.Time, gaps ...time.Duration) ids.DeviceID {
	t.Helper()
	is := is.New(t)
	ctx := t.Context()

	dev, err := srv.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, srv), name, nil)
	is.NoErr(err)
	_, err = srv.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, seedRenewalTTL)
	is.NoErr(err)
	addr, _, err := srv.DeviceService.RegisterAddressActivity(ctx, dev.ID, ip, device.EventSourceHeartbeat)
	is.NoErr(err)

	db := srv.Database.DB()
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, start, addr.ID)
	is.NoErr(err)

	at := start
	for _, gap := range gaps {
		at = at.Add(gap)
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
			addr.ID, at,
		)
		is.NoErr(err)
	}
	return dev.ID
}

// deviceHistory fetches every address-history event for one device over the
// window, so a test can derive what the tuning readout should say from the
// rows the events endpoint actually returns rather than from the literals it
// seeded.
func deviceHistory(t *testing.T, client *httpapi.ClientWithResponses, deviceID ids.DeviceID, from, to time.Time) []httpapi.AddressHistoryEvent {
	t.Helper()
	is := is.New(t)

	deviceIDFilter := []httpapi.ID{deviceID.Int64()}
	limit := 200
	resp, err := client.GetAddressHistoryWithResponse(t.Context(), &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to, Limit: &limit,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.Total, len(resp.JSON200.Events)) // one page holds the whole window
	return resp.JSON200.Events
}

// nearestRankP95 is the textbook nearest-rank 95th percentile, implemented
// independently of the SQL so the two can be compared.
func nearestRankP95(gaps []int64) int64 {
	sorted := slices.Clone(gaps)
	slices.Sort(sorted)
	rank := int(math.Ceil(0.95 * float64(len(sorted))))
	return sorted[rank-1]
}

// TestHandler_GetAddressHistoryTuning_LowLateRateExcluded is the direct
// regression guard for the bug this bundle fixes: a device with many renewals
// and only a single bad outlier — the "388 renewals, 1% late" shape — must not
// appear in the ranking. With n=20 the nearest-rank 95th percentile is the
// nineteenth gap (80s), which does not exceed the 100s TTL, even though the
// single worst renewal (900s) is badly late. A rule keyed off the worst single
// event, rather than p95, would have flagged this device; p95 > ttl does not.
func TestHandler_GetAddressHistoryTuning_LowLateRateExcluded(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	gaps := make([]time.Duration, 0, 20)
	for range 19 {
		gaps = append(gaps, 80*time.Second)
	}
	gaps = append(gaps, 900*time.Second)

	t0 := time.Now().UTC().Add(-8 * time.Hour)
	deviceID := seedRenewalGaps(t, testServer, "low-late-rate-device", "10.9.1.1", t0, gaps...)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	// Derive the expected p95 from the gaps the events endpoint reports, so
	// the assertion measures the percentile logic rather than a hardcoded
	// literal that could drift from SQLite's julianday rounding.
	observed := make([]int64, 0, len(gaps))
	for _, e := range deviceHistory(t, client, deviceID, from, to) {
		if e.RenewalGapSeconds != nil {
			observed = append(observed, *e.RenewalGapSeconds)
		}
	}
	is.Equal(len(observed), len(gaps))
	wantP95 := nearestRankP95(observed)
	is.True(wantP95 <= int64(seedRenewalTTL)) // the case is discriminating: MAX() would flag it, p95 does not

	unfiltered, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(unfiltered.StatusCode(), http.StatusOK)
	is.Equal(unfiltered.JSON200.Total, 0)
	is.Equal(len(unfiltered.JSON200.Devices), 0)

	// device_id still reports the true p95 — the bypass path renders a
	// comfortable reading rather than hiding the number.
	id := deviceID.Int64()
	scoped, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to, DeviceId: &id})
	is.NoErr(err)
	is.Equal(scoped.StatusCode(), http.StatusOK)
	is.Equal(scoped.JSON200.Total, 1)
	is.Equal(len(scoped.JSON200.Devices), 1)
	is.Equal(scoped.JSON200.Devices[0].P95GapSeconds, wantP95)
}

// TestHandler_GetAddressHistoryTuning_SampleFloor is the direct regression
// guard for the confidence floor: a device with a p95 well above its TTL but
// fewer than min_renewals renewals must not qualify — a single renewal past
// its own maximum is not a rate. The same device crosses the floor and
// appears once one more renewal lands.
func TestHandler_GetAddressHistoryTuning_SampleFloor(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	const (
		breach = 200 * time.Second // 2x seedRenewalTTL — comfortably breached
		ip     = "10.9.0.1"
	)
	t0 := time.Now().UTC().Add(-6 * time.Hour)

	gaps := make([]time.Duration, tuningFloor-1) // one short of the floor
	for i := range gaps {
		gaps[i] = breach
	}
	deviceID := seedRenewalGaps(t, testServer, "floor-device", ip, t0, gaps...)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	below, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(below.StatusCode(), http.StatusOK)
	is.Equal(below.JSON200.Total, 0)
	is.Equal(len(below.JSON200.Devices), 0)

	// Cross the floor: one more renewal for the same device, still comfortably
	// breached since it lands hours after the backdated timeline.
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, deviceID, ip, device.EventSourceHeartbeat)
	is.NoErr(err)

	above, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(above.StatusCode(), http.StatusOK)
	is.Equal(above.JSON200.Total, 1)
	is.Equal(len(above.JSON200.Devices), 1)
	is.Equal(above.JSON200.Devices[0].DeviceId, deviceID.Int64())
	is.Equal(above.JSON200.Devices[0].RenewalCount, tuningFloor)
}

// TestHandler_GetAddressHistoryTuning_RankingCappedAndOrdered is the direct
// regression guard for §3's ordering and the top-5 cap: with more than five
// qualifying devices, total counts every one of them while devices holds only
// the worst five, ordered by how far the p95/TTL ratio exceeds 1 — not by
// late_renewal_count, which is identical across every device here.
func TestHandler_GetAddressHistoryTuning_RankingCappedAndOrdered(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	t0 := time.Now().UTC().Add(-6 * time.Hour)

	// Seven devices at seven distinct ratios above the TTL (1.2x .. 2.4x),
	// each with exactly the sample floor's worth of renewals, all breached —
	// late_renewal_count ties across every device, so only the ratio can
	// decide the order.
	byRatio := make([]ids.DeviceID, 7)
	for i := 1; i <= 7; i++ {
		gap := time.Duration(seedRenewalTTL+i*20) * time.Second
		gaps := make([]time.Duration, tuningFloor)
		for j := range gaps {
			gaps[j] = gap
		}
		byRatio[i-1] = seedRenewalGaps(t, testServer,
			fmt.Sprintf("ratio-device-%d", i), fmt.Sprintf("10.9.2.%d", i),
			t0, gaps...,
		)
	}

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	resp, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.Total, 7)
	is.Equal(resp.JSON200.MinRenewals, tuningFloor)

	ranked := resp.JSON200.Devices
	is.Equal(len(ranked), tuningCap) // capped — the two weakest ratios are cut

	// Worst ratio first: device 7 (2.4x) down to device 3 (1.6x).
	for i, d := range ranked {
		want := byRatio[6-i]
		is.Equal(d.DeviceId, want.Int64())
		is.Equal(d.RenewalCount, tuningFloor)
		is.Equal(d.LateRenewalCount, tuningFloor) // every renewal here is breached
	}
}

// TestHandler_GetAddressHistoryTuning_DeviceIDBypassesThreshold is the direct
// regression guard for §4: device_id returns that device's readout even when
// it fails both selection thresholds — too few renewals and a p95 nowhere
// near the TTL — since a reader already looking at one device needs no
// ranking to justify showing it.
func TestHandler_GetAddressHistoryTuning_DeviceIDBypassesThreshold(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	t0 := time.Now().UTC().Add(-time.Hour)
	deviceID := seedRenewalGaps(t, testServer, "comfortable-device", "10.9.3.1", t0, 10*time.Second, 10*time.Second)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	unfiltered, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(unfiltered.StatusCode(), http.StatusOK)
	is.Equal(unfiltered.JSON200.Total, 0) // fails both thresholds fleet-wide

	id := deviceID.Int64()
	scoped, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to, DeviceId: &id})
	is.NoErr(err)
	is.Equal(scoped.StatusCode(), http.StatusOK)
	is.Equal(scoped.JSON200.Total, 1)
	is.Equal(len(scoped.JSON200.Devices), 1)
	readout := scoped.JSON200.Devices[0]
	is.Equal(readout.DeviceId, deviceID.Int64())
	is.Equal(readout.RenewalCount, 2)
	is.Equal(readout.LateRenewalCount, 0)
	is.True(readout.P95GapSeconds <= readout.TtlSeconds) // a comfortable reading, not a recommendation
}

// TestHandler_GetAddressHistoryTuning_DeviceIDNoLeaseRule is the direct
// regression guard for §4's last clause: a device with no lease rule (so
// every event classifies unknown, and none is measurable) returns an empty
// devices list under device_id, not an error.
func TestHandler_GetAddressHistoryTuning_DeviceIDNoLeaseRule(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "no-lease-rule-device", nil)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.9.4.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	id := dev.ID.Int64()
	resp, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{DeviceId: &id})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.Total, 0)
	is.Equal(len(resp.JSON200.Devices), 0)
}

// TestHandler_GetAddressHistoryTuning_IgnoresColumnFilters is the direct
// regression guard for the acceptance criterion that column filters
// demonstrably do not reach this endpoint: GetAddressHistoryTuningParams has
// no filter fields at all, so a filter can only be exercised by injecting it
// onto the raw query — proving that even then the response is unaffected.
func TestHandler_GetAddressHistoryTuning_IgnoresColumnFilters(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	// Ten breached renewals: qualifies for the ranking outright. If a column
	// filter reached this endpoint, narrowing to ttl_risk=ok would empty the
	// qualifying set.
	t0 := time.Now().UTC().Add(-6 * time.Hour)
	gaps := make([]time.Duration, tuningFloor)
	for i := range gaps {
		gaps[i] = 200 * time.Second
	}
	deviceID := seedRenewalGaps(t, testServer, "filter-proof-device", "10.9.5.1", t0, gaps...)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	plainClient := testutils.NewAdminAPIClient(t, testServer)
	baseline, err := plainClient.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(baseline.StatusCode(), http.StatusOK)
	is.Equal(baseline.JSON200.Total, 1)

	filteredClient := testutils.NewAdminAPIClient(t, testServer,
		httpapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			q := req.URL.Query()
			q.Set("ttl_risk", "ok") // would empty the qualifying set if it reached the endpoint
			q.Set("event_kind", "refresh")
			req.URL.RawQuery = q.Encode()
			return nil
		}),
	)
	filtered, err := filteredClient.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(filtered.StatusCode(), http.StatusOK)
	is.Equal(filtered.JSON200.Total, baseline.JSON200.Total)
	is.Equal(len(filtered.JSON200.Devices), 1)
	is.Equal(filtered.JSON200.Devices[0].DeviceId, deviceID.Int64())
}

// TestHandler_GetAddressHistoryTuning_MalformedWindowIsBadRequest pins the 400
// the contract declares. The window params are parsed before the handler runs,
// so a value that is not an RFC3339 timestamp is rejected outright rather than
// falling back to the default window and answering with numbers over a period
// the caller did not ask for.
func TestHandler_GetAddressHistoryTuning_MalformedWindowIsBadRequest(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	client := testutils.NewAdminAPIClient(t, testServer,
		httpapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			q := req.URL.Query()
			q.Set("from", "yesterday")
			req.URL.RawQuery = q.Encode()
			return nil
		}),
	)

	resp, err := client.GetAddressHistoryTuningWithResponse(ctx, &httpapi.GetAddressHistoryTuningParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
	is.True(resp.JSON400 != nil) // the declared error shape, not an untyped body
}

// TestHandler_GetAddressHistoryHistogram_NoLongerCarriesAtRiskDevices is the
// direct regression guard for §1: the ranking moved wholesale to its own
// endpoint, no deprecation window, so the histogram response must not carry
// at_risk_devices at all — checked against the raw JSON, since the field no
// longer exists on the generated Go type either.
func TestHandler_GetAddressHistoryHistogram_NoLongerCarriesAtRiskDevices(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	var body map[string]json.RawMessage
	is.NoErr(json.Unmarshal(resp.Body, &body))
	_, hasBuckets := body["buckets"]
	_, hasAtRisk := body["at_risk_devices"]
	is.True(hasBuckets)
	is.True(!hasAtRisk)
}
