//go:build test

package queries

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/matryer/is"
)

// TestReducePosture verifies the histogram fold over user statuses and the
// pass-through of the audit's top-level fields.
func TestReducePosture(t *testing.T) {
	is := is.New(t)

	refreshedAt := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	audit := httpapi.PolicyUserMapAudit{
		RefreshedAt:    httpapi.UTCTime(refreshedAt),
		SharedIpCount:  3,
		TotalHostCount: 7,
		Users: []httpapi.PolicyUserEntry{
			{Status: httpapi.Bypass},
			{Status: httpapi.LiveWithAccess},
			{Status: httpapi.LiveWithAccess},
			{Status: httpapi.LiveNoHostAccess},
			{Status: httpapi.NoLiveIps},
			{Status: httpapi.NoLiveIps},
			{Status: httpapi.NoLiveIps},
			{Status: httpapi.NoAccess},
		},
		NetworkPolicies: []httpapi.PolicyNetworkPolicyEntry{
			{BypassHostCheck: true},
			{BypassHostCheck: false},
		},
		TotalNetworkPolicyCount: 2,
	}

	got := reducePosture(audit, 5)

	is.Equal(got.RefreshedAt, httpapi.UTCTime(refreshedAt))
	is.Equal(got.Users.Bypass, 1)
	is.Equal(got.Users.LiveWithAccess, 2)
	is.Equal(got.Users.LiveNoHostAccess, 1)
	is.Equal(got.Users.NoLiveIps, 3)
	is.Equal(got.Users.NoAccess, 1)
	is.Equal(got.NetworkPolicies.Enabled, 2)
	is.Equal(got.NetworkPolicies.BypassHostCheck, 1)
	is.Equal(got.SharedIpCount, 3)
	is.Equal(got.KnownHostCount, 7)
	is.Equal(got.PendingSuggestionCount, 5)
}
