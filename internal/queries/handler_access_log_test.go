//go:build test

package queries_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
// internal/queries/filterx; per-column validation is unit-tested in access_log_query_test.go.
//
// The seeded access-log world (10 entries; see testutils.SeedFullWorld):
//   1 AliceAllow         10.1.0.1     allow  contrib[alice-laptop]               host api1  country NULL
//   2 BobHostDeny        10.2.0.1     deny   contrib[bob-phone]                  host api2  country NULL
//   3 UnknownDeny        9.9.9.9      deny   no contrib                          host web1  country NULL
//   4 SharedIPAllow      10.1.0.1     allow  contrib[alice-laptop,charlie-desktop] host web2 country NULL  (ambiguous)
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

	// Continent multi/enum: EU = DE+DE+ES.
	_, eu := getAccessLog(t, srv.HTTPServer, cookie, "?continent_code=EU")
	is.Equal(eu.Total, 3)

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
// relational device/user filters, the contributors[] assembly, and the ambiguous flag.
func TestHandler_GetAccessLog_Contributors(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	seed := testutils.SeedFullWorld(t).Build(srv)
	cookie := testutils.LoginCookie(t, srv.HTTPServer, "admin", testutils.TestAdminPassword)

	aliceLaptop := seed.Device(testutils.FixtureDeviceWithOwnerAccess.Name)
	charlieDesktop := seed.Device(testutils.FixtureDeviceBypassAccess.Name)

	// device filter (relational EXISTS): alice-laptop contributes to AliceAllow + SharedIPAllow.
	_, byDevice := getAccessLog(t, srv.HTTPServer, cookie, fmt.Sprintf("?device_id=%d", aliceLaptop))
	is.Equal(byDevice.Total, 2)

	// user filter: alice owns alice-laptop; same two entries.
	aliceUser := seed.User(testutils.FixtureUserWithAccess.Name)
	_, byUser := getAccessLog(t, srv.HTTPServer, cookie, fmt.Sprintf("?user_id=%d", aliceUser))
	is.Equal(byUser.Total, 2)

	// network_policy not_null: the two policy-matched allows.
	_, byPolicy := getAccessLog(t, srv.HTTPServer, cookie, "?network_policy_id_op=not_null")
	is.Equal(byPolicy.Total, 2)

	// ambiguous=true → only the shared-IP entry (contributor_count > 1), and it
	// surfaces BOTH contributors (ordered by device name), not one collapsed row.
	_, ambiguous := getAccessLog(t, srv.HTTPServer, cookie, "?ambiguous=true")
	is.Equal(ambiguous.Total, 1)
	is.Equal(len(ambiguous.Rows), 1)
	row := ambiguous.Rows[0]
	is.Equal(row.ContributorCount, 2)
	is.Equal(len(row.Contributors), 2)
	is.True(row.Contributors[0].DeviceId != nil)
	is.Equal(*row.Contributors[0].DeviceId, int64(aliceLaptop)) // "alice-laptop" sorts before
	is.Equal(*row.Contributors[1].DeviceId, int64(charlieDesktop))

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
