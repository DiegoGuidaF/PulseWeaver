//go:build test

package integrationtest_test

import (
	"net/http"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// TestVerifyIP_NativeV6_NetworkPolicyCIDR is a cross-domain integration test that
// verifies the network-policy CIDR fallback path works for native IPv6 (PW-67).
//
// It is also the first integration coverage of the CIDR fallback itself: a source
// IP that matches no device address but falls inside an enabled network policy's
// prefix is authorized via that policy.
//
//  1. A bypass-host-check policy covers 2001:db8::/48.
//  2. A v6 source inside the prefix is allowed for any host (bypass_host_check).
//  3. A v6 source outside the prefix is denied.
//
// Background services start AFTER seeding to avoid SQLite lock contention.
func TestVerifyIP_NativeV6_NetworkPolicyCIDR(t *testing.T) {
	is := is.New(t)

	const (
		v6InCIDR  = "2001:db8::5"
		v6OutCIDR = "2001:dead::1"
		anyHost   = "api.internal"
	)

	srv, _ := testutils.SetupRunningIntegrationServer(t,
		testutils.NewSeeder(t).
			WithPolicy(testutils.PolicyFixture{Name: "v6-net", CIDR: "2001:db8::/48"}).
			WithPolicyBypassHostCheck("v6-net").
			WithPolicyInitialize(),
	)

	// A source inside the v6 CIDR is allowed; bypass_host_check makes the host irrelevant.
	w := verifyIP(t, srv, v6InCIDR, anyHost)
	is.Equal(w.Code, http.StatusOK)

	// A source outside the CIDR is denied.
	w = verifyIP(t, srv, v6OutCIDR, anyHost)
	is.Equal(w.Code, http.StatusForbidden)
}
