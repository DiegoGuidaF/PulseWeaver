//go:build test

package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/audit"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupTestRepo(t *testing.T) *audit.Repository {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)
	return audit.NewRepository(db.DB())
}

func TestRepository_BatchInsert_EmptyBatch(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)

	err := repo.BatchInsert(context.Background(), nil)
	is.NoErr(err)

	err = repo.BatchInsert(context.Background(), []policy.DecisionEvent{})
	is.NoErr(err)
}

func TestRepository_BatchInsert_AllowEvent(t *testing.T) {
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
			Headers:    map[string][]string{"User-Agent": {"Mozilla/5.0"}},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Allow events must not appear as deny reasons.
	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}

func TestRepository_BatchInsert_DenyEvent(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	reason := policy.DenyReasonIPNotRegistered
	events := []policy.DecisionEvent{
		{
			ClientIP:   "10.0.0.1",
			Outcome:    false,
			DenyReason: &reason,
			CreatedAt:  time.Now().UTC(),
			Headers:    map[string][]string{},
		},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 1)
	is.Equal(reasons[0], string(policy.DenyReasonIPNotRegistered))
}

func TestRepository_BatchInsert_MultipleEvents(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	r1 := policy.DenyReasonIPNotRegistered
	r2 := policy.DenyReasonNoDeviceMatch
	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &r2, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "3.3.3.3", Outcome: true, DenyReason: nil, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	// Both deny reasons stored; allow event excluded.
	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 2)
}

func TestRepository_ListDenyReasons_Empty(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)

	reasons, err := repo.ListDenyReasons(context.Background())
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}

func TestRepository_ListDenyReasons_ReturnsSortedDistinct(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	r1 := policy.DenyReasonIPNotRegistered
	r2 := policy.DenyReasonNoDeviceMatch

	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "2.2.2.2", Outcome: false, DenyReason: &r1, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}}, // duplicate
		{ClientIP: "3.3.3.3", Outcome: false, DenyReason: &r2, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 2)
	is.Equal(reasons[0], string(r1))
	is.Equal(reasons[1], string(r2))
}

func TestRepository_ListDenyReasons_ExcludesAllowEvents(t *testing.T) {
	is := is.New(t)
	repo := setupTestRepo(t)
	ctx := context.Background()

	events := []policy.DecisionEvent{
		{ClientIP: "1.1.1.1", Outcome: true, DenyReason: nil, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}

	err := repo.BatchInsert(ctx, events)
	is.NoErr(err)

	reasons, err := repo.ListDenyReasons(ctx)
	is.NoErr(err)
	is.Equal(len(reasons), 0)
}
