//go:build test

package anomaly

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/matryer/is"
)

// seedProbeDenies seeds `hosts` distinct host_not_allowed denies for device 1,
// each attributed via a contributor row, within the trailing window.
func seedProbeDenies(t *testing.T, db *database.DB, hosts int) {
	t.Helper()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "203.0.113.5", true, time.Now().Add(-48*time.Hour))
	for i := range hosts {
		id := seedDeny(t, db, "203.0.113.5", fmt.Sprintf("h%d.example.com", i), "host_not_allowed", time.Now().Add(-1*time.Hour))
		seedContributor(t, db, id, 1, 1)
	}
}

func TestHostProbingDetector_ManyDistinctHosts_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedProbeDenies(t, db, 5) // medium threshold is 5

	findings, err := hostProbingDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindHostProbing)
	is.Equal(findings[0].Severity, SeverityWarning)
	is.Equal(findings[0].Evidence["distinct_hosts"], int64(5))
}

// TestHostProbingDetector_ManyDeniesOneHost_NoFinding: volume on a single host is
// not probing — the signal is distinct-host cardinality.
func TestHostProbingDetector_ManyDeniesOneHost_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "203.0.113.5", true, time.Now().Add(-48*time.Hour))
	for range 5 {
		id := seedDeny(t, db, "203.0.113.5", "one.example.com", "host_not_allowed", time.Now().Add(-1*time.Hour))
		seedContributor(t, db, id, 1, 1)
	}

	findings, err := hostProbingDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

// TestHostProbingDetector_NoContributorAttribution_NoFinding: denies matched by a
// network policy (no access_log_contributors row) are out of scope for this kind.
func TestHostProbingDetector_NoContributorAttribution_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	for i := range 6 {
		seedDeny(t, db, "203.0.113.5", fmt.Sprintf("h%d.example.com", i), "host_not_allowed", time.Now().Add(-1*time.Hour))
	}

	findings, err := hostProbingDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

// TestHostProbingDetector_ThresholdScaling drives the same 5 distinct hosts
// through each sensitivity preset (low=8, medium=5, high=3).
func TestHostProbingDetector_ThresholdScaling(t *testing.T) {
	cases := []struct {
		sensitivity string
		wantFlagged bool
	}{
		{"low", false},
		{"medium", true},
		{"high", true},
	}
	for _, tc := range cases {
		t.Run(tc.sensitivity, func(t *testing.T) {
			is := is.New(t)
			repo, db := newRepo(t)
			seedProbeDenies(t, db, 5)

			findings, err := hostProbingDetector{reader: repo}.Detect(context.Background(), scopeAll(tc.sensitivity))

			is.NoErr(err)
			is.Equal(len(findings) == 1, tc.wantFlagged)
		})
	}
}

func TestAddressChurnDetector_ManyNewAddresses_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	for i := range 10 { // medium threshold is 10
		seedAddress(t, db, int64(100+i), 1, fmt.Sprintf("10.0.0.%d", i), true, time.Now().Add(-1*time.Hour))
	}

	findings, err := addressChurnDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindAddressChurn)
	is.Equal(findings[0].Evidence["new_addresses"], int64(10))
}

// TestAddressChurnDetector_RefreshesOneAddress_NoFinding: heartbeat refreshes of
// a single address create no new address rows, so churn stays at one.
func TestAddressChurnDetector_RefreshesOneAddress_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "10.0.0.1", true, time.Now().Add(-1*time.Hour))
	for range 10 {
		seedDisableEvent(t, db, 1, time.Now().Add(-30*time.Minute)) // event churn, not address churn
	}

	findings, err := addressChurnDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}
