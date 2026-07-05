//go:build test

package queries_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/anomaly"
	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// insertAnomaly writes one row straight into the anomalies table and returns its
// id, so the list/acknowledge API can be exercised without driving a scan. Name
// resolution is covered end-to-end by TestAnomalies_ScanToAPI, so seeded rows
// leave the denormalized name columns empty.
func insertAnomaly(t *testing.T, db *database.DB, kind, status string, lastSeen time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowxContext(context.Background(),
		`INSERT INTO anomalies
		   (kind, severity, status, fingerprint, first_seen_at, last_seen_at, evidence_json)
		 VALUES (?, 'warning', ?, ?, ?, ?, '{}') RETURNING id`,
		kind, status, kind+":"+lastSeen.Format(time.RFC3339Nano), lastSeen.UTC(), lastSeen.UTC(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert anomaly: %v", err)
	}
	return id
}

func anomalyByID(rows []httpapi.Anomaly, id int64) (httpapi.Anomaly, bool) {
	for _, a := range rows {
		if a.Id == id {
			return a, true
		}
	}
	return httpapi.Anomaly{}, false
}

func TestListAnomalies_NewestFirstAndFilters(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	db := srv.Database.DB()
	now := time.Now()

	oldOpen := insertAnomaly(t, db, "expired_access", "open", now.Add(-3*time.Hour))
	newestOpen := insertAnomaly(t, db, "deny_spike", "open", now.Add(-1*time.Hour))
	ackMid := insertAnomaly(t, db, "invalid_token", "acknowledged", now.Add(-2*time.Hour))

	client := testutils.NewAdminAPIClient(t, srv)

	all, err := client.ListAnomaliesWithResponse(context.Background(), &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	is.Equal(all.StatusCode(), http.StatusOK)
	is.Equal(len(all.JSON200.Anomalies), 3)
	// Newest-first by last_seen: newestOpen, ackMid, oldOpen.
	is.Equal(all.JSON200.Anomalies[0].Id, newestOpen)
	is.Equal(all.JSON200.Anomalies[1].Id, ackMid)
	is.Equal(all.JSON200.Anomalies[2].Id, oldOpen)

	openStatus := httpapi.Open
	openOnly, err := client.ListAnomaliesWithResponse(context.Background(), &httpapi.ListAnomaliesParams{Status: &openStatus})
	is.NoErr(err)
	is.Equal(len(openOnly.JSON200.Anomalies), 2)
	_, hasAck := anomalyByID(openOnly.JSON200.Anomalies, ackMid)
	is.True(!hasAck)

	kinds := []httpapi.AnomalyKind{httpapi.AnomalyKindExpiredAccess, httpapi.AnomalyKindInvalidToken}
	byKind, err := client.ListAnomaliesWithResponse(context.Background(), &httpapi.ListAnomaliesParams{Kind: &kinds})
	is.NoErr(err)
	is.Equal(len(byKind.JSON200.Anomalies), 2)
	_, hasSpike := anomalyByID(byKind.JSON200.Anomalies, newestOpen)
	is.True(!hasSpike)

	limit := 1
	limited, err := client.ListAnomaliesWithResponse(context.Background(), &httpapi.ListAnomaliesParams{Limit: &limit})
	is.NoErr(err)
	is.Equal(len(limited.JSON200.Anomalies), 1)
	is.Equal(limited.JSON200.Anomalies[0].Id, newestOpen)
}

func TestListAnomalies_Unauthenticated_401(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.ListAnomaliesWithResponse(context.Background(), &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

func TestAcknowledgeAnomaly_FlipsThenIdempotent(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	db := srv.Database.DB()
	id := insertAnomaly(t, db, "deny_spike", "open", time.Now())

	client := testutils.NewAdminAPIClient(t, srv)

	ack, err := client.AcknowledgeAnomalyWithResponse(context.Background(), id)
	is.NoErr(err)
	is.Equal(ack.StatusCode(), http.StatusNoContent)

	listed, err := client.ListAnomaliesWithResponse(context.Background(), &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	row, ok := anomalyByID(listed.JSON200.Anomalies, id)
	is.True(ok)
	is.Equal(row.Status, httpapi.AnomalyStatus(httpapi.Acknowledged))

	// A second acknowledge is a no-op success.
	again, err := client.AcknowledgeAnomalyWithResponse(context.Background(), id)
	is.NoErr(err)
	is.Equal(again.StatusCode(), http.StatusNoContent)
}

func TestAcknowledgeAnomaly_UnknownID_404(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.AcknowledgeAnomalyWithResponse(context.Background(), 987654)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusNotFound)
}

func TestAcknowledgeAnomaly_Unauthenticated_401(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	db := srv.Database.DB()
	id := insertAnomaly(t, db, "deny_spike", "open", time.Now())

	client := testutils.NewAPIClient(t, srv)
	resp, err := client.AcknowledgeAnomalyWithResponse(context.Background(), id)
	is.NoErr(err)
	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

// TestAnomalies_ScanToAPI is the end-to-end story: an expired lease plus a
// post-expiry deny becomes an expired_access anomaly, and the list API returns
// it with the device and user names resolved.
func TestAnomalies_ScanToAPI(t *testing.T) {
	is := is.New(t)
	srv := testutils.SetupIntegrationServer(t)
	db := srv.Database.DB()
	ctx := context.Background()

	disableAt := time.Now().Add(-2 * time.Hour)
	mustExec(t, db, `INSERT INTO users (id, username, display_name, email, role)
		VALUES (900, 'alice', 'Alice Example', 'alice@example.com', 'user')`)
	mustExec(t, db, `INSERT INTO devices (id, name, owner_id) VALUES (900, 'alice-laptop', 900)`)
	mustExec(t, db, `INSERT INTO addresses (id, device_id, ip, source, is_enabled, created_at)
		VALUES (900, 900, '198.51.100.77', 'manual', 0, ?)`, disableAt.Add(-24*time.Hour).UTC())
	mustExec(t, db, `INSERT INTO address_events (address_id, is_enabled, source, created_at)
		VALUES (900, 0, 'lease', ?)`, disableAt.UTC())
	mustExec(t, db, `INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES ('198.51.100.77', 'app.example.com', 0, 'ip_not_registered', ?, '{}')`, disableAt.Add(10*time.Minute).UTC())

	repo := anomaly.NewRepository(db)
	job := anomaly.NewScanJob(repo, anomaly.AllDetectors(repo, nil),
		anomaly.ScanOptions{Interval: 0, Sensitivity: "medium", DetectRules: true}, srv.Logger)
	is.NoErr(job.Run(ctx))

	client := testutils.NewAdminAPIClient(t, srv)
	listed, err := client.ListAnomaliesWithResponse(ctx, &httpapi.ListAnomaliesParams{})
	is.NoErr(err)
	is.Equal(listed.StatusCode(), http.StatusOK)

	var found *httpapi.Anomaly
	for i := range listed.JSON200.Anomalies {
		if listed.JSON200.Anomalies[i].Kind == httpapi.AnomalyKindExpiredAccess {
			found = &listed.JSON200.Anomalies[i]
		}
	}
	is.True(found != nil) // scan produced an expired_access anomaly
	is.Equal(*found.DeviceName, "alice-laptop")
	is.Equal(*found.UserName, "Alice Example")
	is.Equal(found.Status, httpapi.AnomalyStatus(httpapi.Open))
}

func mustExec(t *testing.T, db *database.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}
