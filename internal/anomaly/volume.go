package anomaly

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

const (
	// baselineWindow is the trailing span whose complete buckets form the median.
	baselineWindow = 7 * 24 * time.Hour
	// firstRunBackfill bounds the observed range on the first scan (NULL cursor):
	// old anomalies are stale news, and backfilling all history would flood the
	// feed on upgrade.
	firstRunBackfill = 24 * time.Hour
	// maxEvidenceHostsGeo bounds the host sample stored in a geo finding.
	maxEvidenceHostsGeo = 20
)

// GeoResolver resolves an IP to geo/ASN data. Declared consumer-side (the
// rollup.GeoResolver precedent); *geoip.Lookup satisfies it. A nil resolver
// means the geo detector reports nothing.
type GeoResolver interface {
	Resolve(ip string) geoip.Result
}

// observedBucketStart is the exclusive lower bound of the buckets this pass
// evaluates: the persisted cursor, or now-24h on the first run.
func observedBucketStart(sc Scope) time.Time {
	if sc.FromBucket != nil {
		return *sc.FromBucket
	}
	return sc.ToBucket.Add(-firstRunBackfill)
}

// splitSeries partitions a series' buckets into the established history (before
// observedFrom) and the newly completed observed buckets (>= observedFrom).
func splitSeries(points []BucketCount, observedFrom time.Time) (history []int64, observed []BucketCount) {
	for _, p := range points {
		if p.BucketAt.Before(observedFrom) {
			history = append(history, p.Count)
		} else {
			observed = append(observed, p)
		}
	}
	return history, observed
}

// spikeSet collects findings keyed by fingerprint, keeping the largest-observed
// bucket per day so a continuing incident is one row carrying its worst hour.
type spikeSet struct {
	byFP     map[string]Finding
	observed map[string]int64
	order    []string
}

func newSpikeSet() *spikeSet {
	return &spikeSet{byFP: map[string]Finding{}, observed: map[string]int64{}}
}

func (s *spikeSet) add(fp string, observed int64, f Finding) {
	if prev, ok := s.observed[fp]; ok {
		if prev >= observed {
			return
		}
	} else {
		s.order = append(s.order, fp)
	}
	s.observed[fp] = observed
	s.byFP[fp] = f
}

func (s *spikeSet) findings() []Finding {
	out := make([]Finding, 0, len(s.order))
	for _, fp := range s.order {
		out = append(out, s.byFP[fp])
	}
	return out
}

// denySpikeReader reads the traffic series a spike is measured over.
type denySpikeReader interface {
	GlobalDenyBuckets(ctx context.Context, from, to time.Time) ([]BucketCount, error)
	HostTrafficBuckets(ctx context.Context, from, to time.Time) ([]HostBucketCount, error)
}

// denySpikeDetector flags hours whose request volume overshoots the trailing
// median: global denies, and per-host denies and allows (separately — the two
// outcomes mean different things but share the math).
type denySpikeDetector struct{ reader denySpikeReader }

func (d denySpikeDetector) Family() Family { return FamilyVolume }

func (d denySpikeDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	preset := presetFor(sc.Sensitivity)
	historyStart := sc.ToBucket.Add(-baselineWindow)
	observedFrom := observedBucketStart(sc)

	global, err := d.reader.GlobalDenyBuckets(ctx, historyStart, sc.ToBucket)
	if err != nil {
		return nil, err
	}
	hostRows, err := d.reader.HostTrafficBuckets(ctx, historyStart, sc.ToBucket)
	if err != nil {
		return nil, err
	}

	set := newSpikeSet()
	d.evalSeries(set, "global", false, "", global, observedFrom, preset)

	for series, points := range groupHostSeries(hostRows) {
		d.evalSeries(set, series.label(), series.allow, series.host, points, observedFrom, preset)
	}
	return set.findings(), nil
}

func (d denySpikeDetector) evalSeries(set *spikeSet, label string, allow bool, host string, points []BucketCount, observedFrom time.Time, preset Preset) {
	history, observed := splitSeries(points, observedFrom)
	multiplier, floor := preset.denyThreshold()
	severity := SeverityWarning
	outcome := "deny"
	if allow {
		multiplier, floor = preset.allowThreshold()
		severity = SeverityInfo
		outcome = "allow"
	}
	for _, bucket := range observed {
		verdict, ok := Evaluate(bucket.Count, history, multiplier, floor)
		if !ok {
			continue
		}
		f := Finding{
			Kind:        KindDenySpike,
			Severity:    severity,
			Fingerprint: fmt.Sprintf("deny_spike:%s:%s", label, bucket.BucketAt.UTC().Format(time.DateOnly)),
			Evidence: map[string]any{
				"outcome":   outcome,
				"series":    label,
				"observed":  verdict.Observed,
				"baseline":  verdict.Baseline,
				"threshold": verdict.Threshold,
			},
			ObservedAt: bucket.BucketAt.Time,
		}
		if host != "" {
			h := host
			f.TargetHost = &h
			f.Evidence["target_host"] = host
		}
		set.add(f.Fingerprint, bucket.Count, f)
	}
}

// hostSeriesKey identifies one per-host series; allow and deny are distinct
// series so their day fingerprints never collide.
type hostSeriesKey struct {
	host  string
	allow bool
}

func (k hostSeriesKey) label() string {
	if k.allow {
		return "allow:" + k.host
	}
	return k.host
}

func groupHostSeries(rows []HostBucketCount) map[hostSeriesKey][]BucketCount {
	series := map[hostSeriesKey][]BucketCount{}
	for _, row := range rows {
		key := hostSeriesKey{host: row.TargetHost, allow: row.Outcome == 1}
		series[key] = append(series[key], BucketCount{BucketAt: row.BucketAt, Count: row.Count})
	}
	return series
}

// entityDriftReader reads the per-entity attribution series.
type entityDriftReader interface {
	AttributionBuckets(ctx context.Context, from, to time.Time) ([]EntityBucketCount, error)
}

// entityDriftDetector flags a user/device/policy whose hourly volume drifts far
// above its own trailing median — the same math as deny_spike, scoped per entity.
type entityDriftDetector struct{ reader entityDriftReader }

func (d entityDriftDetector) Family() Family { return FamilyVolume }

func (d entityDriftDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	preset := presetFor(sc.Sensitivity)
	historyStart := sc.ToBucket.Add(-baselineWindow)
	observedFrom := observedBucketStart(sc)

	rows, err := d.reader.AttributionBuckets(ctx, historyStart, sc.ToBucket)
	if err != nil {
		return nil, err
	}

	set := newSpikeSet()
	for key, series := range groupEntitySeries(rows) {
		multiplier, floor := preset.denyThreshold()
		severity := SeverityWarning
		outcome := "deny"
		if key.allow {
			multiplier, floor = preset.allowThreshold()
			severity = SeverityInfo
			outcome = "allow"
		}
		history, observed := splitSeries(series.points, observedFrom)
		for _, bucket := range observed {
			verdict, ok := Evaluate(bucket.Count, history, multiplier, floor)
			if !ok {
				continue
			}
			f := Finding{
				Kind:     KindEntityDrift,
				Severity: severity,
				Fingerprint: fmt.Sprintf("entity_drift:%s:%s:%s:%s",
					key.kind, key.name, outcome, bucket.BucketAt.UTC().Format(time.DateOnly)),
				Evidence: map[string]any{
					"entity_kind": key.kind,
					"entity_name": key.name,
					"outcome":     outcome,
					"observed":    verdict.Observed,
					"baseline":    verdict.Baseline,
					"threshold":   verdict.Threshold,
				},
				ObservedAt: bucket.BucketAt.Time,
			}
			attributeEntity(&f, key.kind, key.name, series.entityID)
			set.add(f.Fingerprint, bucket.Count, f)
		}
	}
	return set.findings(), nil
}

type entitySeriesKey struct {
	kind  string
	name  string
	allow bool
}

type entitySeries struct {
	points   []BucketCount
	entityID *int64
}

func groupEntitySeries(rows []EntityBucketCount) map[entitySeriesKey]*entitySeries {
	series := map[entitySeriesKey]*entitySeries{}
	for _, row := range rows {
		key := entitySeriesKey{kind: row.EntityKind, name: row.EntityName, allow: row.Outcome == 1}
		s := series[key]
		if s == nil {
			s = &entitySeries{}
			series[key] = s
		}
		s.points = append(s.points, BucketCount{BucketAt: row.BucketAt, Count: row.Count})
		if s.entityID == nil && row.EntityID != nil {
			s.entityID = row.EntityID
		}
	}
	return series
}

// attributeEntity fills device/user attribution from a resolved entity id. A
// policy entity carries no device/user link — its name lives only in evidence.
func attributeEntity(f *Finding, kind, name string, entityID *int64) {
	switch kind {
	case "device":
		f.DeviceName = name
		if entityID != nil {
			id := ids.DeviceID(*entityID)
			f.DeviceID = &id
		}
	case "user":
		f.UserName = name
		if entityID != nil {
			id := ids.UserID(*entityID)
			f.UserID = &id
		}
	}
}

// geoDeniedReader reads the geo expected-set inputs and the denied-by-country
// series.
type geoDeniedReader interface {
	EnabledAddressIPs(ctx context.Context) ([]string, error)
	AllowedCountries(ctx context.Context, since time.Time) ([]string, error)
	DeniedCountryBuckets(ctx context.Context, from, to time.Time) ([]CountryBucketCount, error)
}

// geoDeniedDetector flags denied traffic from countries outside the deployment's
// expected set (enabled-address countries ∪ recently-allowed countries). Never
// critical — VPN-exit mismatch is an accepted false-positive source.
type geoDeniedDetector struct {
	reader geoDeniedReader
	geo    GeoResolver
}

func (d geoDeniedDetector) Family() Family { return FamilyVolume }

func (d geoDeniedDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	if d.geo == nil {
		return nil, nil
	}
	historyStart := sc.ToBucket.Add(-baselineWindow)

	expected, err := d.expectedCountries(ctx, historyStart)
	if err != nil {
		return nil, err
	}
	// No baseline of "normal" geography (no live IPs, no allowed history): every
	// country would look foreign, so skip rather than flag everything.
	if len(expected) == 0 {
		return nil, nil
	}

	rows, err := d.reader.DeniedCountryBuckets(ctx, observedBucketStart(sc), sc.ToBucket)
	if err != nil {
		return nil, err
	}
	return d.buildFindings(rows, expected), nil
}

func (d geoDeniedDetector) expectedCountries(ctx context.Context, since time.Time) (map[string]struct{}, error) {
	expected := map[string]struct{}{}
	ips, err := d.reader.EnabledAddressIPs(ctx)
	if err != nil {
		return nil, err
	}
	for _, ip := range ips {
		if code := d.geo.Resolve(ip).CountryCode; code != "" {
			expected[code] = struct{}{}
		}
	}
	allowed, err := d.reader.AllowedCountries(ctx, since)
	if err != nil {
		return nil, err
	}
	for _, code := range allowed {
		expected[code] = struct{}{}
	}
	return expected, nil
}

// geoAccumulator sums a country's denies across the observed buckets of one day.
type geoAccumulator struct {
	finding Finding
	hosts   map[string]struct{}
	count   int64
}

func (d geoDeniedDetector) buildFindings(rows []CountryBucketCount, expected map[string]struct{}) []Finding {
	byFP := map[string]*geoAccumulator{}
	var order []string
	for _, row := range rows {
		if _, ok := expected[row.CountryCode]; ok {
			continue
		}
		fp := fmt.Sprintf("geo_denied:%s:%s", row.CountryCode, row.BucketAt.UTC().Format(time.DateOnly))
		acc := byFP[fp]
		if acc == nil {
			code := row.CountryCode
			acc = &geoAccumulator{
				hosts: map[string]struct{}{},
				finding: Finding{
					Kind:        KindGeoDenied,
					Severity:    SeverityWarning,
					Fingerprint: fp,
					CountryCode: &code,
					Evidence: map[string]any{
						"country_code":   row.CountryCode,
						"country_name":   row.CountryName,
						"continent_code": row.ContinentCode,
					},
				},
			}
			if org := d.geo.Resolve(row.SampleIP).ASNOrg; org != "" {
				acc.finding.Evidence["asn_org"] = org
			}
			byFP[fp] = acc
			order = append(order, fp)
		}
		acc.count += row.Count
		for _, h := range splitHosts(row.Hosts) {
			if len(acc.hosts) < maxEvidenceHostsGeo {
				acc.hosts[h] = struct{}{}
			}
		}
		if row.BucketAt.After(acc.finding.ObservedAt) {
			acc.finding.ObservedAt = row.BucketAt.Time
		}
	}

	out := make([]Finding, 0, len(order))
	for _, fp := range order {
		acc := byFP[fp]
		acc.finding.Evidence["deny_count"] = acc.count
		if len(acc.hosts) > 0 {
			acc.finding.Evidence["hosts"] = sortedKeys(acc.hosts)
		}
		out = append(out, acc.finding)
	}
	return out
}

// sortedKeys returns a set's keys in a stable order, so evidence host samples are
// deterministic across scans.
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
