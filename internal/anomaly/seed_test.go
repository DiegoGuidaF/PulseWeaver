//go:build test

package anomaly

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
)

func newRepo(t *testing.T) (*Repository, *database.DB) {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return NewRepository(db.DB()), db.DB()
}

// scopeAll covers every raw row (unbounded id range) and pins Now for the
// windowed detectors; sensitivity selects the preset under test.
func scopeAll(sensitivity string) Scope {
	return Scope{FromAccessLogID: 0, ToAccessLogID: 1 << 62, Now: time.Now(), Sensitivity: sensitivity}
}

// seedUserID is the single owner every seeded device in this suite belongs to.
const seedUserID = 1

func seedUser(t *testing.T, db *database.DB) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO users (id, username, display_name, email, role) VALUES (?, 'owner', 'owner', 'owner@example.com', 'user')`,
		seedUserID)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedDevice(t *testing.T, db *database.DB, id int64, name string) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO devices (id, name, owner_id) VALUES (?, ?, ?)`, id, name, seedUserID)
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}
}

func seedAddress(t *testing.T, db *database.DB, id, deviceID int64, ip string, enabled bool, createdAt time.Time) {
	t.Helper()
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO addresses (id, device_id, ip, source, is_enabled, created_at) VALUES (?, ?, ?, 'manual', ?, ?)`,
		id, deviceID, ip, enabledInt, createdAt.UTC())
	if err != nil {
		t.Fatalf("seed address: %v", err)
	}
}

func seedDisableEvent(t *testing.T, db *database.DB, addressID int64, at time.Time) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO address_events (address_id, is_enabled, source, created_at) VALUES (?, 0, 'lease', ?)`,
		addressID, at.UTC())
	if err != nil {
		t.Fatalf("seed disable event: %v", err)
	}
}

// seedDeny inserts an access_log deny row and returns its id.
func seedDeny(t *testing.T, db *database.DB, clientIP, targetHost, denyReason string, at time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowxContext(t.Context(),
		`INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		 VALUES (?, ?, 0, ?, ?, '{}') RETURNING id`,
		clientIP, targetHost, denyReason, at.UTC()).Scan(&id)
	if err != nil {
		t.Fatalf("seed deny: %v", err)
	}
	return id
}

// seedAllow inserts an access_log allow row and returns its id.
func seedAllow(t *testing.T, db *database.DB, clientIP, targetHost string, at time.Time) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowxContext(t.Context(),
		`INSERT INTO access_log (client_ip, target_host, outcome, deny_reason, created_at, headers_json)
		 VALUES (?, ?, 1, '', ?, '{}') RETURNING id`,
		clientIP, targetHost, at.UTC()).Scan(&id)
	if err != nil {
		t.Fatalf("seed allow: %v", err)
	}
	return id
}

// seedTrafficAgg inserts one hourly_traffic_aggregates row.
func seedTrafficAgg(t *testing.T, db *database.DB, bucketAt time.Time, targetHost string, outcome bool, count int64, countryCode string) {
	t.Helper()
	outcomeInt := 0
	if outcome {
		outcomeInt = 1
	}
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO hourly_traffic_aggregates
		   (bucket_at, client_ip, target_host, outcome, deny_reason, request_count, country_code, country_name, continent_code)
		 VALUES (?, '203.0.113.9', ?, ?, '', ?, ?, '', '')`,
		bucketAt.UTC().Truncate(time.Hour), targetHost, outcomeInt, count, countryCode)
	if err != nil {
		t.Fatalf("seed traffic aggregate: %v", err)
	}
}

// seedAttrAgg inserts one denied hourly_attribution_aggregates row (nil entityID
// models a hard-deleted entity).
func seedAttrAgg(t *testing.T, db *database.DB, bucketAt time.Time, kind string, entityID *int64, name string, count int64) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO hourly_attribution_aggregates (bucket_at, entity_kind, entity_id, entity_name, outcome, request_count)
		 VALUES (?, ?, ?, ?, 0, ?)`,
		bucketAt.UTC().Truncate(time.Hour), kind, entityID, name, count)
	if err != nil {
		t.Fatalf("seed attribution aggregate: %v", err)
	}
}

func seedContributor(t *testing.T, db *database.DB, accessLogID, deviceID, addressID, userID int64) {
	t.Helper()
	_, err := db.ExecContext(t.Context(),
		`INSERT INTO access_log_contributors (access_log_id, device_id, address_id, user_id) VALUES (?, ?, ?, ?)`,
		accessLogID, deviceID, addressID, userID)
	if err != nil {
		t.Fatalf("seed contributor: %v", err)
	}
}
