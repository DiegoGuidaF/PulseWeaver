//go:build test

package device_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

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

	devA, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "filter-a", nil)
	is.NoErr(err)
	devB, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "filter-b", nil)
	is.NoErr(err)

	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, devA.ID, "10.1.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, devA.ID, "10.1.0.2", device.EventSourceManual)
	is.NoErr(err)
	_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, devB.ID, "10.2.0.1", device.EventSourceHeartbeat)
	is.NoErr(err)

	deviceIDFilter := []httpapi.ID{devA.ID.Int64()}

	eventsResp, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(eventsResp.StatusCode(), http.StatusOK)
	is.Equal(eventsResp.JSON200.Total, 2) // devA's two creations only, devB excluded

	histResp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(histResp.StatusCode(), http.StatusOK)

	sum := 0
	for _, b := range histResp.JSON200.Buckets {
		sum += b.EventCount
	}
	is.Equal(sum, eventsResp.JSON200.Total)
}

// addressHistoryFilterCase is one filter set applied to both address-history
// endpoints. The fields mirror the generated params structs, which are distinct
// types per endpoint even though they carry the same filter columns.
type addressHistoryFilterCase struct {
	name      string
	source    []httpapi.AddressEventSource
	eventKind []httpapi.AddressEventKind
	ttlRisk   []httpapi.TTLRisk
	ip        []string
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

	cases := []addressHistoryFilterCase{
		{name: "event_kind refresh only", eventKind: []httpapi.AddressEventKind{httpapi.AddressEventKindRefresh}},
		{name: "event_kind state changes", eventKind: []httpapi.AddressEventKind{
			httpapi.AddressEventKindCreated, httpapi.AddressEventKindEnabled, httpapi.AddressEventKindDisabled,
		}},
		{name: "ttl_risk breached", ttlRisk: []httpapi.TTLRisk{httpapi.Breached}},
		{name: "ttl_risk unknown", ttlRisk: []httpapi.TTLRisk{httpapi.Unknown}},
		{name: "ttl_risk ok", ttlRisk: []httpapi.TTLRisk{httpapi.Ok}},
		{name: "source expiry", source: []httpapi.AddressEventSource{httpapi.Expiry}},
		{name: "ip", ip: []string{"10.5.0.1"}},
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

			sum := 0
			for _, b := range histResp.JSON200.Buckets {
				sum += b.EventCount
			}
			is.Equal(sum, eventsResp.JSON200.Total)
			// Guard against passing vacuously: every case above must match rows,
			// otherwise 0 == 0 would "prove" parity for a filter that never ran.
			is.True(eventsResp.JSON200.Total > 0)
			is.Equal(len(eventsResp.JSON200.Events), eventsResp.JSON200.Total)
		})
	}
}

// sliceParam adapts a filter case's value slice to the pointer-to-slice the
// generated params structs use, mapping "no values" to "param not supplied".
func sliceParam[T any](values []T) *[]T {
	if len(values) == 0 {
		return nil
	}
	return &values
}

// TestHandler_GetAddressHistoryHistogram_IgnoresEventsCursor confirms the
// histogram is never scoped by the events endpoint's pagination: it has no
// cursor parameter at all, and its total must reflect every matching event,
// not just the page the caller happens to be viewing.
func TestHandler_GetAddressHistoryHistogram_IgnoresEventsCursor(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	testServer := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, testServer)

	dev, err := testServer.DeviceService.CreateDevice(ctx, testutils.AdminPrincipal(t, testServer), "cursor-device", nil)
	is.NoErr(err)

	for i := range 5 {
		_, _, err = testServer.DeviceService.RegisterAddressActivity(ctx, dev.ID, fmt.Sprintf("10.0.0.%d", i+1), device.EventSourceHeartbeat)
		is.NoErr(err)
	}

	deviceIDFilter := []httpapi.ID{dev.ID.Int64()}
	limit := 2
	page1, err := client.GetAddressHistoryWithResponse(ctx, &httpapi.GetAddressHistoryParams{
		DeviceId: &deviceIDFilter,
		Limit:    &limit,
	})
	is.NoErr(err)
	is.Equal(page1.StatusCode(), http.StatusOK)
	is.Equal(len(page1.JSON200.Events), 2)
	is.True(page1.JSON200.NextCursor != nil) // more pages exist

	histResp, err := client.GetAddressHistoryHistogramWithResponse(ctx, &httpapi.GetAddressHistoryHistogramParams{
		DeviceId: &deviceIDFilter,
	})
	is.NoErr(err)
	is.Equal(histResp.StatusCode(), http.StatusOK)

	sum := 0
	for _, b := range histResp.JSON200.Buckets {
		sum += b.EventCount
	}
	is.Equal(sum, 5) // all 5 events, unaffected by the events endpoint being mid-page
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
// regression guard for task 08b item 1: a device with both an approaching and
// a critical event in the same bucket must be counted once, under critical
// only — never in both bands.
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
		{t0.Add(90 * time.Second)},  // gap 90s / ttl 100s = 0.9 → approaching
		{t0.Add(185 * time.Second)}, // gap 95s / ttl 100s = 0.95 → critical
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

	is.Equal(len(buckets), 1) // create + approaching + critical all land in the same 5-minute bucket
	is.Equal(buckets[0].EventCount, 3)
	is.Equal(buckets[0].CriticalDeviceCount, 1)    // worst risk wins
	is.Equal(buckets[0].ApproachingDeviceCount, 0) // not double-counted in the lower band
	is.Equal(buckets[0].BreachedDeviceCount, 0)

	is.Equal(len(resp.JSON200.AtRiskDevices), 1)
	is.Equal(resp.JSON200.AtRiskDevices[0].DeviceId, dev.ID.Int64())
	is.Equal(resp.JSON200.AtRiskDevices[0].WorstRisk, httpapi.Critical)
}

// TestHandler_GetAddressHistoryHistogram_ExcludesOkAndUnknown is the direct
// regression guard for task 08b item 2: ok and unknown never contribute to
// the risk device counts or the at-risk ranking, however many events exist.
func TestHandler_GetAddressHistoryHistogram_ExcludesOkAndUnknown(t *testing.T) {
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

	is.True(len(resp.JSON200.Buckets) > 0) // event_count buckets still present …
	riskTotal := 0
	eventTotal := 0
	for _, b := range resp.JSON200.Buckets {
		eventTotal += b.EventCount
		riskTotal += b.ApproachingDeviceCount + b.CriticalDeviceCount + b.BreachedDeviceCount
	}
	is.Equal(eventTotal, 2) // create + ok refresh
	is.Equal(riskTotal, 0)  // … but contribute nothing to the risk bands
	is.Equal(len(resp.JSON200.AtRiskDevices), 0)
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
// direct regression guard for task 08b item 5's last case: the at-risk
// ranking's event_count reflects a device's total matching events across the
// whole window, not one page of them — the histogram endpoint carries no
// cursor parameter at all, so it cannot be scoped by the events endpoint's
// pagination position.
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
	is.Equal(resp.JSON200.AtRiskDevices[0].DeviceId, dev.ID.Int64())
	is.Equal(resp.JSON200.AtRiskDevices[0].WorstRisk, httpapi.Breached)
	is.Equal(resp.JSON200.AtRiskDevices[0].EventCount, 6) // create + 5 heartbeats, unaffected by the events page size
}
