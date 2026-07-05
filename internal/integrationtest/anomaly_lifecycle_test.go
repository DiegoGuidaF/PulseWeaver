//go:build test

package integrationtest_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/anomaly"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// enableAnomaly turns on the anomaly scan in a test config. Hand-built test
// configs leave ConfAnomaly zero-valued (Sensitivity "", Detect* false), which
// builds an inert scan job; this sets the working options app.go would get from
// config.Load()'s env defaults.
func enableAnomaly(c *config.Conf) {
	c.Anomaly = config.ConfAnomaly{
		Enabled: true,
		// 0 disables the self-gate so an explicit Run always scans; the app boots
		// with one gated pass (ExecuteScheduledRules) over the still-empty log.
		ScanInterval:  0,
		Sensitivity:   "medium",
		LearningDays:  7,
		DetectRules:   true,
		DetectVolume:  true,
		DetectNovelty: true,
	}
}

// TestAnomalyScan_SurfacesFindingsThroughAPI is the full-pipeline e2e: raw
// scenarios in the access log, driven through the app's own registered scan job
// (built by app.go via AllDetectors), then read and acknowledged over the real
// HTTP API, and finally pruned by retention.
//
// It deliberately uses raw-row detector families (rules + host_probing) that need
// no GeoIP resolver or hourly rollup — the volume/novelty/geo detectors are
// asserted against the sample world in internal/database's showcase guard test.
func TestAnomalyScan_SurfacesFindingsThroughAPI(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	const (
		aliceUserID   = 900
		aliceDeviceID = 900
		probeDeviceID = 901

		expiredIP  = "198.51.100.77"
		invalidIP  = "203.0.113.9"
		probingIP  = "198.51.100.90"
		invalidTgt = "app.example.com"
	)

	srv := testutils.SetupIntegrationServerWithConfig(t, enableAnomaly)
	db := srv.Database.DB()

	now := time.Now()

	// Shared owner for the attributed scenarios.
	exec(t, db, `INSERT INTO users (id, username, display_name, email, role)
		VALUES (?, 'alice', 'Alice Example', 'alice@example.com', 'user')`, aliceUserID)

	seedExpiredAccess(t, db, aliceDeviceID, aliceUserID, expiredIP, now)
	seedInvalidToken(t, db, invalidIP, invalidTgt, now)
	seedHostProbing(t, db, probeDeviceID, aliceUserID, probingIP, now)

	// Drive the app's own registered scan job — the wiring under test.
	is.NoErr(srv.AnomalyScanJob.Run(ctx))

	client := testutils.NewAdminAPIClient(t, srv)
	listed, err := client.ListAnomaliesWithResponse(ctx, &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	is.Equal(listed.StatusCode(), http.StatusOK)

	// expired_access — attributed to the device and its owner.
	expired := findKind(listed.JSON200.Anomalies, httpapi.AnomalyKindExpiredAccess)
	is.True(expired != nil)
	is.Equal(*expired.DeviceName, "alice-laptop")
	is.Equal(*expired.UserName, "Alice Example")
	is.Equal(expired.Status, httpapi.AnomalyStatus(httpapi.Open))

	// invalid_token — always critical, no device attribution.
	invalid := findKind(listed.JSON200.Anomalies, httpapi.AnomalyKindInvalidToken)
	is.True(invalid != nil)
	is.Equal(invalid.Severity, httpapi.AnomalySeverity(httpapi.Critical))

	// host_probing — the probe device denied across many distinct hosts.
	probing := findKind(listed.JSON200.Anomalies, httpapi.AnomalyKindHostProbing)
	is.True(probing != nil)
	is.Equal(*probing.DeviceName, "probe-laptop")

	// Acknowledge flips the status and is idempotent.
	ack, err := client.AcknowledgeAnomalyWithResponse(ctx, expired.Id)
	is.NoErr(err)
	is.Equal(ack.StatusCode(), http.StatusNoContent)

	reListed, err := client.ListAnomaliesWithResponse(ctx, &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	acked := findID(reListed.JSON200.Anomalies, expired.Id)
	is.True(acked != nil)
	is.Equal(acked.Status, httpapi.AnomalyStatus(httpapi.Acknowledged))

	again, err := client.AcknowledgeAnomalyWithResponse(ctx, expired.Id)
	is.NoErr(err)
	is.Equal(again.StatusCode(), http.StatusNoContent)

	// Retention prunes an anomaly last seen before the cutoff and leaves the
	// fresh scan findings, which the API then reflects.
	stale := insertAnomalyRow(t, db, "deny_spike", now.Add(-400*24*time.Hour))
	pruned, err := anomaly.NewRepository(db).DeleteAnomaliesOlderThan(ctx, now.Add(-100*24*time.Hour))
	is.NoErr(err)
	is.Equal(pruned, int64(1))

	afterPrune, err := client.ListAnomaliesWithResponse(ctx, &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	is.True(findID(afterPrune.JSON200.Anomalies, stale) == nil)      // stale pruned
	is.True(findID(afterPrune.JSON200.Anomalies, probing.Id) != nil) // fresh kept
}

// ── scenario seeding (raw Givens — no Seeder fixture for access-log rows) ──────

func seedExpiredAccess(t *testing.T, db *database.DB, deviceID, userID int64, ip string, now time.Time) {
	t.Helper()
	disableAt := now.Add(-2 * time.Hour)
	exec(t, db, `INSERT INTO devices (id, name, owner_id) VALUES (?, 'alice-laptop', ?)`, deviceID, userID)
	exec(t, db, `INSERT INTO addresses (id, device_id, ip, source, is_enabled, created_at)
		VALUES (?, ?, ?, 'lease', 0, ?)`, deviceID, deviceID, ip, disableAt.Add(-24*time.Hour).UTC())
	exec(t, db, `INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (?, 0, 'lease', ?)`, deviceID, disableAt.UTC())
	exec(t, db, `INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, 'app.example.com', 0, 'ip_not_registered', ?, '{}')`, ip, disableAt.Add(10*time.Minute).UTC())
}

func seedInvalidToken(t *testing.T, db *database.DB, ip, targetHost string, now time.Time) {
	t.Helper()
	exec(t, db, `INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, 0, 'invalid_token', ?, '{}')`, ip, targetHost, now.Add(-30*time.Minute).UTC())
}

// seedHostProbing writes host_not_allowed denies across enough distinct hosts to
// clear the medium preset (5), each linked to the probe device via a contributor
// row so the detector can attribute it.
func seedHostProbing(t *testing.T, db *database.DB, deviceID, userID int64, ip string, now time.Time) {
	t.Helper()
	const addrID = 9010
	exec(t, db, `INSERT INTO devices (id, name, owner_id) VALUES (?, 'probe-laptop', ?)`, deviceID, userID)
	exec(t, db, `INSERT INTO addresses (id, device_id, ip, source, is_enabled, created_at)
		VALUES (?, ?, ?, 'manual', 1, ?)`, addrID, deviceID, ip, now.Add(-48*time.Hour).UTC())

	hosts := []string{"a.example.com", "b.example.com", "c.example.com", "d.example.com", "e.example.com", "f.example.com"}
	for i, host := range hosts {
		var logID int64
		err := db.QueryRowxContext(ctx(t), `INSERT INTO access_log
			(client_ip, target_host, outcome, deny_reason, contributor_count, created_at, headers_json)
			VALUES (?, ?, 0, 'host_not_allowed', 1, ?, '{}') RETURNING id`,
			ip, host, now.Add(-time.Duration(i)*time.Minute).UTC()).Scan(&logID)
		if err != nil {
			t.Fatalf("insert probing access_log: %v", err)
		}
		exec(t, db, `INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id)
			VALUES (?, ?, ?, ?)`, logID, deviceID, addrID, userID)
	}
}

func insertAnomalyRow(t *testing.T, db *database.DB, kind string, lastSeen time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowxContext(ctx(t), `INSERT INTO anomalies
		(kind, severity, status, fingerprint, first_seen_at, last_seen_at, evidence_json)
		VALUES (?, 'warning', 'open', ?, ?, ?, '{}') RETURNING id`,
		kind, kind+":"+lastSeen.Format(time.RFC3339Nano), lastSeen.UTC(), lastSeen.UTC()).Scan(&id)
	if err != nil {
		t.Fatalf("insert anomaly row: %v", err)
	}
	return id
}

func exec(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx(t), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func ctx(t *testing.T) context.Context { return t.Context() }

func findKind(rows []httpapi.Anomaly, kind httpapi.AnomalyKind) *httpapi.Anomaly {
	for i := range rows {
		if rows[i].Kind == kind {
			return &rows[i]
		}
	}
	return nil
}

func findID(rows []httpapi.Anomaly, id int64) *httpapi.Anomaly {
	for i := range rows {
		if rows[i].Id == id {
			return &rows[i]
		}
	}
	return nil
}
