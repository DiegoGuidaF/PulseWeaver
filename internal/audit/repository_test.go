//go:build test

package audit

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupTestRepo(t *testing.T) *Repository {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return NewRepository(db.DB())
}

func TestRepository_BatchInsert_EmptyBatch(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)

	err := repo.BatchInsert(context.Background(), nil)
	is.NoErr(err)

	err = repo.BatchInsert(context.Background(), []policy.DecisionEvent{})
	is.NoErr(err)
}

func TestRepository_BatchInsert_AllowEvents(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	targetHost := "example.com"
	targetURI := "/api"
	httpMethod := "GET"
	xffChain := "1.2.3.4"

	events := []policy.DecisionEvent{
		{
			ClientIP:   "1.2.3.4",
			Outcome:    true,
			DenyReason: nil,
			DeviceID:   nil,
			AddressID:  nil,
			CreatedAt:  time.Now().UTC(),
			TargetHost: &targetHost,
			TargetURI:  &targetURI,
			HTTPMethod: &httpMethod,
			XFFChain:   &xffChain,
			Headers: map[string][]string{
				"User-Agent": {"Mozilla/5.0"},
			},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Verify the row was persisted.
	type row struct {
		ClientIP   string  `db:"client_ip"`
		Outcome    int     `db:"outcome"`
		DenyReason *string `db:"deny_reason"`
		DeviceID   *int64  `db:"device_id"`
		AddressID  *int64  `db:"address_id"`
	}
	var got row
	err = repo.rootDB.GetContext(ctx, &got,
		`SELECT client_ip, outcome, deny_reason, device_id, address_id FROM request_audit_log WHERE client_ip = ?`,
		"1.2.3.4",
	)
	is.NoErr(err)
	is.Equal(got.ClientIP, "1.2.3.4")
	is.Equal(got.Outcome, 1)
	is.True(got.DenyReason == nil)
	is.True(got.DeviceID == nil)
	is.True(got.AddressID == nil)
}

func TestRepository_BatchInsert_DenyEvents(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	// Deny events where the IP was not registered have no device or address reference.
	reason := policy.DenyReasonIPNotRegistered

	events := []policy.DecisionEvent{
		{
			ClientIP:   "10.0.0.1",
			Outcome:    false,
			DenyReason: &reason,
			DeviceID:   nil,
			AddressID:  nil,
			CreatedAt:  time.Now().UTC(),
			Headers:    map[string][]string{},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	type row struct {
		ClientIP   string  `db:"client_ip"`
		Outcome    int     `db:"outcome"`
		DenyReason *string `db:"deny_reason"`
		DeviceID   *int64  `db:"device_id"`
		AddressID  *int64  `db:"address_id"`
	}
	var got row
	err = repo.rootDB.GetContext(ctx, &got,
		`SELECT client_ip, outcome, deny_reason, device_id, address_id FROM request_audit_log WHERE client_ip = ?`,
		"10.0.0.1",
	)
	is.NoErr(err)
	is.Equal(got.ClientIP, "10.0.0.1")
	is.Equal(got.Outcome, 0)
	is.True(got.DenyReason != nil)
	is.Equal(*got.DenyReason, string(policy.DenyReasonIPNotRegistered))
	is.True(got.DeviceID == nil)
	is.True(got.AddressID == nil)
}

func TestRepository_BatchInsert_MultiplePersisted(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "3.3.3.3", Outcome: true, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	var count int
	err = repo.rootDB.GetContext(ctx, &count, `SELECT COUNT(*) FROM request_audit_log`)
	is.NoErr(err)
	is.Equal(count, 3)
}
