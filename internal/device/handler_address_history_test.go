//go:build test

package device_test

import (
	"context"
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

func TestHandler_GetAddressHistory_EventEnrichment(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "history-device", nil)
	is.NoErr(err)

	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.0.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	body := *resp.JSON200

	is.Equal(len(body.Events), 1)
	is.Equal(body.Total, 1)
	first := body.Events[0]
	is.Equal(first.Ip, "10.0.0.1")
	is.True(first.IsEnabled)
	is.Equal(first.DeviceName, "history-device")
	is.Equal(first.EventKind, httpapi.AddressEventKindCreated)
	is.Equal(first.TtlRisk, httpapi.Unknown) // no lease rule configured
	is.True(first.RenewalGapSeconds == nil)  // first renewal ever for the device
	is.True(first.TtlSeconds == nil)
}

// TestHandler_GetAddressHistory_FiltersMatchHistogram is the direct regression
// guard for the chart/table inconsistency the row-model change fixes: every
// filter must narrow both endpoints identically since they share one
// filtered, enriched read model.
func TestHandler_GetAddressHistory_FiltersMatchHistogram(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	const ttlSeconds = 100
	devA, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "filter-a", nil)
	is.NoErr(err)
	devB, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "filter-b", nil)
	is.NoErr(err)
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, devA.ID, ttlSeconds)
	is.NoErr(err)
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, devB.ID, ttlSeconds)
	is.NoErr(err)

	addrA, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, devA.ID, "10.1.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)
	addrB, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, devB.ID, "10.2.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	// Both devices renew far past their TTL, so both are at risk and the
	// device_id filter is what decides which of them the histogram describes.
	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-time.Hour)
	for _, addrID := range []ids.AddressID{addrA.ID, addrB.ID} {
		_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addrID)
		is.NoErr(err)
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
			addrID, t0.Add(3*ttlSeconds*time.Second),
		)
		is.NoErr(err)
	}

	deviceIDFilter := []httpapi.ID{devA.ID.Int64()}

	eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(eventsResp.StatusCode(), http.StatusOK)
	is.Equal(eventsResp.JSON200.Total, 2) // devA's creation and late renewal only, devB excluded

	histResp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(histResp.StatusCode(), http.StatusOK)

	is.Equal(len(histResp.JSON200.AtRiskDevices), 1) // devB is at risk too, and still excluded
	devA95 := histResp.JSON200.AtRiskDevices[0]
	is.Equal(devA95.DeviceId, devA.ID.Int64())

	// devA's two events are its creation (unknown — first renewal ever, no
	// gap to measure) and the late renewal (breached). renewal_count and
	// late_renewal_count only count the latter.
	is.Equal(devA95.RenewalCount, 1)
	is.Equal(devA95.LateRenewalCount, 1)
	is.Equal(devA95.TtlSeconds, int64(ttlSeconds))
	var wantGap int64
	for _, e := range eventsResp.JSON200.Events {
		if e.RenewalGapSeconds != nil {
			wantGap = *e.RenewalGapSeconds
		}
	}
	is.True(wantGap > 0)
	is.Equal(devA95.P95GapSeconds, wantGap) // a single renewal is trivially its own 95th percentile

	breached := 0
	for _, b := range histResp.JSON200.Buckets {
		breached += b.BreachedDeviceCount
	}
	is.Equal(breached, 1) // one device, one breaching bucket
}

// addressHistoryFilterCase is one filter set applied to both address-history
// endpoints. The fields mirror the generated params structs, which are distinct
// types per endpoint even though they carry the same filter columns.
// expectAtRisk records whether the filtered rows leave the device ranked in
// at_risk_devices (worst ttl_risk approaching or above). A filter narrowed to
// ok/unknown rows never ranks the device — the ranking still requires
// something actionable — even though the zero-filled buckets remain present
// either way.
type addressHistoryFilterCase struct {
	name         string
	source       []httpapi.AddressEventSource
	eventKind    []httpapi.AddressEventKind
	ttlRisk      []httpapi.TTLRisk
	ip           []string
	expectAtRisk bool
}

// TestHandler_GetAddressHistory_DerivedFiltersMatchHistogram extends the
// device_id parity check above to the *derived* columns. device_id exists on the
// base tables, so a histogram grouping over the raw joins would still satisfy
// that test; event_kind and ttl_risk exist only on the enriched derived table,
// so they are what actually prove both endpoints filter the same read model.
func TestHandler_GetAddressHistory_DerivedFiltersMatchHistogram(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "derived-filters", nil)
	is.NoErr(err)

	const ttlSeconds = 100
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.5.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	// Backdate the creation and append a timeline covering every derived value:
	// created (no prior renewal → unknown), a refresh well inside the TTL → ok,
	// a routine expiry → not a renewal, so unknown, and a late heartbeat whose
	// gap since the previous *renewal* is 5.5× the TTL → breached.
	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-time.Hour)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addr.ID)
	is.NoErr(err)
	for _, e := range []struct {
		enabled int
		source  string
		at      time.Time
	}{
		{1, "heartbeat", t0.Add(50 * time.Second)},
		{0, "expiry", t0.Add(200 * time.Second)},
		{1, "heartbeat", t0.Add(600 * time.Second)},
	} {
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, ?, ?, ?)`,
			addr.ID, e.enabled, e.source, e.at,
		)
		is.NoErr(err)
	}

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}

	// Baseline: unfiltered, this device is ranked and charted. Every empty
	// expectation below is therefore attributable to the filter reaching the
	// histogram's read model, not to a fixture that was never at risk.
	baseline, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(baseline.StatusCode(), http.StatusOK)
	is.Equal(len(baseline.JSON200.AtRiskDevices), 1)
	is.True(len(baseline.JSON200.Buckets) > 0)

	cases := []addressHistoryFilterCase{
		{name: "event_kind refresh only", eventKind: []httpapi.AddressEventKind{httpapi.AddressEventKindRefresh}},
		{name: "event_kind state changes", eventKind: []httpapi.AddressEventKind{
			httpapi.AddressEventKindCreated, httpapi.AddressEventKindEnabled, httpapi.AddressEventKindDisabled,
		}, expectAtRisk: true},
		{name: "ttl_risk breached", ttlRisk: []httpapi.TTLRisk{httpapi.Breached}, expectAtRisk: true},
		{name: "ttl_risk unknown", ttlRisk: []httpapi.TTLRisk{httpapi.Unknown}},
		{name: "ttl_risk ok", ttlRisk: []httpapi.TTLRisk{httpapi.Ok}},
		{name: "source expiry", source: []httpapi.AddressEventSource{httpapi.Expiry}},
		{name: "ip", ip: []string{"10.5.0.1"}, expectAtRisk: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)

			eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
				DeviceId: &deviceIDFilter, From: &from, To: &to,
				Source: sliceParam(tc.source), EventKind: sliceParam(tc.eventKind),
				TtlRisk: sliceParam(tc.ttlRisk), Ip: sliceParam(tc.ip),
			})
			is.NoErr(err)
			is.Equal(eventsResp.StatusCode(), http.StatusOK)

			histResp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
				DeviceId: &deviceIDFilter, From: &from, To: &to,
				Source: sliceParam(tc.source), EventKind: sliceParam(tc.eventKind),
				TtlRisk: sliceParam(tc.ttlRisk), Ip: sliceParam(tc.ip),
			})
			is.NoErr(err)
			is.Equal(histResp.StatusCode(), http.StatusOK)

			// Guard against passing vacuously: every case above must match rows,
			// otherwise an empty histogram would "prove" agreement for a filter
			// that never ran.
			is.True(eventsResp.JSON200.Total > 0)
			is.Equal(len(eventsResp.JSON200.Events), eventsResp.JSON200.Total)

			wantRenewals, wantLate := 0, 0
			for _, e := range eventsResp.JSON200.Events {
				if e.TtlRisk != httpapi.Unknown {
					wantRenewals++
				}
				if e.TtlRisk == httpapi.Breached {
					wantLate++
				}
			}

			// Buckets are zero-filled and contiguous regardless of whether
			// anything in them is at risk. A bucket counts a device once
			// however often it renewed inside it, so with a single device in
			// scope the four bands summed across every bucket must reproduce
			// the number of buckets its filtered renewals land in — which is
			// the renewal count only when no two share a bucket.
			is.True(len(histResp.JSON200.Buckets) > 0)
			bandSum := 0
			for _, b := range histResp.JSON200.Buckets {
				bandSum += b.OkDeviceCount + b.ApproachingDeviceCount + b.CriticalDeviceCount + b.BreachedDeviceCount
			}
			is.Equal(bandSum, renewalBucketCount(eventsResp.JSON200.Events, histResp.JSON200.Buckets))

			if !tc.expectAtRisk {
				is.Equal(len(histResp.JSON200.AtRiskDevices), 0)
				return
			}

			is.Equal(len(histResp.JSON200.AtRiskDevices), 1)
			is.Equal(histResp.JSON200.AtRiskDevices[0].RenewalCount, wantRenewals)
			is.Equal(histResp.JSON200.AtRiskDevices[0].LateRenewalCount, wantLate)
		})
	}
}

// renewalBucketCount reports how many of the given histogram buckets contain
// at least one measurable renewal from events. It reads the bucket boundaries
// off the response rather than recomputing the granularity, so it stays
// correct whichever rung of the ladder the window lands on.
func renewalBucketCount(events []httpapi.AddressHistoryEvent, buckets []httpapi.AddressHistoryBucket) int {
	occupied := 0
	for i, b := range buckets {
		start := time.Time(b.Timestamp)
		end := time.Time(buckets[len(buckets)-1].Timestamp).Add(time.Hour * 24 * 366) // open-ended last bucket
		if i+1 < len(buckets) {
			end = time.Time(buckets[i+1].Timestamp)
		}
		for _, e := range events {
			at := time.Time(e.Timestamp)
			if e.TtlRisk != httpapi.Unknown && !at.Before(start) && at.Before(end) {
				occupied++
				break
			}
		}
	}
	return occupied
}

// sliceParam adapts a filter case's value slice to the pointer-to-slice the
// generated params structs use, mapping "no values" to "param not supplied".
func sliceParam[T any](values []T) *[]T {
	if len(values) == 0 {
		return nil
	}
	return &values
}

// TestHandler_GetAddressHistory_RoutineExpiryNotBreached is the direct
// regression guard for the defect described in task 08 §"Why the row model
// changes" point 3: an expiry event must never itself be classified as a TTL
// breach — it terminates a lease, it does not renew one, so it carries no
// renewal_gap_seconds to score against the TTL at all.
func TestHandler_GetAddressHistory_RoutineExpiryNotBreached(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "expiry-device", nil)
	is.NoErr(err)

	const ttlSeconds = 60
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.0.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	err = testServer.DeviceService.DisableAddresses(ctx, []ids.AddressID{addr.ID}, device.EventSourceExpiry)
	is.NoErr(err)

	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	var expiryEvent *httpapi.AddressHistoryEvent
	for i, e := range resp.JSON200.Events {
		if e.Source == httpapi.Expiry {
			expiryEvent = &resp.JSON200.Events[i]
		}
	}
	is.True(expiryEvent != nil)
	is.True(expiryEvent.RenewalGapSeconds == nil)  // expiry rows never renew the lease
	is.Equal(expiryEvent.TtlRisk, httpapi.Unknown) // and are therefore never scored — never breached
}

// TestHandler_GetAddressHistory_BreachedAgreesWithExpiryEvent is the direct
// regression guard for task 08 §7 bullet 4: a breached row means the address
// really did lapse, so a corresponding expiry event must exist for the
// device. Timestamps are backdated directly so the TTL ratio can be
// deterministic without a real wall-clock wait.
func TestHandler_GetAddressHistory_BreachedAgreesWithExpiryEvent(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "breach-device", nil)
	is.NoErr(err)

	const ttlSeconds = 100
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.0.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-20 * time.Minute)

	// Backdate the initial creation event — the first renewal.
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addr.ID)
	is.NoErr(err)

	// Expiry fires right at the TTL boundary — a routine termination, not a renewal.
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 0, 'expiry', ?)`,
		addr.ID, t0.Add(ttlSeconds*time.Second),
	)
	is.NoErr(err)

	// The device only comes back well past the TTL — this renewal is breached.
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
		addr.ID, t0.Add(4*ttlSeconds*time.Second),
	)
	is.NoErr(err)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
		From:     &from,
		To:       &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	events := resp.JSON200.Events

	hasExpiryEvent := false
	breachedCount := 0
	for _, e := range events {
		if e.Source == httpapi.Expiry && !e.IsEnabled {
			hasExpiryEvent = true
		}
		if e.TtlRisk == httpapi.Breached {
			breachedCount++
		}
	}
	is.True(hasExpiryEvent)     // the address really did lapse
	is.True(breachedCount >= 1) // the late renewal is classified breached
}

// TestHandler_GetAddressHistory_EventKindStableAcrossWindow is the direct
// regression guard for task 08 §7 bullet 5: event_kind is an unbounded,
// per-address comparison, so a row's classification must not change when its
// true predecessor falls outside the queried window.
func TestHandler_GetAddressHistory_EventKindStableAcrossWindow(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "stable-device", nil)
	is.NoErr(err)
	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.0.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)
	_, err = testServer.DeviceService.DisableAddress(ctx, dev.ID, addr.ID)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-time.Hour)
	t1 := t0.Add(time.Minute)
	var createdID, disabledID int64
	is.NoErr(db.GetContext(ctx, &createdID, `SELECT id FROM address_events WHERE address_id = ? AND is_enabled = 1`, addr.ID))
	is.NoErr(db.GetContext(ctx, &disabledID, `SELECT id FROM address_events WHERE address_id = ? AND is_enabled = 0`, addr.ID))
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE id = ?`, t0, createdID)
	is.NoErr(err)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE id = ?`, t1, disabledID)
	is.NoErr(err)

	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}

	// Wide window: both events visible.
	wideFrom := t0.Add(-time.Minute)
	wideTo := t1.Add(time.Minute)
	wide, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, From: &wideFrom, To: &wideTo,
	})
	is.NoErr(err)
	is.Equal(wide.StatusCode(), http.StatusOK)
	is.Equal(len(wide.JSON200.Events), 2)

	// Narrow window: excludes the "created" event, keeps only "disabled".
	narrowFrom := t0.Add(30 * time.Second)
	narrowTo := t1.Add(time.Minute)
	narrow, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, From: &narrowFrom, To: &narrowTo,
	})
	is.NoErr(err)
	is.Equal(narrow.StatusCode(), http.StatusOK)
	is.Equal(len(narrow.JSON200.Events), 1)

	// The surviving row must still classify as "disabled" — its predecessor
	// falling outside the window must not reclassify it as "created".
	is.Equal(narrow.JSON200.Events[0].Id, disabledID)
	is.Equal(narrow.JSON200.Events[0].EventKind, httpapi.AddressEventKindDisabled)

	var wideDisabled *httpapi.AddressHistoryEvent
	for i, e := range wide.JSON200.Events {
		if e.Id == disabledID {
			wideDisabled = &wide.JSON200.Events[i]
		}
	}
	is.True(wideDisabled != nil)
	is.Equal(wideDisabled.EventKind, narrow.JSON200.Events[0].EventKind)
}

func TestHandler_GetAddressHistory_Pagination(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "pagination-dev", nil)
	is.NoErr(err)

	for i := range 5 {
		_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, fmt.Sprintf("10.0.0.%d", i+1), device.EventSourceHeartbeat)
		is.NoErr(err)
	}

	limit := 2
	page1Resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		Limit: &limit,
	})
	is.NoErr(err)
	is.Equal(page1Resp.StatusCode(), http.StatusOK)
	page1 := *page1Resp.JSON200
	is.Equal(len(page1.Events), 2)
	is.True(page1.NextCursor != nil)
	is.True(page1.Total >= 5)

	page2Resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		Limit:    &limit,
		BeforeId: page1.NextCursor,
	})
	is.NoErr(err)
	is.Equal(page2Resp.StatusCode(), http.StatusOK)
	page2 := *page2Resp.JSON200
	is.Equal(len(page2.Events), 2)

	for _, e := range page2.Events {
		is.True(e.Id < *page1.NextCursor)
	}
}

func TestHandler_GetAddressHistory_InvalidFilterOperator(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)

	// contains/not_contains are not allowed on device_id — exercise the
	// handler's 400 path via a raw query rewrite (the typed operator param
	// only accepts what AddressHistoryFilterOperator enumerates).
	client := testutils.NewAdminAPIClient(t, testServer,
		httpapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			q := req.URL.Query()
			q.Set("device_id", "1")
			q.Set("device_id_op", "contains")
			req.URL.RawQuery = q.Encode()
			return nil
		}),
	)
	resp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

// TestHandler_GetAddressHistoryHistogram_WorstRiskPerBucket is the direct
// regression guard for the four bands partitioning a bucket: a device with
// events at several risk levels in the same bucket — including several ok
// renewals — is counted once, under its worst level only, and every other
// bucket in the window is still present, zero-filled.
func TestHandler_GetAddressHistoryHistogram_WorstRiskPerBucket(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "worst-risk-device", nil)
	is.NoErr(err)

	const ttlSeconds = 100
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.6.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	// t0 sits on a 5-minute boundary so every offset below stays inside one
	// 5-minute bucket regardless of wall-clock alignment.
	db := testServer.Database.DB()
	t0 := time.Now().UTC().Truncate(5 * time.Minute).Add(-2 * time.Hour)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addr.ID)
	is.NoErr(err)
	for _, e := range []struct {
		at time.Time
	}{
		{t0.Add(10 * time.Second)},  // gap 10s / ttl 100s = 0.1 → ok
		{t0.Add(20 * time.Second)},  // gap 10s from the previous renewal → ok
		{t0.Add(110 * time.Second)}, // gap 90s / ttl 100s = 0.9 → approaching
		{t0.Add(205 * time.Second)}, // gap 95s / ttl 100s = 0.95 → critical
	} {
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
			addr.ID, e.at,
		)
		is.NoErr(err)
	}

	from := t0.Add(-time.Minute)
	to := t0.Add(10 * time.Minute)
	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	buckets := resp.JSON200.Buckets

	is.Equal(len(buckets), 4) // t0-5min, t0, t0+5min, t0+10min — zero-filled, contiguous
	var hit *httpapi.AddressHistoryBucket
	for i, b := range buckets {
		if time.Time(b.Timestamp).Equal(t0) {
			hit = &buckets[i]
			continue
		}
		is.Equal(b.OkDeviceCount, 0)
		is.Equal(b.ApproachingDeviceCount, 0)
		is.Equal(b.CriticalDeviceCount, 0)
		is.Equal(b.BreachedDeviceCount, 0)
	}
	is.True(hit != nil)
	is.Equal(hit.CriticalDeviceCount, 1) // worst risk wins over its own ok and approaching events
	is.Equal(hit.OkDeviceCount, 0)       // not double-counted in the lower bands
	is.Equal(hit.ApproachingDeviceCount, 0)
	is.Equal(hit.BreachedDeviceCount, 0)

	is.Equal(len(resp.JSON200.AtRiskDevices), 1)
	is.Equal(resp.JSON200.AtRiskDevices[0].DeviceId, dev.ID.Int64())
	is.Equal(resp.JSON200.AtRiskDevices[0].WorstRisk, httpapi.Critical)
}

// TestHandler_GetAddressHistoryHistogram_ContiguousBucketsAcrossGap is the
// direct regression guard for the reason this response shape exists: every
// bucket across the window is present, even one nothing renewed in, so the
// caller never has to reconstruct a gap it cannot see.
func TestHandler_GetAddressHistoryHistogram_ContiguousBucketsAcrossGap(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "gap-device", nil)
	is.NoErr(err)

	const ttlSeconds = 10000
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.7.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	t0 := time.Now().UTC().Truncate(time.Hour).Add(-6 * time.Hour)
	db := testServer.Database.DB()
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0.Add(-time.Hour), addr.ID)
	is.NoErr(err)

	// Renewals in the first and fourth hourly buckets, nothing in between.
	for _, at := range []time.Time{
		t0.Add(10 * time.Minute),
		t0.Add(3*time.Hour + 50*time.Minute),
	} {
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
			addr.ID, at,
		)
		is.NoErr(err)
	}

	from := t0
	to := t0.Add(4 * time.Hour)
	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	buckets := resp.JSON200.Buckets

	is.Equal(len(buckets), 5) // t0 .. t0+4h at hourly granularity
	is.True(time.Time(buckets[0].Timestamp).Equal(from))
	is.True(time.Time(buckets[len(buckets)-1].Timestamp).Equal(to))

	// Interior gap: the second and third buckets (t0+1h, t0+2h) see no
	// renewal at all, so all four bands read zero rather than being absent.
	for _, b := range buckets[1:3] {
		is.Equal(b.OkDeviceCount, 0)
		is.Equal(b.ApproachingDeviceCount, 0)
		is.Equal(b.CriticalDeviceCount, 0)
		is.Equal(b.BreachedDeviceCount, 0)
	}

	is.Equal(buckets[0].OkDeviceCount, 1) // the early renewal, well inside the TTL
}

// TestHandler_GetAddressHistoryHistogram_UnknownOnlyBucketIsAllZero is the
// direct regression guard against reading an unknown-only bucket as healthy:
// a bucket where every event has no measurable gap must show every band,
// including ok, at zero — it is not evidence the device renewed comfortably.
func TestHandler_GetAddressHistoryHistogram_UnknownOnlyBucketIsAllZero(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "unknown-only-device", nil)
	is.NoErr(err)

	// No lease rule configured: every event classifies unknown regardless of
	// the gap between them.
	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.7.2.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-time.Hour)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addr.ID)
	is.NoErr(err)
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
		addr.ID, t0.Add(30*time.Second),
	)
	is.NoErr(err)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}

	eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(eventsResp.StatusCode(), http.StatusOK)
	is.Equal(eventsResp.JSON200.Total, 2)
	for _, e := range eventsResp.JSON200.Events {
		is.Equal(e.TtlRisk, httpapi.Unknown)
	}

	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(resp.JSON200.AtRiskDevices), 0)
	is.True(len(resp.JSON200.Buckets) > 0) // zero-filled, contiguous — not absent
	for _, b := range resp.JSON200.Buckets {
		is.Equal(b.OkDeviceCount, 0)
		is.Equal(b.ApproachingDeviceCount, 0)
		is.Equal(b.CriticalDeviceCount, 0)
		is.Equal(b.BreachedDeviceCount, 0)
	}
}

// TestHandler_GetAddressHistoryHistogram_OkNeverRanksButCountsInBuckets is
// the direct regression guard for restoring the healthy band: ok renewals
// count toward ok_device_count in the buckets they fall in, but a device
// whose worst risk is ok still never appears in the ranking, however many
// such renewals it has.
func TestHandler_GetAddressHistoryHistogram_OkNeverRanksButCountsInBuckets(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "healthy-device", nil)
	is.NoErr(err)

	const ttlSeconds = 1000
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.6.0.2", device.EventSourceHeartbeat)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-time.Hour)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addr.ID)
	is.NoErr(err)
	// gap 10s / ttl 1000s = 0.01 → ok, well clear of any risk threshold.
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
		addr.ID, t0.Add(10*time.Second),
	)
	is.NoErr(err)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(eventsResp.StatusCode(), http.StatusOK)
	is.Equal(eventsResp.JSON200.Total, 2) // create (unknown) + ok renewal

	is.Equal(len(resp.JSON200.AtRiskDevices), 0) // never ranked — worst risk is ok

	is.True(len(resp.JSON200.Buckets) > 0) // zero-filled, contiguous — never absent
	okSum, atRiskSum := 0, 0
	for _, b := range resp.JSON200.Buckets {
		okSum += b.OkDeviceCount
		atRiskSum += b.ApproachingDeviceCount + b.CriticalDeviceCount + b.BreachedDeviceCount
	}
	is.Equal(okSum, 1)     // the one ok renewal is the denominator this response restores
	is.Equal(atRiskSum, 0) // …but it never contributes to an at-risk band
}

// TestHandler_GetAddressHistoryHistogram_P95GapSeconds is the direct
// regression guard for the nearest-rank calculation: with few renewals (n=1
// or n=4 here), the 95th-percentile gap must equal the largest gap observed.
func TestHandler_GetAddressHistoryHistogram_P95GapSeconds(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	const ttlSeconds = 5

	devOne, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "p95-n1-device", nil)
	is.NoErr(err)
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, devOne.ID, ttlSeconds)
	is.NoErr(err)
	addrOne, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, devOne.ID, "10.7.1.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	devFour, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "p95-n4-device", nil)
	is.NoErr(err)
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, devFour.ID, ttlSeconds)
	is.NoErr(err)
	addrFour, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, devFour.ID, "10.7.1.2", device.EventSourceHeartbeat)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-2 * time.Hour)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addrOne.ID)
	is.NoErr(err)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addrFour.ID)
	is.NoErr(err)

	// devOne: a single renewal, gap 30s.
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
		addrOne.ID, t0.Add(30*time.Second),
	)
	is.NoErr(err)

	// devFour: four renewals with gaps 10s, 20s, 30s, 40s — nearest-rank over
	// n=4 always resolves to the maximum.
	at := t0
	for _, gap := range []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second, 40 * time.Second} {
		at = at.Add(gap)
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
			addrFour.ID, at,
		)
		is.NoErr(err)
	}

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	for _, tc := range []struct {
		name     string
		deviceID ids.DeviceID
	}{
		{"n=1", devOne.ID},
		{"n=4", devFour.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			deviceIDFilter := []httpapi.ID{tc.deviceID.Int64()}

			// The expected p95 is read back from the events endpoint's own
			// renewal_gap_seconds rather than hardcoded, so the assertion
			// tracks the actual gaps computed for these rows.
			eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
				DeviceId: &deviceIDFilter, From: &from, To: &to,
			})
			is.NoErr(err)
			is.Equal(eventsResp.StatusCode(), http.StatusOK)
			var maxGap int64
			for _, e := range eventsResp.JSON200.Events {
				if e.RenewalGapSeconds != nil && *e.RenewalGapSeconds > maxGap {
					maxGap = *e.RenewalGapSeconds
				}
			}
			is.True(maxGap > 0)

			resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
				DeviceId: &deviceIDFilter, From: &from, To: &to,
			})
			is.NoErr(err)
			is.Equal(resp.StatusCode(), http.StatusOK)
			is.Equal(len(resp.JSON200.AtRiskDevices), 1)
			is.Equal(resp.JSON200.AtRiskDevices[0].P95GapSeconds, maxGap) // n<=4 always resolves to the maximum
		})
	}
}

// TestHandler_GetAddressHistoryHistogram_RankingRespectsFilters is the direct
// regression guard for task 08b item 5: the at-risk ranking narrows exactly
// like the events list and the buckets, since all three read the same
// filtered, enriched row set.
func TestHandler_GetAddressHistoryHistogram_RankingRespectsFilters(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	devCritical, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "critical-device", nil)
	is.NoErr(err)
	devBreached, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "breached-device", nil)
	is.NoErr(err)

	const ttlSeconds = 100
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, devCritical.ID, ttlSeconds)
	is.NoErr(err)
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, devBreached.ID, ttlSeconds)
	is.NoErr(err)

	addrCritical, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, devCritical.ID, "10.6.0.3", device.EventSourceHeartbeat)
	is.NoErr(err)
	addrBreached, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, devBreached.ID, "10.6.0.4", device.EventSourceHeartbeat)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-time.Hour)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addrCritical.ID)
	is.NoErr(err)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addrBreached.ID)
	is.NoErr(err)
	// gap 95s / ttl 100s = 0.95 → critical.
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
		addrCritical.ID, t0.Add(95*time.Second),
	)
	is.NoErr(err)
	// gap 150s / ttl 100s = 1.5 → breached.
	_, err = db.ExecContext(ctx,
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
		addrBreached.ID, t0.Add(150*time.Second),
	)
	is.NoErr(err)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	unfiltered, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(unfiltered.StatusCode(), http.StatusOK)
	is.Equal(len(unfiltered.JSON200.AtRiskDevices), 2)
	is.Equal(unfiltered.JSON200.AtRiskDevices[0].DeviceId, devBreached.ID.Int64()) // breached ranks above critical
	is.Equal(unfiltered.JSON200.AtRiskDevices[0].WorstRisk, httpapi.Breached)
	is.Equal(unfiltered.JSON200.AtRiskDevices[1].DeviceId, devCritical.ID.Int64())
	is.Equal(unfiltered.JSON200.AtRiskDevices[1].WorstRisk, httpapi.Critical)

	deviceIDFilter := []httpapi.ID{devCritical.ID.Int64()}
	filtered, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(filtered.StatusCode(), http.StatusOK)
	is.Equal(len(filtered.JSON200.AtRiskDevices), 1)
	is.Equal(filtered.JSON200.AtRiskDevices[0].DeviceId, devCritical.ID.Int64())
}

// TestHandler_GetAddressHistoryHistogram_RankingStableUnderCursor is the
// direct regression guard for the at-risk ranking reflecting a device's
// totals across the whole window, not one page of them — the histogram
// endpoint carries no cursor parameter at all, so it cannot be scoped by the
// events endpoint's pagination position.
func TestHandler_GetAddressHistoryHistogram_RankingStableUnderCursor(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "many-events-device", nil)
	is.NoErr(err)

	const ttlSeconds = 10
	_, err = testServer.RuleService.EnableDeviceAddressLeaseRule(ctx, dev.ID, ttlSeconds)
	is.NoErr(err)

	addr, _, err := testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, "10.6.0.5", device.EventSourceHeartbeat)
	is.NoErr(err)

	db := testServer.Database.DB()
	t0 := time.Now().UTC().Add(-20 * time.Minute)
	_, err = db.ExecContext(ctx, `UPDATE address_events SET created_at = ? WHERE address_id = ?`, t0, addr.ID)
	is.NoErr(err)
	// Each renewal's gap (100s) against a 10s TTL is a 10x ratio — every one breached.
	for i := 1; i <= 5; i++ {
		_, err = db.ExecContext(ctx,
			`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 1, 'heartbeat', ?)`,
			addr.ID, t0.Add(time.Duration(i*100)*time.Second),
		)
		is.NoErr(err)
	}

	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	limit := 2
	page1, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter, Limit: &limit,
	})
	is.NoErr(err)
	is.Equal(page1.StatusCode(), http.StatusOK)
	is.True(page1.JSON200.NextCursor != nil) // events endpoint is mid-page

	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	is.Equal(len(resp.JSON200.AtRiskDevices), 1)
	device := resp.JSON200.AtRiskDevices[0]
	is.Equal(device.DeviceId, dev.ID.Int64())
	is.Equal(device.WorstRisk, httpapi.Breached)
	is.Equal(device.RenewalCount, 5)     // the 5 heartbeats, unaffected by the events page size — creation has no gap to measure
	is.Equal(device.LateRenewalCount, 5) // every renewal breached its TTL
	is.Equal(device.TtlSeconds, int64(ttlSeconds))

	// Every renewal has the same ~100s gap, so the maximum observed on the
	// full (unpaged) events response is what p95 must resolve to.
	full, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(full.StatusCode(), http.StatusOK)
	var maxGap int64
	for _, e := range full.JSON200.Events {
		if e.RenewalGapSeconds != nil && *e.RenewalGapSeconds > maxGap {
			maxGap = *e.RenewalGapSeconds
		}
	}
	is.True(maxGap > 0)
	is.Equal(device.P95GapSeconds, maxGap)

	// The buckets are charted for the same reason: the device shows as
	// breached across the window, not only on the page being viewed. They
	// are zero-filled and contiguous, so most of the default 24h window's
	// hourly buckets carry no breach at all — only the one (or, if the
	// events happen to straddle an hour boundary, two adjacent) bucket(s)
	// the renewals actually fall in.
	is.True(len(resp.JSON200.Buckets) > 0)
	totalBreached := 0
	for _, b := range resp.JSON200.Buckets {
		is.Equal(b.OkDeviceCount, 0)
		is.Equal(b.ApproachingDeviceCount, 0)
		is.Equal(b.CriticalDeviceCount, 0)
		is.True(b.BreachedDeviceCount == 0 || b.BreachedDeviceCount == 1)
		totalBreached += b.BreachedDeviceCount
	}
	is.True(totalBreached >= 1)
}

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
// window, so a test can derive what the ranking should say from the rows the
// events endpoint actually returns rather than from the literals it seeded.
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

// TestHandler_GetAddressHistoryHistogram_P95BelowMaximum is the direct
// regression guard for the nearest-rank selection itself. Every other p95 test
// runs at a renewal count where the percentile collapses onto the maximum, so
// they would pass against a plain MAX(); this one uses twenty renewals with a
// single outlier, where nearest-rank must discard the outlier and return the
// nineteenth gap instead.
func TestHandler_GetAddressHistoryHistogram_P95BelowMaximum(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	// Nineteen renewals in the approaching band plus one long outlier: with
	// n=20 the 95th percentile is the nineteenth gap, not the twentieth.
	gaps := make([]time.Duration, 0, 20)
	for range 19 {
		gaps = append(gaps, 80*time.Second)
	}
	gaps = append(gaps, 900*time.Second)

	t0 := time.Now().UTC().Add(-6 * time.Hour)
	deviceID := seedRenewalGaps(t, testServer, "p95-outlier-device", "10.8.0.1", t0, gaps...)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)

	// Derive the expectation from the gaps the events endpoint reports, so the
	// assertion measures the percentile logic rather than SQLite's julianday
	// rounding.
	observed := make([]int64, 0, len(gaps))
	for _, e := range deviceHistory(t, client, deviceID, from, to) {
		if e.RenewalGapSeconds != nil {
			observed = append(observed, *e.RenewalGapSeconds)
		}
	}
	is.Equal(len(observed), len(gaps)) // every seeded renewal has a measurable gap

	wantP95 := nearestRankP95(observed)
	maxGap := slices.Max(observed)
	is.True(wantP95 < maxGap) // the case is discriminating: MAX() would be wrong here

	deviceIDFilter := []httpapi.ID{deviceID.Int64()}
	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter, From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(len(resp.JSON200.AtRiskDevices), 1)
	is.Equal(resp.JSON200.AtRiskDevices[0].P95GapSeconds, wantP95)
}

// TestHandler_GetAddressHistoryHistogram_CountsAgreeWithEventsEndpoint is the
// executable form of the ADR-009 §4 observability invariant: for the same
// filters and window, a ranked device's renewal_count is exactly its
// non-unknown row count on the events endpoint, and its late_renewal_count is
// exactly its breached row count. Without this the histogram's counters could
// drift from the table the user is reading beside them and nothing would say
// so.
func TestHandler_GetAddressHistoryHistogram_CountsAgreeWithEventsEndpoint(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	// One renewal in each band, then two breaches — plus an expiry row, which
	// renews nothing and must stay out of the denominator entirely.
	t0 := time.Now().UTC().Add(-3 * time.Hour)
	deviceID := seedRenewalGaps(t, testServer, "mixed-band-device", "10.8.1.1", t0,
		10*time.Second,  // 0.10 × TTL → ok
		80*time.Second,  // 0.80 × TTL → approaching
		92*time.Second,  // 0.92 × TTL → critical
		150*time.Second, // 1.50 × TTL → breached
		200*time.Second, // 2.00 × TTL → breached
	)

	addrs, err := testServer.DeviceService.GetEnabledAddressesForDevice(ctx, deviceID)
	is.NoErr(err)
	is.Equal(len(addrs), 1)
	err = testServer.DeviceService.DisableAddresses(ctx, []ids.AddressID{addrs[0].ID}, device.EventSourceExpiry)
	is.NoErr(err)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	deviceIDFilter := []httpapi.ID{deviceID.Int64()}

	assertCountsAgree := func(t *testing.T, params *httpapi.GetAddressHistoryHistogramParams, events []httpapi.AddressHistoryEvent) httpapi.AddressHistoryAtRiskDevice {
		t.Helper()
		is := is.New(t)

		renewals, late := 0, 0
		for _, e := range events {
			if e.TtlRisk != httpapi.Unknown {
				renewals++
			}
			if e.TtlRisk == httpapi.Breached {
				late++
			}
		}

		resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, params)
		is.NoErr(err)
		is.Equal(resp.StatusCode(), http.StatusOK)
		is.Equal(len(resp.JSON200.AtRiskDevices), 1)

		ranked := resp.JSON200.AtRiskDevices[0]
		is.Equal(ranked.RenewalCount, renewals) // ADR-009 §4: same filtered rows, same denominator
		is.Equal(ranked.LateRenewalCount, late) // …and the same breached numerator
		is.Equal(ranked.TtlSeconds, int64(seedRenewalTTL))
		is.True(ranked.P95GapSeconds > 0) // a ranked device always resolves a gap
		return ranked
	}

	t.Run("unfiltered", func(t *testing.T) {
		is := is.New(t)
		events := deviceHistory(t, client, deviceID, from, to)

		ranked := assertCountsAgree(t, &httpapi.GetAddressHistoryHistogramParams{
			DeviceId: &deviceIDFilter, From: &from, To: &to,
		}, events)

		is.Equal(ranked.RenewalCount, 5)     // the five heartbeat renewals
		is.Equal(ranked.LateRenewalCount, 2) // …of which two arrived past the TTL
		is.Equal(ranked.WorstRisk, httpapi.Breached)
		is.Equal(len(events), 7) // registration + 5 renewals + the expiry row
	})

	// Narrowing to the breached band must move both counters together: the
	// invariant is stated per filter set, not only for the unfiltered case.
	t.Run("filtered to breached", func(t *testing.T) {
		is := is.New(t)

		riskFilter := []httpapi.TTLRisk{httpapi.Breached}
		limit := 200
		eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
			DeviceId: &deviceIDFilter, TtlRisk: &riskFilter, From: &from, To: &to, Limit: &limit,
		})
		is.NoErr(err)
		is.Equal(eventsResp.StatusCode(), http.StatusOK)
		is.Equal(eventsResp.JSON200.Total, 2)

		ranked := assertCountsAgree(t, &httpapi.GetAddressHistoryHistogramParams{
			DeviceId: &deviceIDFilter, TtlRisk: &riskFilter, From: &from, To: &to,
		}, eventsResp.JSON200.Events)

		is.Equal(ranked.RenewalCount, 2)     // every surviving row is a breached renewal
		is.Equal(ranked.LateRenewalCount, 2) // …so the two counters coincide under this filter
	})
}

// TestHandler_GetAddressHistoryHistogram_RankingTieBreaks pins the ranking's
// full sort key. Worst risk alone leaves ties, and the reader is meant to work
// down the list top-first, so the order among equally-risky devices has to be
// the one the API documents: more late renewals first, then more renewals, then
// device id.
func TestHandler_GetAddressHistoryHistogram_RankingTieBreaks(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	const (
		breach  = 200 * time.Second // 2.00 × TTL → breached
		healthy = 10 * time.Second  // 0.10 × TTL → ok
	)

	// All three devices peak at breached, so risk_rank cannot separate them.
	t0 := time.Now().UTC().Add(-3 * time.Hour)
	mostLate := seedRenewalGaps(t, testServer, "tie-most-late", "10.8.2.1", t0, breach, breach, breach)
	moreRenewals := seedRenewalGaps(t, testServer, "tie-more-renewals", "10.8.2.2", t0, breach, healthy, healthy)
	fewerRenewals := seedRenewalGaps(t, testServer, "tie-fewer-renewals", "10.8.2.3", t0, breach, healthy)

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	ranked := resp.JSON200.AtRiskDevices
	is.Equal(len(ranked), 3)
	for _, d := range ranked {
		is.Equal(d.WorstRisk, httpapi.Breached) // the tie the sort key has to break
	}
	is.Equal(ranked[0].DeviceId, mostLate.Int64())      // 3 late renewals
	is.Equal(ranked[1].DeviceId, moreRenewals.Int64())  // 1 late, 3 renewals
	is.Equal(ranked[2].DeviceId, fewerRenewals.Int64()) // 1 late, 2 renewals

	is.Equal(ranked[0].LateRenewalCount, 3)
	is.Equal(ranked[1].LateRenewalCount, 1)
	is.Equal(ranked[1].RenewalCount, 3)
	is.Equal(ranked[2].LateRenewalCount, 1)
	is.Equal(ranked[2].RenewalCount, 2)
}

// TestHandler_GetAddressHistoryHistogram_RankingIsDeterministic guards the
// last tiebreaker: two devices identical on every sort key must still come back
// in a fixed order, or the list would reshuffle between polls of an unchanged
// window.
func TestHandler_GetAddressHistoryHistogram_RankingIsDeterministic(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	t0 := time.Now().UTC().Add(-3 * time.Hour)
	first := seedRenewalGaps(t, testServer, "twin-a", "10.8.3.1", t0, 200*time.Second, 200*time.Second)
	second := seedRenewalGaps(t, testServer, "twin-b", "10.8.3.2", t0, 200*time.Second, 200*time.Second)
	is.True(first < second) // seeded in id order, so ascending id is the expected order

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	for range 3 {
		resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
			From: &from, To: &to,
		})
		is.NoErr(err)
		is.Equal(resp.StatusCode(), http.StatusOK)
		is.Equal(len(resp.JSON200.AtRiskDevices), 2)
		is.Equal(resp.JSON200.AtRiskDevices[0].DeviceId, first.Int64())
		is.Equal(resp.JSON200.AtRiskDevices[1].DeviceId, second.Int64())
	}
}

// TestHandler_GetAddressHistoryHistogram_RankingCappedToWorst is the direct
// regression guard for the ranking being a top-N and not a full list: with more
// at-risk devices than the cap, the response must carry the worst ones rather
// than an arbitrary slice, since the reader treats the list as "what to fix
// first".
func TestHandler_GetAddressHistoryHistogram_RankingCappedToWorst(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	const (
		breach    = 200 * time.Second
		atRiskCap = 5 // addressHistoryAtRiskLimit
	)

	// Seven at-risk devices, each with a distinct number of late renewals, so
	// the expected top five is unambiguous.
	t0 := time.Now().UTC().Add(-3 * time.Hour)
	byLateCount := make([]ids.DeviceID, 0, 7)
	for lateCount := 1; lateCount <= 7; lateCount++ {
		gaps := make([]time.Duration, lateCount)
		for i := range gaps {
			gaps[i] = breach
		}
		id := seedRenewalGaps(t, testServer,
			fmt.Sprintf("capped-device-%d", lateCount),
			fmt.Sprintf("10.8.4.%d", lateCount),
			t0, gaps...,
		)
		byLateCount = append(byLateCount, id)
	}

	from := t0.Add(-time.Minute)
	to := time.Now().UTC().Add(time.Minute)
	resp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		From: &from, To: &to,
	})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)

	ranked := resp.JSON200.AtRiskDevices
	is.Equal(len(ranked), atRiskCap)

	// The devices with 7 … 3 late renewals, worst first; the 2 and 1 are cut.
	for i, d := range ranked {
		wantLate := 7 - i
		is.Equal(d.DeviceId, byLateCount[wantLate-1].Int64())
		is.Equal(d.LateRenewalCount, wantLate)
		is.Equal(d.WorstRisk, httpapi.Breached)
		is.True(d.P95GapSeconds > 0) // never the silent zero a missing gap row would leave
		is.True(d.TtlSeconds > 0)    // ranked devices always carry their configured TTL
		is.True(d.RenewalCount >= d.LateRenewalCount)
	}
}
