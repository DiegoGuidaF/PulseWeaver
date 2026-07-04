package anomaly

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// travelHopWindow is the trailing span the country-hop signal walks each pass.
// Re-evaluated every pass; the device+country-pair fingerprint keeps a stable
// hop as one open row rather than one per scan.
const travelHopWindow = 6 * time.Hour

// travelReader is the narrow persistence impossible_travel depends on.
type travelReader interface {
	EnabledAddresses(ctx context.Context) ([]AddressSightingRow, error)
	EnabledAddressSightings(ctx context.Context, since time.Time) ([]AddressSightingRow, error)
}

// travelDetector emits impossible_travel — per device, never per user (a user's
// phone abroad plus their home server at home is normal; devices are independent
// presences). GeoIP resolves country + ASN only, so both signals are
// country-granular; severity is capped at warning (the VPN false-positive family).
type travelDetector struct {
	reader travelReader
	geo    GeoResolver
}

func (d travelDetector) Family() Family { return FamilyNovelty }

func (d travelDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	if d.geo == nil {
		return nil, nil
	}

	byFP := map[string]Finding{}
	var order []string
	// Concurrent presence is added first, so a device that is both present in and
	// hopping between the same country pair keeps the current-state finding.
	addFinding := func(f Finding) {
		if _, ok := byFP[f.Fingerprint]; ok {
			return
		}
		byFP[f.Fingerprint] = f
		order = append(order, f.Fingerprint)
	}

	if err := d.detectConcurrent(ctx, sc, addFinding); err != nil {
		return nil, err
	}
	if err := d.detectHops(ctx, sc, addFinding); err != nil {
		return nil, err
	}

	out := make([]Finding, 0, len(order))
	for _, fp := range order {
		out = append(out, byFP[fp])
	}
	return out, nil
}

// countryInfo is the representative IP and ASN org for one country a device is
// present in — the evidence a finding cites.
type countryInfo struct {
	ip     string
	asnOrg string
}

// detectConcurrent flags a device whose currently-enabled addresses resolve to
// more than one country. No time math: it directly catches the stolen-key case
// where thief and owner are both active.
func (d travelDetector) detectConcurrent(ctx context.Context, sc Scope, addFinding func(Finding)) error {
	rows, err := d.reader.EnabledAddresses(ctx)
	if err != nil {
		return err
	}

	type presence struct {
		row       AddressSightingRow
		countries map[string]countryInfo
	}
	byDevice := map[int64]*presence{}
	var order []int64
	for _, row := range rows {
		res := d.geo.Resolve(row.IP)
		if res.CountryCode == "" {
			continue
		}
		p := byDevice[row.DeviceID]
		if p == nil {
			p = &presence{row: row, countries: map[string]countryInfo{}}
			byDevice[row.DeviceID] = p
			order = append(order, row.DeviceID)
		}
		if _, ok := p.countries[res.CountryCode]; !ok {
			p.countries[res.CountryCode] = countryInfo{ip: row.IP, asnOrg: res.ASNOrg}
		}
	}

	for _, deviceID := range order {
		p := byDevice[deviceID]
		if len(p.countries) < 2 {
			continue
		}
		codes := make([]string, 0, len(p.countries))
		for c := range p.countries {
			codes = append(codes, c)
		}
		slices.Sort(codes)

		ips := make([]string, 0, len(codes))
		var orgs []string
		for _, c := range codes {
			ips = append(ips, p.countries[c].ip)
			if org := p.countries[c].asnOrg; org != "" {
				orgs = append(orgs, org)
			}
		}
		f := d.baseFinding(p.row, travelFingerprint(deviceID, codes), sc.Now)
		f.Evidence = map[string]any{
			"signal":    "concurrent_presence",
			"countries": codes,
			"ips":       ips,
		}
		if len(orgs) > 0 {
			f.Evidence["asn_orgs"] = orgs
		}
		addFinding(f)
	}
	return nil
}

// resolvedSighting is one enabled (device, IP) with its country resolved — a link
// in a device's ordered movement chain.
type resolvedSighting struct {
	row       AddressSightingRow
	country   string
	continent string
	asnOrg    string
}

// detectHops flags a device whose consecutive enabled-address countries differ:
// a cross-continent change always fires; a same-continent one only when the flag
// is set (border commuting and carrier routing make those the noisiest signal).
func (d travelDetector) detectHops(ctx context.Context, sc Scope, addFinding func(Finding)) error {
	rows, err := d.reader.EnabledAddressSightings(ctx, sc.Now.Add(-travelHopWindow))
	if err != nil {
		return err
	}

	byDevice := map[int64][]AddressSightingRow{}
	var order []int64
	for _, row := range rows {
		if _, ok := byDevice[row.DeviceID]; !ok {
			order = append(order, row.DeviceID)
		}
		byDevice[row.DeviceID] = append(byDevice[row.DeviceID], row)
	}

	for _, deviceID := range order {
		var prev *resolvedSighting
		for _, row := range byDevice[deviceID] {
			res := d.geo.Resolve(row.IP)
			if res.CountryCode == "" {
				continue // an unresolvable hop endpoint carries no signal; keep the chain
			}
			cur := resolvedSighting{row: row, country: res.CountryCode, continent: res.ContinentCode, asnOrg: res.ASNOrg}
			if prev != nil && prev.country != cur.country {
				sameContinent := prev.continent != "" && prev.continent == cur.continent
				if !sameContinent || sc.TravelSameContinent {
					addFinding(d.hopFinding(*prev, cur, sameContinent))
				}
			}
			c := cur
			prev = &c
		}
	}
	return nil
}

func (d travelDetector) hopFinding(from, to resolvedSighting, sameContinent bool) Finding {
	f := d.baseFinding(to.row, travelFingerprint(to.row.DeviceID, []string{from.country, to.country}), to.row.CreatedAt.Time)
	f.Evidence = map[string]any{
		"signal":         "country_hop",
		"from_country":   from.country,
		"to_country":     to.country,
		"from_ip":        from.row.IP,
		"to_ip":          to.row.IP,
		"from_at":        from.row.CreatedAt.Time,
		"to_at":          to.row.CreatedAt.Time,
		"same_continent": sameContinent,
	}
	var orgs []string
	if from.asnOrg != "" {
		orgs = append(orgs, from.asnOrg)
	}
	if to.asnOrg != "" {
		orgs = append(orgs, to.asnOrg)
	}
	if len(orgs) > 0 {
		f.Evidence["asn_orgs"] = orgs
	}
	return f
}

// baseFinding fills the device attribution and envelope both signals share; the
// caller sets the evidence.
func (d travelDetector) baseFinding(row AddressSightingRow, fingerprint string, observedAt time.Time) Finding {
	deviceID := ids.DeviceID(row.DeviceID)
	userID := ids.UserID(row.UserID)
	return Finding{
		Kind:        KindImpossibleTravel,
		Severity:    SeverityWarning,
		Fingerprint: fingerprint,
		DeviceID:    &deviceID,
		DeviceName:  row.DeviceName,
		UserID:      &userID,
		UserName:    row.UserName,
		ObservedAt:  observedAt,
	}
}

// travelFingerprint keys a finding on the device and its sorted country pair, so a
// stable dual-presence and a repeated hop between the same two countries each stay
// a single open row.
func travelFingerprint(deviceID int64, codes []string) string {
	sorted := slices.Clone(codes)
	slices.Sort(sorted)
	return fmt.Sprintf("impossible_travel:%d:%s", deviceID, strings.Join(sorted, "-"))
}
