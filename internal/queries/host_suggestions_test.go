//go:build test

package queries

import (
	"strings"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/accesslog"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

// suggestionsTestRepos bundles the repositories a pendingHostSuggestions test
// needs: the queries repository under test, plus the two repositories used to
// seed raw and rolled-up traffic directly (there is no Seeder fixture for
// backdated access-log rows — see seeder.md's "raw seeding stays" carve-out).
type suggestionsTestRepos struct {
	queries   *Repository
	accessLog *accesslog.Repository
	rollup    *rollup.Repository
}

func setupSuggestionsRepos(t *testing.T) suggestionsTestRepos {
	t.Helper()
	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	return suggestionsTestRepos{
		queries:   NewRepository(db.DB()),
		accessLog: accesslog.NewRepository(db.DB()),
		rollup:    rollup.NewRepository(db.DB(), nil),
	}
}

// TestPendingHostSuggestions_RawRegime covers the ≤24h window: since is one
// hour ago, so pendingHostSuggestions must dispatch entirely to the raw
// access_log path (no aggregates exist at all in this test, so the aggregate
// path would return nothing — proving the raw path is what actually answers).
func TestPendingHostSuggestions_RawRegime(t *testing.T) {
	is := is.New(t)
	repos := setupSuggestionsRepos(t)
	ctx := t.Context()

	unknownHost := "new-app.internal"
	knownHost := "known-app.internal"
	ignoredHost := "ignored-app.internal"

	is.NoErr(repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "9.9.9.9", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &unknownHost, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "9.9.9.9", Outcome: true, TargetHost: &knownHost, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
		{ClientIP: "9.9.9.9", Outcome: false, DenyReason: new(policy.DenyReasonHostNotAllowed), TargetHost: &ignoredHost, CreatedAt: time.Now().UTC(), Headers: map[string][]string{}},
	}))

	since := time.Now().UTC().Add(-1 * time.Hour)
	is.True(time.Since(since) <= rollup.RawWindowThreshold) // exercising the raw-only branch

	knownHosts := map[string]bool{knownHost: true}
	ignoredSet := map[string]bool{ignoredHost: true}
	merged, err := repos.queries.pendingHostSuggestions(ctx, since, knownHosts, ignoredSet)
	is.NoErr(err)

	is.Equal(len(merged), 1)
	got, ok := merged[unknownHost]
	is.True(ok)
	is.Equal(got.deniedHits, 1)
	is.Equal(got.allowedHits, 0)
}

// TestPendingHostSuggestions_AggregateRegimeMergesCurrentHour covers the >24h
// window: since is 48h ago, so the already-complete portion must answer from
// hourly_traffic_aggregates while the current, not-yet-rolled hour is merged
// in from access_log. The old host's raw rows are deleted after rolling up
// (mirroring retention pruning), so it can only appear via the aggregate
// path — a direct proof the >24h path answers without access_log for that
// portion of the window.
func TestPendingHostSuggestions_AggregateRegimeMergesCurrentHour(t *testing.T) {
	is := is.New(t)
	repos := setupSuggestionsRepos(t)
	ctx := t.Context()

	oldHost := "old-app.internal"
	freshHost := "fresh-app.internal"

	now := time.Now().UTC()
	currentHourStart := now.Truncate(time.Hour)
	oldHour := currentHourStart.Add(-30 * time.Hour) // well within a 48h window, a complete past hour

	is.NoErr(repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "8.8.8.8", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &oldHost, CreatedAt: oldHour.Add(5 * time.Minute), Headers: map[string][]string{}},
		{ClientIP: "8.8.8.8", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &oldHost, CreatedAt: oldHour.Add(10 * time.Minute), Headers: map[string][]string{}},
	}))

	// Roll up everything up to the current hour boundary, then prune the raw
	// rows it just covered — exactly what the retention job does once data is
	// safely aggregated.
	is.NoErr(repos.rollup.RunRollup(ctx, oldHour.Add(-time.Hour), currentHourStart))
	_, err := repos.accessLog.DeleteOlderThan(ctx, currentHourStart)
	is.NoErr(err)

	// A fresh, not-yet-rolled-up row observed just now — must still surface
	// immediately via the raw-tail merge, not wait for the next rollup pass.
	is.NoErr(repos.accessLog.BatchInsert(ctx, []policy.DecisionEvent{
		{ClientIP: "7.7.7.7", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &freshHost, CreatedAt: now, Headers: map[string][]string{}},
	}))

	since := now.Add(-48 * time.Hour)
	is.True(time.Since(since) > rollup.RawWindowThreshold) // exercising the aggregate+merge branch

	merged, err := repos.queries.pendingHostSuggestions(ctx, since, map[string]bool{}, map[string]bool{})
	is.NoErr(err)

	old, ok := merged[oldHost]
	is.True(ok) // answered from hourly_traffic_aggregates alone — its raw rows were pruned
	is.Equal(old.deniedHits, 2)

	fresh, ok := merged[freshHost]
	is.True(ok) // answered from the raw current-hour tail, never rolled up
	is.Equal(fresh.deniedHits, 1)
}

// TestAggregateHostSuggestionsQuery_DoesNotScanAccessLog asserts via EXPLAIN
// QUERY PLAN that the aggregate-path query touches only
// hourly_traffic_aggregates — the whole point of routing >24h windows there.
func TestAggregateHostSuggestionsQuery_DoesNotScanAccessLog(t *testing.T) {
	is := is.New(t)
	repos := setupSuggestionsRepos(t)
	ctx := t.Context()

	rows, err := repos.queries.db.QueryxContext(ctx, "EXPLAIN QUERY PLAN "+aggregateHostSuggestionsQuery,
		time.Now().UTC().Add(-48*time.Hour), time.Now().UTC())
	is.NoErr(err)
	defer func() { is.NoErr(rows.Close()) }()

	var details []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		is.NoErr(rows.Scan(&id, &parent, &notused, &detail))
		details = append(details, detail)
	}
	is.NoErr(rows.Err())
	is.True(len(details) > 0)

	for _, d := range details {
		is.True(!strings.Contains(strings.ToLower(d), "access_log"))
	}
	is.True(strings.Contains(strings.ToLower(details[0]), "hourly_traffic_aggregates"))
}

// TestPendingHostSuggestions_ParityAcrossWindowRegimes seeds the identical FQDN
// set into a complete past hour and drives pendingHostSuggestions twice: once
// with a ≤24h window (raw path) and once with a >24h window after rolling up
// and pruning the raw rows (aggregate path). Both must produce the same hit
// counts for the same data — the F18-style single-source-per-window invariant,
// applied to suggestions.
func TestPendingHostSuggestions_ParityAcrossWindowRegimes(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()

	host := "parity-app.internal"
	oldHour := time.Now().UTC().Truncate(time.Hour).Add(-3 * time.Hour)
	events := []policy.DecisionEvent{
		{ClientIP: "6.6.6.6", Outcome: false, DenyReason: new(policy.DenyReasonIPNotRegistered), TargetHost: &host, CreatedAt: oldHour.Add(5 * time.Minute), Headers: map[string][]string{}},
		{ClientIP: "6.6.6.6", Outcome: true, TargetHost: &host, CreatedAt: oldHour.Add(10 * time.Minute), Headers: map[string][]string{}},
	}

	// Path A: raw regime — since (4h ago) still covers oldHour (3h ago) directly.
	rawRepos := setupSuggestionsRepos(t)
	is.NoErr(rawRepos.accessLog.BatchInsert(ctx, events))
	since := time.Now().UTC().Add(-4 * time.Hour)
	is.True(time.Since(since) <= rollup.RawWindowThreshold)
	rawMerged, err := rawRepos.queries.pendingHostSuggestions(ctx, since, map[string]bool{}, map[string]bool{})
	is.NoErr(err)

	// Path B: aggregate regime — oldHour is rolled up and its raw rows pruned,
	// so a wide (>24h) window can only answer it from hourly_traffic_aggregates.
	aggRepos := setupSuggestionsRepos(t)
	is.NoErr(aggRepos.accessLog.BatchInsert(ctx, events))
	currentHourStart := time.Now().UTC().Truncate(time.Hour)
	is.NoErr(aggRepos.rollup.RunRollup(ctx, oldHour, currentHourStart))
	_, err = aggRepos.accessLog.DeleteOlderThan(ctx, currentHourStart)
	is.NoErr(err)
	sinceWide := time.Now().UTC().Add(-48 * time.Hour)
	is.True(time.Since(sinceWide) > rollup.RawWindowThreshold)
	aggMerged, err := aggRepos.queries.pendingHostSuggestions(ctx, sinceWide, map[string]bool{}, map[string]bool{})
	is.NoErr(err)

	is.Equal(len(rawMerged), 1)
	is.Equal(len(aggMerged), 1)
	rawHost, ok := rawMerged[host]
	is.True(ok)
	aggHost, ok := aggMerged[host]
	is.True(ok)
	is.Equal(rawHost.deniedHits, aggHost.deniedHits)
	is.Equal(rawHost.allowedHits, aggHost.allowedHits)
}
