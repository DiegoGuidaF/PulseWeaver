//go:build test

package rollup_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

func setupTestRepo(t *testing.T) (*rollup.Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	repo := rollup.NewRepository(db.DB(), nil)
	return repo, db.DB()
}

// seedAccessLogRow inserts a single row into access_log for testing.
func seedAccessLogRow(t *testing.T, db *database.DB, clientIP string, targetHost string, outcome bool, denyReason string, createdAt time.Time) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, ?, ?, ?, '{}')
	`, clientIP, targetHost, outcomeInt, denyReason, createdAt.UTC())
	if err != nil {
		t.Fatalf("seed access row: %v", err)
	}
}

// seedAggregateRow inserts a pre-computed aggregate row directly into hourly_traffic_aggregates.
// Used to set up the long-range (> 24h) query path without needing RunRollup.
func seedAggregateRow(t *testing.T, db *database.DB, bucketAt time.Time, clientIP string, targetHost string, outcome bool, count int64) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	bucketStr := bucketAt.UTC().Truncate(time.Hour).Format("2006-01-02 15:04:05+00:00")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO hourly_traffic_aggregates
			(bucket_at, client_ip, target_host, outcome, deny_reason, request_count, sum_duration_us, max_duration_us)
		VALUES (?, ?, ?, ?, '', ?, 0, 0)
	`, bucketStr, clientIP, targetHost, outcomeInt, count)
	if err != nil {
		t.Fatalf("seed aggregate row: %v", err)
	}
}

// seedAccessLogRowWithGeo inserts an access_log row plus its access_log_geoip child.
func seedAccessLogRowWithGeo(t *testing.T, db *database.DB, clientIP string, targetHost string, outcome bool, createdAt time.Time, countryCode, countryName, continentCode string) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	var logID int64
	err := db.QueryRowxContext(t.Context(), `
		INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, ?, '', ?, '{}') RETURNING id
	`, clientIP, targetHost, outcomeInt, createdAt.UTC()).Scan(&logID)
	if err != nil {
		t.Fatalf("seed access row with geo: %v", err)
	}
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO access_log_geoip (access_log_id, country_code, country_name, continent_code)
		VALUES (?, ?, ?, ?)
	`, logID, countryCode, countryName, continentCode)
	if err != nil {
		t.Fatalf("seed geoip row: %v", err)
	}
}

// seedNetworkPolicy inserts a network_policies row with an explicit id so
// contributor and aggregate FKs (policy_id REFERENCES network_policies(id))
// resolve. cidr must be unique across rows.
func seedNetworkPolicy(t *testing.T, db *database.DB, id int64, name, cidr string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO network_policies (id, name, cidr) VALUES (?, ?, ?)
	`, id, name, cidr)
	if err != nil {
		t.Fatalf("seed network policy: %v", err)
	}
}

// seedPolicyAccessLogRow inserts an access_log row plus its network-policy
// contributor, the raw-path source for the policy split. A nil policyID models
// a deleted policy (ON DELETE SET NULL); policyName is always retained.
func seedPolicyAccessLogRow(t *testing.T, db *database.DB, clientIP string, policyID *int64, policyName string, outcome bool, createdAt time.Time) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	var logID int64
	err := db.QueryRowxContext(t.Context(), `
		INSERT INTO access_log (client_ip, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, '', ?, '{}') RETURNING id
	`, clientIP, outcomeInt, createdAt.UTC()).Scan(&logID)
	if err != nil {
		t.Fatalf("seed policy access row: %v", err)
	}
	_, err = db.ExecContext(t.Context(), `
		INSERT INTO access_log_network_policy_contributors (access_log_id, policy_id, policy_name)
		VALUES (?, ?, ?)
	`, logID, policyID, policyName)
	if err != nil {
		t.Fatalf("seed policy contributor: %v", err)
	}
}

// seedAttributionAggregateRow inserts a pre-computed row into
// hourly_attribution_aggregates, setting up the aggregate (> 24h) path without
// needing RunAttributionRollup. A nil entityID models a hard-deleted entity.
func seedAttributionAggregateRow(t *testing.T, db *database.DB, bucketAt time.Time, kind rollup.AttributionKind, entityID *int64, entityName string, outcome bool, count int64) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	bucketStr := bucketAt.UTC().Truncate(time.Hour).Format("2006-01-02 15:04:05+00:00")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO hourly_attribution_aggregates (bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`, bucketStr, string(kind), entityID, entityName, outcomeInt, count)
	if err != nil {
		t.Fatalf("seed attribution aggregate row: %v", err)
	}
}

// seedUser inserts a users row with an explicit id so contributor FKs resolve.
func seedUser(t *testing.T, db *database.DB, id int64, username, displayName string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO users (id, username, display_name, email, role)
		VALUES (?, ?, ?, ?, 'user')
	`, id, username, displayName, username+"@example.com")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

// seedDevice inserts a devices row owned by ownerID with an explicit id.
func seedDevice(t *testing.T, db *database.DB, id, ownerID int64, name string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO devices (id, name, owner_id) VALUES (?, ?, ?)
	`, id, name, ownerID)
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}
}

// seedAddress inserts an addresses row on deviceID with an explicit id.
func seedAddress(t *testing.T, db *database.DB, id, deviceID int64, ip string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO addresses (id, device_id, ip, source, is_enabled) VALUES (?, ?, ?, 'manual', 1)
	`, id, deviceID, ip)
	if err != nil {
		t.Fatalf("seed address: %v", err)
	}
}

// contributor identifies one device/address/user link of an access_log row.
type contributor struct {
	deviceID, addressID, userID int64
}

// seedContributorAccessLogRow inserts one access_log row plus an
// access_log_contributors row per contributor — the raw-path source for the
// user and device splits. Multiple contributors model a shared-IP request that
// matched several devices.
func seedContributorAccessLogRow(t *testing.T, db *database.DB, clientIP string, outcome bool, createdAt time.Time, contributors ...contributor) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	var logID int64
	err := db.QueryRowxContext(t.Context(), `
		INSERT INTO access_log (client_ip, outcome, deny_reason, created_at, headers_json)
		VALUES (?, ?, '', ?, '{}') RETURNING id
	`, clientIP, outcomeInt, createdAt.UTC()).Scan(&logID)
	if err != nil {
		t.Fatalf("seed contributor access row: %v", err)
	}
	for _, c := range contributors {
		_, err = db.ExecContext(t.Context(), `
			INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id)
			VALUES (?, ?, ?, ?)
		`, logID, c.deviceID, c.addressID, c.userID)
		if err != nil {
			t.Fatalf("seed access log contributor: %v", err)
		}
	}
}

// seedAggregateDenyRow inserts a denied aggregate row carrying an explicit
// deny_reason, used to exercise the long-range (> 24h) deny-by-reason split.
func seedAggregateDenyRow(t *testing.T, db *database.DB, bucketAt time.Time, clientIP, denyReason string, count int64) {
	t.Helper()
	bucketStr := bucketAt.UTC().Truncate(time.Hour).Format("2006-01-02 15:04:05+00:00")
	_, err := db.ExecContext(t.Context(), `
		INSERT INTO hourly_traffic_aggregates
			(bucket_at, client_ip, target_host, outcome, deny_reason, request_count, sum_duration_us, max_duration_us)
		VALUES (?, ?, 'app.example.com', 0, ?, ?, 0, 0)
	`, bucketStr, clientIP, denyReason, count)
	if err != nil {
		t.Fatalf("seed aggregate deny row: %v", err)
	}
}

func attributionByName(counts []rollup.AttributionCount) map[string]rollup.AttributionCount {
	m := make(map[string]rollup.AttributionCount, len(counts))
	for _, c := range counts {
		m[c.EntityName] = c
	}
	return m
}
