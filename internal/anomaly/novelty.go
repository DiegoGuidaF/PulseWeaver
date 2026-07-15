package anomaly

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

const (
	// dimUserAgent / dimCountry are the device_profiles dimensions the novelty
	// family learns; they mirror the schema CHECK constraint.
	dimUserAgent = "user_agent"
	dimCountry   = "country"

	// noveltyAddressWindow trails the current pass for new_country's address-event
	// feed. Generous relative to the scan interval so a short outage cannot skip an
	// enable event; the profile makes re-reads across passes idempotent.
	noveltyAddressWindow = 24 * time.Hour
)

// noveltyReader is the narrow persistence the novelty detector depends on.
type noveltyReader interface {
	NewUserAgentRows(ctx context.Context, fromID, toID int64) ([]UARow, error)
	AllowedTrafficCountries(ctx context.Context, fromID, toID int64) ([]CountryTrafficRow, error)
	EnabledAddressSightings(ctx context.Context, since time.Time) ([]AddressSightingRow, error)
	DeviceProfiles(ctx context.Context, deviceIDs []int64) ([]DeviceProfileRow, error)
}

// noveltyDetector emits new_user_agent and new_country. The two kinds share one
// device_profiles store and one learning gate, so a single detector owns them: it
// loads the pass's baseline once, decides novelty, and reports both findings and
// the sightings for the job to persist. It performs no writes — the observations
// it accumulates are drained by the job into the scan transaction (ProfileLearner).
type noveltyDetector struct {
	reader noveltyReader
	geo    GeoResolver
	// pending holds the sightings the last Detect observed, handed to the job.
	pending []ProfileObservation
}

func (d *noveltyDetector) Family() Family { return FamilyNovelty }

// ProfileObservations returns the sightings the last Detect observed so the job
// can upsert them into device_profiles inside the scan transaction.
func (d *noveltyDetector) ProfileObservations() []ProfileObservation { return d.pending }

func (d *noveltyDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	d.pending = nil

	acc := newSightingAcc()

	uaRows, err := d.reader.NewUserAgentRows(ctx, sc.FromAccessLogID, sc.ToAccessLogID)
	if err != nil {
		return nil, err
	}
	for _, row := range uaRows {
		ua := userAgent(row.HeadersJSON)
		if ua == "" {
			continue
		}
		norm := normalizeUA(ua)
		if norm == "" {
			continue
		}
		acc.add(sighting{
			deviceID: row.DeviceID, deviceName: row.DeviceName,
			userID: row.UserID, userName: row.UserName,
			dimension: dimUserAgent, fingerprint: uaFingerprint(norm),
			seenAt: row.CreatedAt.Time,
			kind:   KindNewUserAgent, clientIP: row.ClientIP, fullUA: ua,
		})
	}

	// Country novelty is GeoIP-dependent: feed 1 resolves at scan time, feed 2
	// reads the log-time country. With no resolver, skip the kind entirely.
	if d.geo != nil {
		if err := d.collectCountries(ctx, sc, acc); err != nil {
			return nil, err
		}
	}

	if len(acc.order) == 0 {
		return nil, nil
	}

	profiles, err := d.loadProfiles(ctx, acc.deviceIDs())
	if err != nil {
		return nil, err
	}

	var findings []Finding
	d.pending = make([]ProfileObservation, 0, len(acc.order))
	for _, key := range acc.order {
		s := acc.byKey[key]
		d.pending = append(d.pending, ProfileObservation{
			DeviceID:    s.deviceID,
			Dimension:   s.dimension,
			Fingerprint: s.fingerprint,
			SeenAt:      s.seenAt,
		})
		if profiles.isKnown(key) {
			continue
		}
		// A device stays silent until its profile is old enough — a brand-new
		// device's every value would otherwise look novel. Profiles still populate.
		if !profiles.warm(s.deviceID, sc.Now, sc.LearningWindow) {
			continue
		}
		findings = append(findings, s.toFinding())
	}
	return findings, nil
}

func (d *noveltyDetector) collectCountries(ctx context.Context, sc Scope, acc *sightingAcc) error {
	// Feed 1 — enabled-address IPs resolved at scan time. The stolen-key guard:
	// a thief's heartbeat registers their IP before any traffic flows.
	sightings, err := d.reader.EnabledAddressSightings(ctx, sc.Now.Add(-noveltyAddressWindow))
	if err != nil {
		return err
	}
	for _, row := range sightings {
		code := d.geo.Resolve(row.IP).CountryCode
		if code == "" {
			continue
		}
		acc.add(sighting{
			deviceID: row.DeviceID, deviceName: row.DeviceName,
			userID: row.UserID, userName: row.UserName,
			dimension: dimCountry, fingerprint: code,
			seenAt: row.CreatedAt.Time,
			kind:   KindNewCountry, country: code,
		})
	}

	// Feed 2 — the persisted country on allowed traffic.
	trafficRows, err := d.reader.AllowedTrafficCountries(ctx, sc.FromAccessLogID, sc.ToAccessLogID)
	if err != nil {
		return err
	}
	for _, row := range trafficRows {
		acc.add(sighting{
			deviceID: row.DeviceID, deviceName: row.DeviceName,
			userID: row.UserID, userName: row.UserName,
			dimension: dimCountry, fingerprint: row.CountryCode,
			seenAt: row.CreatedAt.Time,
			kind:   KindNewCountry, country: row.CountryCode,
		})
	}
	return nil
}

func (d *noveltyDetector) loadProfiles(ctx context.Context, deviceIDs []int64) (profileState, error) {
	rows, err := d.reader.DeviceProfiles(ctx, deviceIDs)
	if err != nil {
		return profileState{}, err
	}
	ps := profileState{
		known:  make(map[sightingKey]struct{}, len(rows)),
		oldest: map[int64]time.Time{},
	}
	for _, row := range rows {
		ps.known[sightingKey{row.DeviceID, row.Dimension, row.Fingerprint}] = struct{}{}
		first := row.FirstSeenAt.Time
		if cur, ok := ps.oldest[row.DeviceID]; !ok || first.Before(cur) {
			ps.oldest[row.DeviceID] = first
		}
	}
	return ps, nil
}

// sighting is one (device, dimension, fingerprint) observed this pass, carrying
// the payload a finding needs if the value turns out novel.
type sighting struct {
	deviceID    int64
	deviceName  string
	userID      int64
	userName    string
	dimension   string
	fingerprint string
	seenAt      time.Time
	kind        Kind
	clientIP    string // user_agent dimension
	fullUA      string // user_agent dimension, evidence
	country     string // country dimension
}

func (s sighting) toFinding() Finding {
	deviceID := ids.DeviceID(s.deviceID)
	userID := ids.UserID(s.userID)
	f := Finding{
		Kind:       s.kind,
		Severity:   SeverityWarning,
		DeviceID:   &deviceID,
		DeviceName: s.deviceName,
		UserID:     &userID,
		UserName:   s.userName,
		ObservedAt: s.seenAt,
	}
	switch s.dimension {
	case dimUserAgent:
		f.Fingerprint = fmt.Sprintf("new_user_agent:%d:%s", s.deviceID, s.fingerprint)
		f.Evidence = map[string]any{"user_agent": s.fullUA, "ua_fingerprint": s.fingerprint}
		if s.clientIP != "" {
			ip := s.clientIP
			f.ClientIP = &ip
			f.Evidence["client_ip"] = s.clientIP
		}
	case dimCountry:
		code := s.country
		f.Fingerprint = fmt.Sprintf("new_country:%d:%s", s.deviceID, code)
		f.CountryCode = &code
		f.Evidence = map[string]any{"country_code": code}
	}
	return f
}

// sightingKey identifies a profile entry; it is both the per-pass dedup key and
// the device_profiles lookup key.
type sightingKey struct {
	device      int64
	dimension   string
	fingerprint string
}

// sightingAcc dedups sightings per key within a pass so repeated rows collapse to
// one profile observation and at most one finding, keeping the latest seenAt.
type sightingAcc struct {
	byKey map[sightingKey]sighting
	order []sightingKey
}

func newSightingAcc() *sightingAcc {
	return &sightingAcc{byKey: map[sightingKey]sighting{}}
}

func (a *sightingAcc) add(s sighting) {
	key := sightingKey{s.deviceID, s.dimension, s.fingerprint}
	if cur, ok := a.byKey[key]; ok {
		if s.seenAt.After(cur.seenAt) {
			cur.seenAt = s.seenAt
			a.byKey[key] = cur
		}
		return
	}
	a.byKey[key] = s
	a.order = append(a.order, key)
}

// deviceIDs returns the distinct devices seen this pass, in first-seen order.
func (a *sightingAcc) deviceIDs() []int64 {
	seen := map[int64]struct{}{}
	var ids []int64
	for _, key := range a.order {
		if _, ok := seen[key.device]; !ok {
			seen[key.device] = struct{}{}
			ids = append(ids, key.device)
		}
	}
	return ids
}

// profileState is the pass's novelty baseline: which fingerprints each device has
// already learned, and how old its oldest profile is (the learning gate).
type profileState struct {
	known  map[sightingKey]struct{}
	oldest map[int64]time.Time
}

func (ps profileState) isKnown(k sightingKey) bool {
	_, ok := ps.known[k]
	return ok
}

// warm reports whether a device has learned long enough to flag novelty: its
// oldest profile row must be older than the learning window. A device with no
// profile row yet is by definition still learning.
func (ps profileState) warm(device int64, now time.Time, window time.Duration) bool {
	first, ok := ps.oldest[device]
	if !ok {
		return false
	}
	return now.Sub(first) > window
}

// userAgent extracts the first User-Agent value from a stored headers_json blob
// (canonical map[string][]string). Empty when the header is absent or the blob
// does not parse.
func userAgent(headersJSON string) string {
	if headersJSON == "" {
		return ""
	}
	var headers map[string][]string
	if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
		return ""
	}
	if vals := headers["User-Agent"]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// normalizeUA collapses a User-Agent into a version-insensitive form: every run
// of digits becomes a single "x" and whitespace is collapsed. This absorbs
// browser auto-updates (Firefox/128.0.1 and Firefox/130.0.1 both become
// Firefox/x.x.x) while keeping distinct products and platforms apart. Pure string
// processing — no UA-parser dependency.
func normalizeUA(ua string) string {
	var b strings.Builder
	inDigits := false
	for _, r := range ua {
		if r >= '0' && r <= '9' {
			if !inDigits {
				b.WriteByte('x')
				inDigits = true
			}
			continue
		}
		inDigits = false
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// uaFingerprint hashes a normalized User-Agent so the stored fingerprint stays
// bounded regardless of UA length; the full string lives in the finding evidence.
func uaFingerprint(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
