//go:build test

package testutils

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/anomaly"
	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
)

// sample IPs referenced by the anomaly scenarios below. Kept next to the helper so
// the lookups against the seeded world fail fast if the sample addresses change.
const (
	samplePriyaIP  = "49.36.220.7" // Priya's offline iPhone → expired_access
	sampleLiamIP   = "86.43.220.11"
	sampleTravelIP = "91.64.12.9" // Liam's ThinkPad's second, German presence
)

// sampleGeoResolver resolves the sample world's public IPs from the hand-maintained
// sampleGeo table, standing in for the GeoIP database that is absent at seed time.
// It satisfies anomaly.GeoResolver (and rollup.GeoResolver).
type sampleGeoResolver struct{}

func (sampleGeoResolver) Resolve(ip string) geoip.Result {
	if g, ok := sampleGeo[ip]; ok {
		return g
	}
	return geoip.Result{}
}

// MaterializeSampleAnomalies fills the sample world's anomalies table by running the
// real detection pipeline over a few crafted scenarios, so a local deploy or a
// screenshot shows genuine findings across all three families rather than an empty
// list. It is the counterpart to the periodic scan that would produce these on a
// live instance; the seed generator starts no background jobs, so it is invoked
// explicitly after Build.
//
// Scenarios: expired_access (Priya's lapsed lease + a denied request), invalid_token
// (a bad-bearer caller), impossible_travel + new_country (Liam's ThinkPad suddenly
// also live from Germany), plus host_probing and geo_denied, which emerge from the
// existing scanner and no-access traffic once the scan runs.
func MaterializeSampleAnomalies(t *testing.T, srv *app.App, result *SeedResult) {
	t.Helper()
	ctx := t.Context()
	db := srv.Database.DB()
	now := time.Now()

	// expired_access — Priya's device lease lapsed, then her next request was denied.
	// Rebuild the address's events into a clean story (long-enabled, then disabled 40
	// minutes ago) so the lapse is the most-recent disable, and add a deny inside the
	// 60-minute grace window so it is attributed to the lapse rather than a scanner.
	priyaAddr := result.Address("Priya's iPhone", samplePriyaIP).Int64()
	execSample(t, db, `DELETE FROM address_events WHERE address_id = ?`, priyaAddr)
	execSample(t, db, `INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, 1, 'heartbeat', ?)`, priyaAddr, now.Add(-26*time.Hour).UTC())
	execSample(t, db, `INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, 0, 'lease', ?)`, priyaAddr, now.Add(-40*time.Minute).UTC())
	execSample(t, db, `INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, 'jellyfin.example.com', 0, 'ip_not_registered', ?, '{}')`,
		samplePriyaIP, now.Add(-25*time.Minute).UTC())

	// invalid_token — a caller hitting the verify endpoint with a broken bearer token.
	execSample(t, db, `INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES ('45.155.205.50', 'vault.example.com', 0, 'invalid_token', ?, '{}')`,
		now.Add(-20*time.Minute).UTC())

	// impossible_travel + new_country — Liam's ThinkPad, long established in Ireland,
	// is suddenly also live from Germany. The old IE profile satisfies the novelty
	// learning gate; the extra enabled German address makes both a novel country and
	// a concurrent two-country presence.
	anomalyRepo := anomaly.NewRepository(db)
	liamDev := result.Device("Liam's ThinkPad").Int64()
	if err := anomalyRepo.UpsertDeviceProfile(ctx, anomaly.ProfileObservation{
		DeviceID: liamDev, Dimension: "country", Fingerprint: "IE",
		SeenAt: now.Add(-30 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("seed baseline country profile: %v", err)
	}
	var travelAddr int64
	if err := db.QueryRowxContext(ctx, `INSERT INTO addresses (device_id, ip, source, is_enabled, created_at)
		VALUES (?, ?, 'heartbeat', 1, ?) RETURNING id`,
		liamDev, sampleTravelIP, now.Add(-2*time.Hour).UTC()).Scan(&travelAddr); err != nil {
		t.Fatalf("seed travel address: %v", err)
	}
	// The novelty country feed reads enable events, not the addresses table, so the
	// German presence needs a heartbeat enable event within the trailing window.
	execSample(t, db, `INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, 1, 'heartbeat', ?)`, travelAddr, now.Add(-2*time.Hour).UTC())

	// Aggregate the seeded traffic (geo_denied and the volume family read complete
	// hourly buckets), then run the real scan with the sample geo resolver.
	if err := rollup.NewRepository(db, nil).NewRollupJob(srv.Logger).Run(ctx); err != nil {
		t.Fatalf("rollup for anomaly showcase: %v", err)
	}
	job := anomaly.NewScanJob(anomalyRepo, anomaly.AllDetectors(anomalyRepo, sampleGeoResolver{}),
		anomaly.ScanOptions{
			Sensitivity:   "medium",
			LearningDays:  7,
			DetectRules:   true,
			DetectVolume:  true,
			DetectNovelty: true,
		}, srv.Logger)
	if err := job.Run(ctx); err != nil {
		t.Fatalf("anomaly scan for showcase: %v", err)
	}
}

func execSample(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(t.Context(), query, args...); err != nil {
		t.Fatalf("materialize anomalies: exec %q: %v", query, err)
	}
}
