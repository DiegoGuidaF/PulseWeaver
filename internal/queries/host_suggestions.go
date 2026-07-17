package queries

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
	"github.com/DiegoGuidaF/PulseWeaver/internal/hosts"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/rollup"
)

// hostSuggestionsWindow is how far back both the suggestions page
// (GetHostSuggestionsPage) and the dashboard's pending-suggestion count
// (countPendingHostSuggestions) look for hosts that have received traffic but
// are neither known nor ignored.
const hostSuggestionsWindow = 7 * 24 * time.Hour

// GetHostSuggestionsPage returns FQDNs seen in the access log that are not yet
// known hosts, plus the operator's ignored-host list.
func (r *Repository) GetHostSuggestionsPage(ctx context.Context) (httpapi.HostSuggestionsPage, error) {
	rawIgnored, err := r.ignoredHostSuggestions(ctx)
	if err != nil {
		return httpapi.HostSuggestionsPage{}, err
	}
	knownHosts, err := r.knownHostSet(ctx)
	if err != nil {
		return httpapi.HostSuggestionsPage{}, err
	}

	ignored := make([]httpapi.IgnoredHostSuggestion, len(rawIgnored))
	ignoredSet := make(map[string]bool, len(rawIgnored))
	for i, s := range rawIgnored {
		ignored[i] = httpapi.IgnoredHostSuggestion{
			Id:        s.ID,
			Fqdn:      s.FQDN,
			CreatedAt: httpapi.UTCTime(s.CreatedAt),
		}
		ignoredSet[hosts.NormaliseHost(s.FQDN)] = true
	}

	since := time.Now().UTC().Add(-hostSuggestionsWindow)
	merged, err := r.pendingHostSuggestions(ctx, since, knownHosts, ignoredSet)
	if err != nil {
		return httpapi.HostSuggestionsPage{}, fmt.Errorf("get host suggestions: %w", err)
	}

	suggestions := make([]httpapi.HostSuggestion, 0, len(merged))
	for fqdn, a := range merged {
		suggestions = append(suggestions, httpapi.HostSuggestion{
			Fqdn:        fqdn,
			FirstSeen:   httpapi.UTCTime(a.firstSeen),
			AllowedHits: a.allowedHits,
			DeniedHits:  a.deniedHits,
		})
	}
	slices.SortFunc(suggestions, func(a, b httpapi.HostSuggestion) int {
		if d := b.DeniedHits - a.DeniedHits; d != 0 {
			return d
		}
		if d := b.AllowedHits - a.AllowedHits; d != 0 {
			return d
		}
		return strings.Compare(a.Fqdn, b.Fqdn)
	})

	return httpapi.HostSuggestionsPage{Suggestions: suggestions, Ignored: ignored}, nil
}

// ignoredHostSuggestions returns the operator's ignored-host list, ordered by FQDN.
func (r *Repository) ignoredHostSuggestions(ctx context.Context) ([]hosts.IgnoredHostSuggestion, error) {
	const query = `
		SELECT id, fqdn, created_at
		FROM ignored_host_suggestions
		ORDER BY fqdn
	`
	var rows []hosts.IgnoredHostSuggestion
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("get ignored suggestions: %w", err)
	}
	return rows, nil
}

// knownHostSet returns the registered host FQDNs keyed by their normalised form,
// for excluding already-granted hosts from suggestions.
func (r *Repository) knownHostSet(ctx context.Context) (map[string]bool, error) {
	const query = `SELECT fqdn FROM hosts`
	var fqdns []string
	if err := r.db.SelectContext(ctx, &fqdns, query); err != nil {
		return nil, fmt.Errorf("get known hosts: %w", err)
	}
	set := make(map[string]bool, len(fqdns))
	for _, f := range fqdns {
		set[hosts.NormaliseHost(f)] = true
	}
	return set, nil
}

// hostSuggestionRow is one FQDN's hit counts over a queried range, sourced
// from either access_log or hourly_traffic_aggregates.
type hostSuggestionRow struct {
	FQDN        string          `db:"fqdn"`
	FirstSeen   database.DBTime `db:"first_seen"`
	AllowedHits int             `db:"allowed_hits"`
	DeniedHits  int             `db:"denied_hits"`
}

// hostSuggestionAggregate accumulates hit counts for one normalised FQDN
// across however many raw target_host variants (bare + port-suffixed,
// differently-cased aggregate rows) fold onto it.
type hostSuggestionAggregate struct {
	firstSeen   time.Time
	allowedHits int
	deniedHits  int
}

// pendingHostSuggestions is the single implementation behind both the
// suggestions page and the dashboard's pending-suggestion count: it windows,
// normalises, and excludes known/ignored hosts identically for both callers,
// so the two can never silently drift on filtering.
//
// Mirrors internal/rollup's RawWindowThreshold dispatch (rollup.go,
// traffic_reads.go): a window no wider than the threshold answers entirely
// from access_log. A wider window answers its already-complete portion from
// hourly_traffic_aggregates and merges in the current, not-yet-rolled hour
// from access_log. Unlike the read-only dashboard widgets — which tolerate a
// host's traffic being invisible for up to an hour — a host observed moments
// ago must surface immediately, since that is exactly the signal a
// suggestion exists to surface.
func (r *Repository) pendingHostSuggestions(ctx context.Context, since time.Time, knownHosts, ignoredSet map[string]bool) (map[string]*hostSuggestionAggregate, error) {
	now := time.Now().UTC()

	var rows []hostSuggestionRow
	if now.Sub(since) <= rollup.RawWindowThreshold {
		raw, err := r.rawHostSuggestionRows(ctx, since, now)
		if err != nil {
			return nil, err
		}
		rows = raw
	} else {
		currentHourStart := now.Truncate(time.Hour)
		aggRows, err := r.aggregateHostSuggestionRows(ctx, since, currentHourStart)
		if err != nil {
			return nil, err
		}
		tailRows, err := r.rawHostSuggestionRows(ctx, currentHourStart, now)
		if err != nil {
			return nil, err
		}
		rows = append(aggRows, tailRows...)
	}

	// The policy engine matches a requested host after stripping its port
	// (hosts.NormaliseHost), so suggestions aggregate on the same normalised
	// key: a service observed as app.example.com:8443 surfaces as a
	// suggestion for app.example.com, merged with any bare-host hits, and
	// disappears once app.example.com is granted or ignored. Exclusion
	// compares the normalised form, not the raw (possibly port-suffixed,
	// possibly differently-cased) target_host.
	merged := make(map[string]*hostSuggestionAggregate, len(rows))
	for _, row := range rows {
		fqdn := hosts.NormaliseHost(row.FQDN)
		if hosts.ValidateFQDN(fqdn) != nil || knownHosts[fqdn] || ignoredSet[fqdn] {
			continue
		}
		a := merged[fqdn]
		if a == nil {
			a = &hostSuggestionAggregate{firstSeen: row.FirstSeen.Time}
			merged[fqdn] = a
		} else if row.FirstSeen.Before(a.firstSeen) {
			a.firstSeen = row.FirstSeen.Time
		}
		a.allowedHits += row.AllowedHits
		a.deniedHits += row.DeniedHits
	}
	return merged, nil
}

// rawHostSuggestionsQuery scans access_log directly, bounded by the bare
// created_at range so LOWER() only runs over the narrowed window
// (sargable-predicates.md). Named (rather than inlined) so a test can run
// EXPLAIN QUERY PLAN against the exact statement the repository executes.
const rawHostSuggestionsQuery = `
	SELECT
		LOWER(al.target_host) AS fqdn,
		MIN(al.created_at)    AS first_seen,
		SUM(CASE WHEN al.outcome = 1 THEN 1 ELSE 0 END) AS allowed_hits,
		SUM(CASE WHEN al.outcome = 0 THEN 1 ELSE 0 END) AS denied_hits
	FROM access_log al
	WHERE al.target_host IS NOT NULL
	  AND al.created_at >= ? AND al.created_at < ?
	GROUP BY LOWER(al.target_host)
`

func (r *Repository) rawHostSuggestionRows(ctx context.Context, from, to time.Time) ([]hostSuggestionRow, error) {
	var rows []hostSuggestionRow
	if err := r.db.SelectContext(ctx, &rows, rawHostSuggestionsQuery, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("raw host suggestion rows: %w", err)
	}
	return rows, nil
}

// aggregateHostSuggestionsQuery scans hourly_traffic_aggregates instead of
// access_log for the already-rolled portion of a window wider than
// rollup.RawWindowThreshold. target_host is stored as ” rather than NULL at
// rollup time, mirroring the raw path's target_host IS NOT NULL filter.
const aggregateHostSuggestionsQuery = `
	SELECT
		target_host                                               AS fqdn,
		MIN(bucket_at)                                             AS first_seen,
		SUM(CASE WHEN outcome = 1 THEN request_count ELSE 0 END)  AS allowed_hits,
		SUM(CASE WHEN outcome = 0 THEN request_count ELSE 0 END)  AS denied_hits
	FROM hourly_traffic_aggregates
	WHERE target_host != ''
	  AND bucket_at >= ? AND bucket_at < ?
	GROUP BY target_host
`

func (r *Repository) aggregateHostSuggestionRows(ctx context.Context, from, to time.Time) ([]hostSuggestionRow, error) {
	var rows []hostSuggestionRow
	if err := r.db.SelectContext(ctx, &rows, aggregateHostSuggestionsQuery, from.UTC(), to.UTC()); err != nil {
		return nil, fmt.Errorf("aggregate host suggestion rows: %w", err)
	}
	return rows, nil
}
