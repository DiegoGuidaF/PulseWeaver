package anomaly

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// hostProbingWindow / addressChurnWindow are the trailing windows the two
// cardinality detectors evaluate. They re-scan the whole window each pass and
// rely on the UTC-day fingerprint for idempotency rather than a watermark.
const (
	hostProbingWindow  = 24 * time.Hour
	addressChurnWindow = 24 * time.Hour
	// maxEvidenceHosts bounds the host sample stored in a finding's evidence.
	maxEvidenceHosts = 20
)

// hostProbingReader reads devices denied host_not_allowed across many distinct
// hosts within a window.
type hostProbingReader interface {
	HostProbingDenials(ctx context.Context, since time.Time, threshold int) ([]DeviceDenyRow, error)
}

// hostProbingDetector flags a known device denied across many distinct hosts — a
// patient probe that never spikes, so spike detection can't see it; the signal
// is distinct-host cardinality.
type hostProbingDetector struct{ reader hostProbingReader }

func (d hostProbingDetector) Family() Family { return FamilyVolume }

func (d hostProbingDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	threshold := presetFor(sc.Sensitivity).ProbingDistinctHosts
	rows, err := d.reader.HostProbingDenials(ctx, sc.Now.Add(-hostProbingWindow), threshold)
	if err != nil {
		return nil, err
	}
	utcDay := sc.Now.UTC().Format(time.DateOnly)
	findings := make([]Finding, 0, len(rows))
	for _, row := range rows {
		deviceID := ids.DeviceID(row.DeviceID)
		userID := ids.UserID(row.UserID)
		evidence := map[string]any{
			"distinct_hosts": row.DistinctHosts,
			"deny_count":     row.DenyCount,
			"threshold":      threshold,
			"first_seen":     row.FirstSeen.Time,
			"last_seen":      row.LastSeen.Time,
		}
		if hosts := splitHosts(row.Hosts); len(hosts) > 0 {
			evidence["hosts"] = hosts
		}
		findings = append(findings, Finding{
			Kind:        KindHostProbing,
			Severity:    SeverityWarning,
			Fingerprint: fmt.Sprintf("host_probing:%d:%s", row.DeviceID, utcDay),
			DeviceID:    &deviceID,
			DeviceName:  row.DeviceName,
			UserID:      &userID,
			UserName:    row.UserName,
			Evidence:    evidence,
			ObservedAt:  row.LastSeen.Time,
		})
	}
	return findings, nil
}

// addressChurnReader reads devices creating many addresses within a window.
type addressChurnReader interface {
	AddressChurn(ctx context.Context, since time.Time, threshold int) ([]AddressChurnRow, error)
}

// addressChurnDetector flags a device registering an abnormal number of new
// addresses in a day — key sharing or spoofing.
type addressChurnDetector struct{ reader addressChurnReader }

func (d addressChurnDetector) Family() Family { return FamilyVolume }

func (d addressChurnDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	threshold := presetFor(sc.Sensitivity).AddressChurnNewAddresses
	rows, err := d.reader.AddressChurn(ctx, sc.Now.Add(-addressChurnWindow), threshold)
	if err != nil {
		return nil, err
	}
	utcDay := sc.Now.UTC().Format(time.DateOnly)
	findings := make([]Finding, 0, len(rows))
	for _, row := range rows {
		deviceID := ids.DeviceID(row.DeviceID)
		userID := ids.UserID(row.UserID)
		findings = append(findings, Finding{
			Kind:        KindAddressChurn,
			Severity:    SeverityWarning,
			Fingerprint: fmt.Sprintf("address_churn:%d:%s", row.DeviceID, utcDay),
			DeviceID:    &deviceID,
			DeviceName:  row.DeviceName,
			UserID:      &userID,
			UserName:    row.UserName,
			Evidence: map[string]any{
				"new_addresses": row.NewAddresses,
				"threshold":     threshold,
				"first_seen":    row.FirstSeen.Time,
				"last_seen":     row.LastSeen.Time,
			},
			ObservedAt: row.LastSeen.Time,
		})
	}
	return findings, nil
}

// splitHosts turns a GROUP_CONCAT result into a bounded, empty-free slice for
// evidence. Nil input (no rows) and empty segments both yield no entries.
func splitHosts(concat *string) []string {
	if concat == nil || *concat == "" {
		return nil
	}
	parts := strings.Split(*concat, ",")
	hosts := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			hosts = append(hosts, p)
		}
		if len(hosts) == maxEvidenceHosts {
			break
		}
	}
	return hosts
}
