//go:build test

package rollup_test

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/matryer/is"
)

// --- GetAttributionSplit: policy kind ---

// TestGetAttributionSplit_Policy_RawPath: a ≤24h window aggregates per-policy
// allow/deny straight from access_log + contributors.
func TestGetAttributionSplit_Policy_RawPath(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	policyA := int64(1)
	policyB := int64(2)
	seedNetworkPolicy(t, db, policyA, "policy-a", "10.0.0.0/8")
	seedNetworkPolicy(t, db, policyB, "policy-b", "10.1.0.0/16")
	// policy-a: 2 allow + 1 deny; policy-b: 1 allow.
	seedPolicyAccessLogRow(t, db, "10.0.0.1", &policyA, "policy-a", true, hour.Add(1*time.Minute))
	seedPolicyAccessLogRow(t, db, "10.0.0.1", &policyA, "policy-a", true, hour.Add(2*time.Minute))
	seedPolicyAccessLogRow(t, db, "10.0.0.2", &policyA, "policy-a", false, hour.Add(3*time.Minute))
	seedPolicyAccessLogRow(t, db, "10.0.0.3", &policyB, "policy-b", true, hour.Add(4*time.Minute))

	entities, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindPolicy, from, to)
	is.NoErr(err)

	byName := attributionByName(entities)
	is.Equal(len(entities), 2)
	is.Equal(byName["policy-a"].AllowCount, int64(2))
	is.Equal(byName["policy-a"].DenyCount, int64(1))
	is.Equal(*byName["policy-a"].EntityID, policyA)
	is.Equal(byName["policy-b"].AllowCount, int64(1))
	is.Equal(byName["policy-b"].DenyCount, int64(0))
	// Highest total first.
	is.Equal(entities[0].EntityName, "policy-a")
}

// TestGetAttributionSplit_Policy_AggregatePath: a >24h window reads from
// hourly_attribution_aggregates and ignores raw access_log rows.
func TestGetAttributionSplit_Policy_AggregatePath(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	// Raw sentinel that must NOT surface on the aggregate path.
	rawID := int64(9)
	seedNetworkPolicy(t, db, rawID, "raw-only", "10.9.0.0/16")
	seedPolicyAccessLogRow(t, db, "10.9.9.9", &rawID, "raw-only", true, base.Add(5*time.Minute))

	policyA := int64(1)
	seedAttributionAggregateRow(t, db, base, rollup.AttributionKindPolicy, &policyA, "policy-a", true, 5)
	seedAttributionAggregateRow(t, db, base.Add(time.Hour), rollup.AttributionKindPolicy, &policyA, "policy-a", false, 2)

	from := base
	to := base.Add(48 * time.Hour) // > 24h → aggregate path

	entities, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindPolicy, from, to)
	is.NoErr(err)

	is.Equal(len(entities), 1) // raw-only is invisible to the aggregate path
	is.Equal(entities[0].EntityName, "policy-a")
	is.Equal(entities[0].AllowCount, int64(5))
	is.Equal(entities[0].DenyCount, int64(2))
}

// TestGetAttributionSplit_KindIsolation: rows of a different entity_kind in the
// same window never leak into a kind's aggregate split.
func TestGetAttributionSplit_KindIsolation(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	policyID := int64(1)
	userID := int64(1)
	seedAttributionAggregateRow(t, db, base, rollup.AttributionKindPolicy, &policyID, "policy-a", true, 5)
	seedAttributionAggregateRow(t, db, base, rollup.AttributionKindUser, &userID, "alice", true, 9)

	entities, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindUser, base, base.Add(48*time.Hour))
	is.NoErr(err)
	is.Equal(len(entities), 1)
	is.Equal(entities[0].EntityName, "alice")
	is.Equal(entities[0].AllowCount, int64(9))
}

// TestGetAttributionSplit_DeletedEntity: traffic whose entity was hard-deleted
// (entity_id nulled) is still reported under its retained entity_name.
func TestGetAttributionSplit_DeletedEntity(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	base := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	seedAttributionAggregateRow(t, db, base, rollup.AttributionKindDevice, nil, "retired-laptop", true, 4)

	entities, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindDevice, base, base.Add(48*time.Hour))
	is.NoErr(err)

	is.Equal(len(entities), 1)
	is.Equal(entities[0].EntityName, "retired-laptop")
	is.Equal(entities[0].AllowCount, int64(4))
	is.Equal(entities[0].EntityID, (*int64)(nil)) // nil for a deleted entity
}

func TestGetAttributionSplit_UnknownKind_Errors(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	_, err := repo.GetAttributionSplit(ctx, rollup.AttributionKind("nonsense"),
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC))
	is.True(err != nil)
}

func TestGetAttributionSplit_Empty(t *testing.T) {
	is := is.New(t)
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	entities, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindPolicy, from, to)
	is.NoErr(err)
	is.Equal(len(entities), 0)
}

// --- GetAttributionSplit: user / device kinds and the fan-out dedup rule ---

// TestGetAttributionSplit_SharedIPMultiDeviceDedup is the key fan-out guard: a
// single shared-IP request that lists two devices of the SAME user must count
// once for that user (COUNT(DISTINCT access_log.id)) and once per device.
func TestGetAttributionSplit_SharedIPMultiDeviceDedup(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	hour := time.Date(2025, 3, 15, 14, 0, 0, 0, time.UTC)
	from := hour
	to := hour.Add(time.Hour)

	userID := int64(1)
	dev1, dev2 := int64(1), int64(2)
	addr1, addr2 := int64(1), int64(2)
	seedUser(t, db, userID, "alice", "Alice")
	seedDevice(t, db, dev1, userID, "alice-laptop")
	seedDevice(t, db, dev2, userID, "alice-phone")
	seedAddress(t, db, addr1, dev1, "10.0.0.1")
	seedAddress(t, db, addr2, dev2, "10.0.0.1")

	// One request, two contributors of the same user (shared IP, two devices).
	seedContributorAccessLogRow(t, db, "10.0.0.1", true, hour.Add(5*time.Minute),
		contributor{deviceID: dev1, addressID: addr1, userID: userID},
		contributor{deviceID: dev2, addressID: addr2, userID: userID},
	)

	// User split: the request counts once for Alice, not twice.
	users, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindUser, from, to)
	is.NoErr(err)
	is.Equal(len(users), 1)
	is.Equal(users[0].EntityName, "Alice")
	is.Equal(users[0].AllowCount, int64(1))
	is.Equal(users[0].DenyCount, int64(0))

	// Device split: once per device.
	devices, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindDevice, from, to)
	is.NoErr(err)
	byName := attributionByName(devices)
	is.Equal(len(devices), 2)
	is.Equal(byName["alice-laptop"].AllowCount, int64(1))
	is.Equal(byName["alice-phone"].AllowCount, int64(1))
}

// TestGetAttributionSplit_User_RawAndAggregatePathsAgree: the F18 cross-path
// guard for the user kind. The rolled-up aggregate path returns the same
// per-user allow/deny counts the raw path computes for the same window,
// including the shared-IP dedup.
func TestGetAttributionSplit_User_RawAndAggregatePathsAgree(t *testing.T) {
	is := is.New(t)
	repo, db := setupTestRepo(t)
	ctx := context.Background()

	currentHour := time.Now().UTC().Truncate(time.Hour)
	h1 := currentHour.Add(-3 * time.Hour)
	h2 := currentHour.Add(-2 * time.Hour)

	alice, bob := int64(1), int64(2)
	aliceLaptop, alicePhone, bobPhone := int64(1), int64(2), int64(3)
	aAddr1, aAddr2, bAddr := int64(1), int64(2), int64(3)
	seedUser(t, db, alice, "alice", "Alice")
	seedUser(t, db, bob, "bob", "Bob")
	seedDevice(t, db, aliceLaptop, alice, "alice-laptop")
	seedDevice(t, db, alicePhone, alice, "alice-phone")
	seedDevice(t, db, bobPhone, bob, "bob-phone")
	seedAddress(t, db, aAddr1, aliceLaptop, "10.0.0.1")
	seedAddress(t, db, aAddr2, alicePhone, "10.0.0.1")
	seedAddress(t, db, bAddr, bobPhone, "10.0.0.2")

	// Alice: a shared-IP request matching two of her devices (counts once), plus
	// a deny. Bob: one allow.
	seedContributorAccessLogRow(t, db, "10.0.0.1", true, h1.Add(5*time.Minute),
		contributor{deviceID: aliceLaptop, addressID: aAddr1, userID: alice},
		contributor{deviceID: alicePhone, addressID: aAddr2, userID: alice},
	)
	seedContributorAccessLogRow(t, db, "10.0.0.1", false, h2.Add(5*time.Minute),
		contributor{deviceID: aliceLaptop, addressID: aAddr1, userID: alice},
	)
	seedContributorAccessLogRow(t, db, "10.0.0.2", true, h2.Add(6*time.Minute),
		contributor{deviceID: bobPhone, addressID: bAddr, userID: bob},
	)

	is.NoErr(newTestJob(repo).Run(ctx))

	raw, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindUser, currentHour.Add(-4*time.Hour), currentHour)
	is.NoErr(err)
	agg, err := repo.GetAttributionSplit(ctx, rollup.AttributionKindUser, currentHour.Add(-48*time.Hour), currentHour)
	is.NoErr(err)

	rawByName := attributionByName(raw)
	aggByName := attributionByName(agg)
	is.Equal(len(rawByName), len(aggByName))
	is.Equal(rawByName["Alice"].AllowCount, int64(1)) // shared-IP request deduped
	is.Equal(rawByName["Alice"].DenyCount, int64(1))
	for name, rc := range rawByName {
		ac := aggByName[name]
		is.Equal(rc.AllowCount, ac.AllowCount)
		is.Equal(rc.DenyCount, ac.DenyCount)
	}
}
