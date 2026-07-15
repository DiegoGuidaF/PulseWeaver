//go:build test

package anomaly_test

import (
	"encoding/json"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/anomaly"
	"github.com/matryer/is"
)

// roundTrip mimics how evidence actually reaches Summarize: persisted as
// JSON and decoded back into map[string]any, so numbers become float64 and
// string slices become []any.
func roundTrip(t *testing.T, evidence map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(evidence)
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal evidence: %v", err)
	}
	return out
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name     string
		kind     anomaly.Kind
		evidence map[string]any
		want     string
	}{
		{
			name:     "expired_access",
			kind:     anomaly.KindExpiredAccess,
			evidence: map[string]any{"deny_count": 3, "lease_source": "rule"},
			want:     "Denied 3 times after its address lease expired — the user may be silently locked out.",
		},
		{
			name:     "expired_access singular",
			kind:     anomaly.KindExpiredAccess,
			evidence: map[string]any{"deny_count": 1},
			want:     "Denied 1 time after its address lease expired — the user may be silently locked out.",
		},
		{
			name:     "expired_access empty",
			kind:     anomaly.KindExpiredAccess,
			evidence: map[string]any{},
			want:     "Denied after its address lease expired — the user may be silently locked out.",
		},
		{
			name:     "invalid_token with hosts",
			kind:     anomaly.KindInvalidToken,
			evidence: map[string]any{"deny_count": 5, "target_hosts": []string{"a.example.com", "b.example.com", "c.example.com"}},
			want:     "5 requests with an invalid bearer token targeting 3 hosts — the proxy token may be broken, or something else is calling the verify endpoint.",
		},
		{
			name:     "invalid_token without hosts",
			kind:     anomaly.KindInvalidToken,
			evidence: map[string]any{"deny_count": 1},
			want:     "1 request with an invalid bearer token — the proxy token may be broken, or something else is calling the verify endpoint.",
		},
		{
			name:     "invalid_token empty",
			kind:     anomaly.KindInvalidToken,
			evidence: map[string]any{},
			want:     "Requests with an invalid bearer token — the proxy token may be broken, or something else is calling the verify endpoint.",
		},
		{
			name:     "deny_spike deny outcome",
			kind:     anomaly.KindDenySpike,
			evidence: map[string]any{"outcome": "deny", "series": "global", "observed": 120, "baseline": 15.0, "threshold": 50},
			want:     "120 denials in an hour on global vs a typical 15 (threshold 50).",
		},
		{
			name:     "deny_spike allow outcome with fractional baseline",
			kind:     anomaly.KindDenySpike,
			evidence: map[string]any{"outcome": "allow", "series": "app.example.com", "observed": 42, "baseline": 15.55, "threshold": 30},
			want:     "42 requests in an hour on app.example.com vs a typical 15.6 (threshold 30).",
		},
		{
			name:     "deny_spike empty",
			kind:     anomaly.KindDenySpike,
			evidence: map[string]any{},
			want:     "Unusual traffic volume in an hour.",
		},
		{
			name: "entity_drift device",
			kind: anomaly.KindEntityDrift,
			evidence: map[string]any{
				"entity_kind": "device", "entity_name": "alice-laptop", "outcome": "deny",
				"observed": 80, "baseline": 12.0, "threshold": 40,
			},
			want: "Device 'alice-laptop' saw 80 deny-requests in an hour vs a typical 12.",
		},
		{
			name: "entity_drift policy",
			kind: anomaly.KindEntityDrift,
			evidence: map[string]any{
				"entity_kind": "policy", "entity_name": "corp-net", "outcome": "allow",
				"observed": 500, "baseline": 100.5,
			},
			want: "Policy 'corp-net' saw 500 allow-requests in an hour vs a typical 100.5.",
		},
		{
			name:     "entity_drift empty",
			kind:     anomaly.KindEntityDrift,
			evidence: map[string]any{},
			want:     "This entity saw an unusual number of request-requests in an hour.",
		},
		{
			name:     "geo_denied with asn",
			kind:     anomaly.KindGeoDenied,
			evidence: map[string]any{"country_code": "DE", "country_name": "Germany", "deny_count": 12, "asn_org": "Hetzner Online GmbH", "hosts": []string{"app.example.com"}},
			want:     "12 denials from Germany (Hetzner Online GmbH), outside the expected countries.",
		},
		{
			name:     "geo_denied without asn",
			kind:     anomaly.KindGeoDenied,
			evidence: map[string]any{"country_code": "FR", "country_name": "France", "deny_count": 1},
			want:     "1 denial from France, outside the expected countries.",
		},
		{
			name:     "geo_denied empty",
			kind:     anomaly.KindGeoDenied,
			evidence: map[string]any{},
			want:     "Denials from an unrecognized country, outside the expected countries.",
		},
		{
			name:     "host_probing",
			kind:     anomaly.KindHostProbing,
			evidence: map[string]any{"distinct_hosts": 8, "deny_count": 22, "threshold": 5, "hosts": []string{"a.example.com"}},
			want:     "Denied on 8 distinct hosts (22 denials) — fanning across services looks like probing.",
		},
		{
			name:     "host_probing empty",
			kind:     anomaly.KindHostProbing,
			evidence: map[string]any{},
			want:     "Denied on multiple distinct hosts — fanning across services looks like probing.",
		},
		{
			name:     "address_churn",
			kind:     anomaly.KindAddressChurn,
			evidence: map[string]any{"new_addresses": 6, "threshold": 3},
			want:     "6 new addresses registered within 24 h (threshold 3) — possible key sharing or spoofing.",
		},
		{
			name:     "address_churn singular",
			kind:     anomaly.KindAddressChurn,
			evidence: map[string]any{"new_addresses": 1, "threshold": 3},
			want:     "1 new address registered within 24 h (threshold 3) — possible key sharing or spoofing.",
		},
		{
			name:     "address_churn empty",
			kind:     anomaly.KindAddressChurn,
			evidence: map[string]any{},
			want:     "New addresses registered within 24 h — possible key sharing or spoofing.",
		},
		{
			name:     "new_user_agent",
			kind:     anomaly.KindNewUserAgent,
			evidence: map[string]any{"user_agent": "curl/8.5.0", "ua_fingerprint": "abc123"},
			want:     `First time this device presents "curl/8.5.0".`,
		},
		{
			name:     "new_user_agent empty",
			kind:     anomaly.KindNewUserAgent,
			evidence: map[string]any{},
			want:     "First time this device presents a new user agent.",
		},
		{
			name:     "new_country",
			kind:     anomaly.KindNewCountry,
			evidence: map[string]any{"country_code": "JP"},
			want:     "First activity from JP for this device.",
		},
		{
			name:     "new_country empty",
			kind:     anomaly.KindNewCountry,
			evidence: map[string]any{},
			want:     "First activity from a new country for this device.",
		},
		{
			name:     "impossible_travel concurrent_presence two countries",
			kind:     anomaly.KindImpossibleTravel,
			evidence: map[string]any{"signal": "concurrent_presence", "countries": []string{"DE", "US"}, "ips": []string{"1.2.3.4", "5.6.7.8"}},
			want:     "Active in DE and US at the same time.",
		},
		{
			name:     "impossible_travel concurrent_presence three countries",
			kind:     anomaly.KindImpossibleTravel,
			evidence: map[string]any{"signal": "concurrent_presence", "countries": []string{"DE", "FR", "US"}},
			want:     "Active in DE, FR, and US at the same time.",
		},
		{
			name:     "impossible_travel country_hop",
			kind:     anomaly.KindImpossibleTravel,
			evidence: map[string]any{"signal": "country_hop", "from_country": "DE", "to_country": "JP", "same_continent": false},
			want:     "Moved DE → JP faster than travel allows.",
		},
		{
			name:     "impossible_travel empty",
			kind:     anomaly.KindImpossibleTravel,
			evidence: map[string]any{},
			want:     "Impossible travel detected for this device.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			got := anomaly.Summarize(tc.kind, roundTrip(t, tc.evidence))
			is.Equal(got, tc.want)
		})
	}
}
