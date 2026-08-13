//go:build test

package accesslog_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// These are integration tests over the full seeded world (SeedFullWorld), exercised
// through the real HTTP stack. The access-log list query is too data-complex to test
// meaningfully against hand-built rows: filters, sort, keyset pagination, and the
// multi-contributor assembly all interact, so the realistic cross-domain world is the
// honest fixture. Operator→SQL translation and cursor mechanics are unit-tested in
// internal/filterx; per-column validation is unit-tested in access_log_query_test.go.
//
// The seeded access-log world (10 entries; see testutils.SeedFullWorld). Hosts
// are referenced by their Fixture identifier (api1=Backend1 … web2=Frontend2):
//   1 AliceAllow         10.1.0.1     allow  contrib[james-laptop]               host api1  country NULL
//   2 BobHostDeny        10.2.0.1     deny   contrib[noah-phone]                 host api2  country NULL
//   3 UnknownDeny        9.9.9.9      deny   no contrib                          host web1  country NULL
//   4 SharedIPAllow      10.1.0.1     allow  contrib[james-laptop,maria-desktop] host web2 country NULL  (ambiguous)
//   5 NetworkPolicyAllow 10.3.0.1     allow  policy corp-vpn                     host api1  country NULL
//   6 BypassAllow        192.168.1.50 allow  policy ops-network                  host web1  country NULL
//   7 GeoGermanyAPI      198.51.100.10 deny  no contrib  GET  30us  /api/users   host api1  country DE
//   8 GeoGermanyLogin    198.51.100.11 deny  no contrib  POST 220us /api/login   host api2  country DE
//   9 GeoUSA             198.51.100.20 deny  no contrib  GET  150us              host web1  country US
//  10 GeoSpain           198.51.100.30 deny  no contrib  DELETE 90us             host web2  country ES

func adminAccessLog(t *testing.T) (*app.App, *http.Cookie) {
	t.Helper()
	srv := testutils.SetupIntegrationServer(t)
	testutils.SeedFullWorld(t).Build(srv)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)
	return srv, cookie
}

func getAccessLog(t *testing.T, server http.Handler, cookie *http.Cookie, query string) (*httptest.ResponseRecorder, httpapi.AccessLogResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log"+query, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	var response httpapi.AccessLogResponse
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return rec, response
}

func getAccessLogHistogram(t *testing.T, server http.Handler, cookie *http.Cookie, query string) (*httptest.ResponseRecorder, httpapi.AccessLogHistogramResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log/histogram"+query, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	var response httpapi.AccessLogHistogramResponse
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	return rec, response
}

func TestHandler_GetAccessLog_EmptyRows(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	rec, response := getAccessLog(t, srv.HTTPServer, cookie, "")
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(response.Total, 0)
	is.Equal(len(response.Rows), 0)
	is.True(response.NextCursor == nil)
}

func TestHandler_GetAccessLog_Baseline(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	rec, all := getAccessLog(t, srv.HTTPServer, cookie, "")
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(all.Total, 10) // every seeded entry within the default 24h window

	_, allow := getAccessLog(t, srv.HTTPServer, cookie, "?outcome=true")
	is.Equal(allow.Total, 4) // AliceAllow, SharedIPAllow, NetworkPolicyAllow, BypassAllow

	_, deny := getAccessLog(t, srv.HTTPServer, cookie, "?outcome=false")
	is.Equal(deny.Total, 6) // BobHostDeny, UnknownDeny + 4 geolocated denies
}

// TestHandler_GetAccessLog_ValueFilters is the data-complex core: every operator
// shape against one realistic world, including the NULL-inclusion correctness rule.
func TestHandler_GetAccessLog_ValueFilters(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	// Multi-value IN: "traffic from Germany or the US".
	_, deOrUS := getAccessLog(t, srv.HTTPServer, cookie, "?country_code=DE&country_code=US")
	is.Equal(deOrUS.Total, 3) // GeoGermanyAPI, GeoGermanyLogin, GeoUSA

	// not_in on a nullable column MUST include the NULL-country rows. Six seeded
	// entries have no GeoIP; only GeoSpain is ES, so "everything except ES" is 9.
	_, notES := getAccessLog(t, srv.HTTPServer, cookie, "?country_code=ES&country_code_op=not_in")
	is.Equal(notES.Total, 9)

	// is_null: only the six entries lacking GeoIP.
	_, noGeo := getAccessLog(t, srv.HTTPServer, cookie, "?country_code_op=is_null")
	is.Equal(noGeo.Total, 6)

	// Substring on a host column: api1.internal / api2.internal contain "api".
	_, apiHosts := getAccessLog(t, srv.HTTPServer, cookie, "?target_host=api&target_host_op=contains")
	is.Equal(apiHosts.Total, 5) // AliceAllow, BobHostDeny, NetworkPolicyAllow, GeoGermanyAPI, GeoGermanyLogin

	// Substring on the URI column (only the two geo-germany rows carry a URI).
	_, apiURI := getAccessLog(t, srv.HTTPServer, cookie, "?target_uri=/api&target_uri_op=contains")
	is.Equal(apiURI.Total, 2)

	// is_null on the URI column: everything except those two.
	_, noURI := getAccessLog(t, srv.HTTPServer, cookie, "?target_uri_op=is_null")
	is.Equal(noURI.Total, 8)

	// http_method multi-value.
	_, gets := getAccessLog(t, srv.HTTPServer, cookie, "?http_method=GET")
	is.Equal(gets.Total, 2) // GeoGermanyAPI, GeoUSA

	// deny_reason enum.
	_, unregistered := getAccessLog(t, srv.HTTPServer, cookie, "?deny_reason=ip_not_registered")
	is.Equal(unregistered.Total, 5) // UnknownDeny + 4 geo
}

// TestHandler_GetAccessLog_Contributors covers the multi-contributor display: the
// relational device/user filters and the contributors[] assembly.
func TestHandler_GetAccessLog_Contributors(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	seed := testutils.SeedFullWorld(t).Build(srv)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	jamesLaptop := seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name)
	mariaDesktop := seed.Device(testutils.FixtureDeviceBypassAccess.Name)

	// device filter (relational EXISTS): james-laptop contributes to AliceAllow + SharedIPAllow.
	_, byDevice := getAccessLog(t, srv.HTTPServer, cookie, fmt.Sprintf("?device_id=%d", jamesLaptop))
	is.Equal(byDevice.Total, 2)

	// user filter: james owns james-laptop; same two entries.
	jamesUser := seed.User(testutils.FixtureUserWithAccess.Name)
	_, byUser := getAccessLog(t, srv.HTTPServer, cookie, fmt.Sprintf("?user_id=%d", jamesUser))
	is.Equal(byUser.Total, 2)

	// network_policy not_null: the two policy-matched allows.
	_, byPolicy := getAccessLog(t, srv.HTTPServer, cookie, "?network_policy_id_op=not_null")
	is.Equal(byPolicy.Total, 2)

	// The shared-IP entry surfaces BOTH contributors (ordered by device name),
	// not one collapsed row.
	_, sharedIP := getAccessLog(t, srv.HTTPServer, cookie, "?client_ip="+url.QueryEscape(testutils.FixtureAccessLogSharedIPAllow.ClientIP))
	var row *httpapi.AccessLogRow
	for i := range sharedIP.Rows {
		if sharedIP.Rows[i].ContributorCount == 2 {
			row = &sharedIP.Rows[i]
		}
	}
	is.True(row != nil)
	is.Equal(len(row.Contributors), 2)
	is.True(row.Contributors[0].DeviceId != nil)
	is.Equal(*row.Contributors[0].DeviceId, int64(jamesLaptop)) // "james-laptop" sorts before "maria-desktop"
	is.Equal(*row.Contributors[1].DeviceId, int64(mariaDesktop))

	// A 0-contributor entry (denied unknown IP) returns an empty slice, never nil.
	_, unknown := getAccessLog(t, srv.HTTPServer, cookie, "?client_ip="+url.QueryEscape(testutils.FixtureAccessLogUnknownDeny.ClientIP))
	is.Equal(unknown.Total, 1)
	is.Equal(unknown.Rows[0].ContributorCount, 0)
	is.Equal(len(unknown.Rows[0].Contributors), 0)
}

// TestHandler_GetAccessLog_SortAndPagination drives a non-default keyset sort and the
// default-sort cursor round-trip: ordered, stable, no duplicate or skipped rows.
func TestHandler_GetAccessLog_SortAndPagination(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	// Non-default sort: slowest requests first. Durations are 220,150,90,30 and six 0s.
	var durations []int64
	seen := map[int64]bool{}
	cursor := ""
	for range 10 { // safety bound > pages needed
		q := "?sort=duration_us&order=desc&limit=3"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		rec, page := getAccessLog(t, srv.HTTPServer, cookie, q)
		is.Equal(rec.Code, http.StatusOK)
		for _, r := range page.Rows {
			is.True(!seen[r.Id]) // no row repeats across pages
			seen[r.Id] = true
			is.True(r.DurationUs != nil)
			durations = append(durations, *r.DurationUs)
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}

	is.Equal(len(durations), 10) // every row paged through exactly once
	is.Equal(durations[0], int64(220))
	for i := 1; i < len(durations); i++ {
		is.True(durations[i] <= durations[i-1]) // monotonically non-increasing
	}

	// Default sort (created_at desc) cursor round-trip across all pages.
	seen = map[int64]bool{}
	cursor = ""
	pages := 0
	for range 10 {
		q := "?limit=4"
		if cursor != "" {
			q += "&cursor=" + url.QueryEscape(cursor)
		}
		_, page := getAccessLog(t, srv.HTTPServer, cookie, q)
		pages++
		for _, r := range page.Rows {
			is.True(!seen[r.Id])
			seen[r.Id] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	is.Equal(len(seen), 10) // 4 + 4 + 2
	is.Equal(pages, 3)
}

func TestHandler_GetAccessLog_BadRequests(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	// is_null is a valid operator in the enum but not allowed on client_ip → registry 400.
	rec, _ := getAccessLog(t, srv.HTTPServer, cookie, "?client_ip=1.2.3.4&client_ip_op=is_null")
	is.Equal(rec.Code, http.StatusBadRequest)

	// Malformed cursor token → 400.
	rec, _ = getAccessLog(t, srv.HTTPServer, cookie, "?cursor=not-a-valid-cursor")
	is.Equal(rec.Code, http.StatusBadRequest)

	// Sort value outside the enum is rejected by the OpenAPI request validator → 400.
	rec, _ = getAccessLog(t, srv.HTTPServer, cookie, "?sort=device_name")
	is.Equal(rec.Code, http.StatusBadRequest)
}

// TestHandler_GetAccessLog_SlimRows pins the list row to what the table renders.
// The six fields below are shown by the drawer alone, and the headers blob among
// them is unbounded, so shipping them on every row of every page is pure waste.
func TestHandler_GetAccessLog_SlimRows(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/access-log?limit=1", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(rec, req)
	is.Equal(rec.Code, http.StatusOK)

	var raw struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	is.NoErr(json.NewDecoder(rec.Body).Decode(&raw))
	is.Equal(len(raw.Rows), 1)
	for _, field := range []string{"headers", "xff_chain", "asn", "asn_org", "country_name", "continent_code"} {
		_, present := raw.Rows[0][field]
		is.True(!present)
	}
}

// TestHandler_GetAccessLogEntry checks the detail against the list row for the
// same id: identical on every shared field, with the drawer-only ones added on
// top. Two shapes over one table drift silently otherwise.
func TestHandler_GetAccessLogEntry(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv, cookie := adminAccessLog(t)
	client := testutils.NewAdminAPIClient(t, srv)

	// A geolocated entry, so the geo fields the detail adds are populated.
	_, page := getAccessLog(t, srv.HTTPServer, cookie, "?country_code=DE&sort=duration_us&order=desc&limit=1")
	is.Equal(len(page.Rows), 1)
	row := page.Rows[0]

	resp, err := client.GetAccessLogEntryWithResponse(ctx, row.Id)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusOK)
	detail := *resp.JSON200

	is.Equal(detail.Id, row.Id)
	is.Equal(detail.ClientIp, row.ClientIp)
	is.Equal(detail.Outcome, row.Outcome)
	is.Equal(detail.DenyReason, row.DenyReason)
	is.Equal(detail.CreatedAt, row.CreatedAt)
	is.Equal(detail.ContributorCount, row.ContributorCount)
	is.Equal(detail.Contributors, row.Contributors)
	is.Equal(detail.TargetHost, row.TargetHost)
	is.Equal(detail.TargetUri, row.TargetUri)
	is.Equal(detail.HttpMethod, row.HttpMethod)
	is.Equal(detail.CountryCode, row.CountryCode)
	is.Equal(detail.DurationUs, row.DurationUs)
	is.Equal(detail.NetworkPolicyId, row.NetworkPolicyId)
	is.Equal(detail.NetworkPolicyName, row.NetworkPolicyName)

	// The drawer-only fields the list no longer carries.
	is.True(detail.Headers != nil)
	is.True(detail.CountryName != nil)
	is.Equal(*detail.CountryName, testutils.FixtureGeoGermany.CountryName)
	is.True(detail.ContinentCode != nil)
	is.Equal(*detail.ContinentCode, testutils.FixtureGeoGermany.ContinentCode)
	is.True(detail.Asn != nil)
}

// TestHandler_GetAccessLogEntry_IsRowSuperset guards the AccessLogDetail schema's
// allOf over AccessLogRow: the contract promises detail carries every row field,
// but the two are mapped by separate hand-written functions. A field added to the
// row and forgotten in toAccessLogDetail would serialize as absent or zero, which
// the explicit field-by-field assertions above cannot notice — they only cover the
// fields someone remembered to list. Comparing the encoded objects needs no upkeep.
func TestHandler_GetAccessLogEntry_IsRowSuperset(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	getJSON := func(path string) map[string]json.RawMessage {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		srv.HTTPServer.ServeHTTP(rec, req)
		is.Equal(rec.Code, http.StatusOK)

		var decoded map[string]json.RawMessage
		if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		return decoded
	}

	// A geolocated entry with a contributor, so both the geo fields and the
	// nested array are non-empty rather than trivially equal.
	page := getJSON("/api/v1/access-log?country_code=DE&sort=duration_us&order=desc&limit=1")
	var rows []map[string]json.RawMessage
	is.NoErr(json.Unmarshal(page["rows"], &rows))
	is.Equal(len(rows), 1)
	row := rows[0]

	var id int64
	is.NoErr(json.Unmarshal(row["id"], &id))
	detail := getJSON(fmt.Sprintf("/api/v1/access-log/%d", id))

	for field, rowValue := range row {
		detailValue, ok := detail[field]
		if !ok {
			t.Errorf("detail is missing row field %q", field)
			continue
		}
		if !bytes.Equal(rowValue, detailValue) {
			t.Errorf("field %q: row has %s, detail has %s", field, rowValue, detailValue)
		}
	}
}

// TestHandler_GetAccessLogEntry_NotFound covers both ways an id stops resolving:
// it never existed, or retention pruned the row out from under a deep link.
func TestHandler_GetAccessLogEntry_NotFound(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv, cookie := adminAccessLog(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.GetAccessLogEntryWithResponse(ctx, 999_999)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)

	_, page := getAccessLog(t, srv.HTTPServer, cookie, "?limit=1")
	is.Equal(len(page.Rows), 1)
	prunedID := page.Rows[0].Id

	// Retention prunes by age; nothing survives a cutoff in the future.
	repo := accesslog.NewRepository(srv.Database.DB())
	_, err = repo.DeleteOlderThan(ctx, time.Now().UTC().Add(time.Hour))
	is.NoErr(err)

	resp, err = client.GetAccessLogEntryWithResponse(ctx, prunedID)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

// TestHandler_GetAccessLogHistogram_MatchesListTotal is the assertion that
// proves the chart and the table are built from one WHERE builder: for any
// filter set, summing the buckets gives the list's total. A second hand-written
// WHERE — or a bound that differs by one comparison — breaks this immediately.
func TestHandler_GetAccessLogHistogram_MatchesListTotal(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	seed := testutils.SeedFullWorld(t).Build(srv)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	for _, query := range []string{
		"",                                 // unfiltered
		"?outcome=false",                   // simple filter
		"?outcome=true",                    //
		"?country_code=DE&country_code=US", // multi-value IN over the geoip join
		"?country_code=ES&country_code_op=not_in", // NULL-inclusive not_in
		"?country_code_op=is_null",                // no GeoIP row at all
		"?target_host=api&target_host_op=contains",
		"?target_uri_op=is_null",
		"?http_method=GET",
		"?deny_reason=ip_not_registered",
		"?network_policy_id_op=not_null", // the network-policy join
		fmt.Sprintf("?device_id=%d", seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name)), // relational EXISTS
		fmt.Sprintf("?user_id=%d", seed.User(testutils.FixtureUserWithAccess.Name)),
		// Past the rollup threshold, where the histogram once answered from the
		// hourly aggregates and silently dropped the in-flight hour.
		"?from=" + url.QueryEscape(time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339)),
		"?outcome=true&from=" + url.QueryEscape(time.Now().UTC().Add(-30*24*time.Hour).Format(time.RFC3339)),
	} {
		_, list := getAccessLog(t, srv.HTTPServer, cookie, query)
		rec, hist := getAccessLogHistogram(t, srv.HTTPServer, cookie, query)
		is.Equal(rec.Code, http.StatusOK)

		var summed int64
		for _, b := range hist.Buckets {
			summed += b.AllowCount + b.DenyCount
		}
		if int(summed) != list.Total {
			t.Errorf("query %q: histogram sums to %d, list total is %d", query, summed, list.Total)
		}
	}
}

// TestHandler_GetAccessLogHistogram_ZeroFillsQuietBuckets: the window drives the
// series, not the matching rows. Filters make empty buckets the common case, and
// a series that omitted them would draw a quiet week as an unbroken wall.
func TestHandler_GetAccessLogHistogram_ZeroFillsQuietBuckets(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	// The seeded world writes every entry at once, so the default 24h window is
	// one busy hour surrounded by quiet ones.
	rec, hist := getAccessLogHistogram(t, srv.HTTPServer, cookie, "")
	is.Equal(rec.Code, http.StatusOK)
	is.Equal(len(hist.Buckets), 25) // hourly across 24h, both ends inclusive

	var quiet int
	for _, b := range hist.Buckets {
		if b.AllowCount+b.DenyCount == 0 {
			quiet++
		}
	}
	is.True(quiet >= 23) // the seeding may straddle one hour boundary, never more

	// Contiguous and ordered oldest-first, exactly one bucket width apart.
	for i := 1; i < len(hist.Buckets); i++ {
		gap := time.Time(hist.Buckets[i].Timestamp).Sub(time.Time(hist.Buckets[i-1].Timestamp))
		is.Equal(gap, time.Hour)
	}
}

// TestHandler_GetAccessLogHistogram_OutcomeFilterDegenerates documents the
// expected shape under the outcome filter: one band carries everything and the
// other is flat zero, because the filter has already excluded it.
func TestHandler_GetAccessLogHistogram_OutcomeFilterDegenerates(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	_, denied := getAccessLogHistogram(t, srv.HTTPServer, cookie, "?outcome=false")
	var allowed, denies int64
	for _, b := range denied.Buckets {
		allowed += b.AllowCount
		denies += b.DenyCount
	}
	is.Equal(allowed, int64(0))
	is.Equal(denies, int64(6)) // BobHostDeny, UnknownDeny + 4 geolocated denies
}

func TestHandler_GetAccessLogHistogram_BadRequest(t *testing.T) {
	is := is.New(t)
	srv, cookie := adminAccessLog(t)

	// is_null is in the operator enum but not allowed on client_ip → registry 400.
	rec, _ := getAccessLogHistogram(t, srv.HTTPServer, cookie, "?client_ip=1.2.3.4&client_ip_op=is_null")
	is.Equal(rec.Code, http.StatusBadRequest)
}
