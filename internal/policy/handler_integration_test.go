//go:build test

package policy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// verifyIPRequest builds a forward-auth request that exercises the real
// ClientIPFromRealIPMiddleware: RemoteAddr is set to the test server's trusted
// proxy (127.0.0.1) so the middleware trusts and extracts X-Real-IP as the
// client IP. targetHost is sent as X-Forwarded-Host when non-empty.
func verifyIPRequest(clientIP, targetHost string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/policy-engine/verify-ip", nil)
	r.RemoteAddr = "127.0.0.1:1"
	r.Header.Set("Authorization", "Bearer test-policy-secret")
	r.Header.Set("X-Real-IP", clientIP)
	if targetHost != "" {
		r.Header.Set("X-Forwarded-Host", targetHost)
	}
	return r
}

// TestHandlerIntegration_ForwardAuth_XRealIPMiddleware verifies the full
// middleware wiring: X-Real-IP from a trusted proxy feeds the policy decision.
func TestHandlerIntegration_ForwardAuth_XRealIPMiddleware(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	testutils.NewSeeder(t).
		WithUser(testutils.UserFixture{Name: "alice"}).
		SetUserAccess("alice", true).
		WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
		WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: "10.0.0.1"}).
		WithPolicyInitialize().
		Build(srv)

	w := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w, verifyIPRequest("10.0.0.1", ""))
	is.Equal(w.Code, http.StatusOK)
}

// TestHandlerIntegration_ForwardAuth_FullWorldDecisionTour exercises every
// decision path present in the full seeded world via real HTTP calls.
func TestHandlerIntegration_ForwardAuth_FullWorldDecisionTour(t *testing.T) {
	srv := testutils.SetupIntegrationServer(t)
	testutils.SeedFullWorld(t).Build(srv)

	cases := []struct {
		ip     string
		host   string
		expect int
		label  string
	}{
		// device path — alice+charlie share 10.1.0.1; alice has backend+frontend,
		// charlie bypasses; host is in alice's allowed set.
		{"10.1.0.1", testutils.FixtureHostBackend1.FQDN, http.StatusOK, "device allow shared IP"},
		// device path — bob has no group access; any host is denied.
		{"10.2.0.1", testutils.FixtureHostBackend1.FQDN, http.StatusForbidden, "device deny no groups"},
		// CIDR path — corp-vpn (10.0.0.0/8) covers 10.3.0.1; backend host allowed.
		{"10.3.0.1", testutils.FixtureHostBackend1.FQDN, http.StatusOK, "CIDR allow corp-vpn"},
		// CIDR path — same corp-vpn policy but host not in any assigned group.
		{"10.3.0.1", "unknown.internal", http.StatusForbidden, "CIDR deny unknown host"},
		// bypass CIDR — ops-network (192.168.0.0/16) has bypass_host_check=true.
		{"192.168.1.50", "any.host.example", http.StatusOK, "bypass CIDR allow any host"},
		// no device, no CIDR — ip_not_registered.
		{"9.9.9.9", testutils.FixtureHostBackend1.FQDN, http.StatusForbidden, "ip not registered"},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			is := is.New(t)
			w := httptest.NewRecorder()
			srv.HTTPServer.ServeHTTP(w, verifyIPRequest(tc.ip, tc.host))
			is.Equal(w.Code, tc.expect)
		})
	}
}

// TestHandlerIntegration_ForwardAuth_MostSpecificCIDRWins confirms that when an
// IP falls under two overlapping CIDRs, the narrower prefix takes precedence.
func TestHandlerIntegration_ForwardAuth_MostSpecificCIDRWins(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	// /8 allows a.com; /16 allows b.com. IP 10.1.2.3 is in both — /16 must win.
	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "group-a"}).
		WithGroup(testutils.GroupFixture{Name: "group-b"}).
		WithHost(testutils.HostFixture{FQDN: "a.com", Groups: []string{"group-a"}}).
		WithHost(testutils.HostFixture{FQDN: "b.com", Groups: []string{"group-b"}}).
		WithPolicy(testutils.PolicyFixture{Name: "broad", CIDR: "10.0.0.0/8"}).
		WithPolicy(testutils.PolicyFixture{Name: "narrow", CIDR: "10.1.0.0/16"}).
		AssignGroupsToPolicy("broad", "group-a").
		AssignGroupsToPolicy("narrow", "group-b").
		WithPolicyInitialize().
		Build(srv)

	w1 := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w1, verifyIPRequest("10.1.2.3", "b.com"))
	is.Equal(w1.Code, http.StatusOK) // /16 wins: b.com allowed

	w2 := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w2, verifyIPRequest("10.1.2.3", "a.com"))
	is.Equal(w2.Code, http.StatusForbidden) // /16 wins: a.com not in its host list
}

// TestHandlerIntegration_ForwardAuth_DeviceBeatsNetworkPolicy confirms that when
// an IP is registered as a device address and also falls within a CIDR policy,
// the device path — with its own host restrictions — takes priority.
func TestHandlerIntegration_ForwardAuth_DeviceBeatsNetworkPolicy(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)

	// alice's device at 10.0.0.5 is restricted to x.com via x-group.
	// CIDR 10.0.0.0/8 would allow y.com, but the device path must win.
	testutils.NewSeeder(t).
		WithGroup(testutils.GroupFixture{Name: "x-group"}).
		WithGroup(testutils.GroupFixture{Name: "y-group"}).
		WithHost(testutils.HostFixture{FQDN: "x.com", Groups: []string{"x-group"}}).
		WithHost(testutils.HostFixture{FQDN: "y.com", Groups: []string{"y-group"}}).
		WithUser(testutils.UserFixture{Name: "alice"}).
		SetUserAccess("alice", false, "x-group").
		WithDevice(testutils.DeviceFixture{Name: "alice-laptop", OwnerUser: "alice"}).
		WithAddress(testutils.AddressFixture{Device: "alice-laptop", IP: "10.0.0.5"}).
		WithPolicy(testutils.PolicyFixture{Name: "broad", CIDR: "10.0.0.0/8"}).
		AssignGroupsToPolicy("broad", "y-group").
		WithPolicyInitialize().
		Build(srv)

	w1 := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w1, verifyIPRequest("10.0.0.5", "x.com"))
	is.Equal(w1.Code, http.StatusOK) // device path: x.com in alice's allowed set

	w2 := httptest.NewRecorder()
	srv.HTTPServer.ServeHTTP(w2, verifyIPRequest("10.0.0.5", "y.com"))
	is.Equal(w2.Code, http.StatusForbidden) // device path wins: y.com not in alice's set
}
