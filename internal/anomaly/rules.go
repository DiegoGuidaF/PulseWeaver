package anomaly

import (
	"context"
	"fmt"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
)

// expiredAccessGrace is how long after an address is disabled a subsequent
// ip_not_registered deny is still attributed to the lapse rather than to an
// unrelated scanner.
const expiredAccessGrace = 60 * time.Minute

// expiredAccessReader reads ip_not_registered denies whose IP was disabled just
// before the deny. Narrow consumer-side interface: *Repository satisfies it.
type expiredAccessReader interface {
	ExpiredAccessDenials(ctx context.Context, fromID, toID int64, grace time.Duration) ([]ExpiredAccessRow, error)
}

// expiredAccessDetector flags users silently losing access: their device's lease
// expired, the address was disabled, and their next request was denied.
type expiredAccessDetector struct{ reader expiredAccessReader }

func (d expiredAccessDetector) Family() Family { return FamilyRules }

func (d expiredAccessDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	rows, err := d.reader.ExpiredAccessDenials(ctx, sc.FromAccessLogID, sc.ToAccessLogID, expiredAccessGrace)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(rows))
	for _, row := range rows {
		deviceID := ids.DeviceID(row.DeviceID)
		userID := ids.UserID(row.UserID)
		clientIP := row.ClientIP
		findings = append(findings, Finding{
			Kind:        KindExpiredAccess,
			Severity:    SeverityWarning,
			Fingerprint: fmt.Sprintf("expired_access:%d:%s", row.DeviceID, row.ClientIP),
			DeviceID:    &deviceID,
			DeviceName:  row.DeviceName,
			UserID:      &userID,
			UserName:    row.UserName,
			ClientIP:    &clientIP,
			Evidence: map[string]any{
				"deny_count":   row.DenyCount,
				"disabled_at":  row.DisabledAt.Time,
				"lease_source": row.LeaseSource,
				"first_seen":   row.FirstSeen.Time,
				"last_seen":    row.LastSeen.Time,
			},
			ObservedAt: row.LastSeen.Time,
		})
	}
	return findings, nil
}

// invalidTokenReader reads invalid_token denies grouped by source and day.
type invalidTokenReader interface {
	InvalidTokenDenials(ctx context.Context, fromID, toID int64) ([]InvalidTokenRow, error)
}

// invalidTokenDetector flags invalid_token denies: either the proxy's bearer
// token broke (the whole install is silently failing) or a non-proxy caller is
// hitting the verify endpoint. Always critical.
type invalidTokenDetector struct{ reader invalidTokenReader }

func (d invalidTokenDetector) Family() Family { return FamilyRules }

func (d invalidTokenDetector) Detect(ctx context.Context, sc Scope) ([]Finding, error) {
	rows, err := d.reader.InvalidTokenDenials(ctx, sc.FromAccessLogID, sc.ToAccessLogID)
	if err != nil {
		return nil, err
	}
	findings := make([]Finding, 0, len(rows))
	for _, row := range rows {
		clientIP := row.ClientIP
		evidence := map[string]any{
			"deny_count": row.DenyCount,
			"first_seen": row.FirstSeen.Time,
			"last_seen":  row.LastSeen.Time,
		}
		if hosts := splitHosts(row.TargetHosts); len(hosts) > 0 {
			evidence["target_hosts"] = hosts
		}
		findings = append(findings, Finding{
			Kind:        KindInvalidToken,
			Severity:    SeverityCritical,
			Fingerprint: fmt.Sprintf("invalid_token:%s:%s", row.ClientIP, row.UTCDay),
			ClientIP:    &clientIP,
			Evidence:    evidence,
			ObservedAt:  row.LastSeen.Time,
		})
	}
	return findings, nil
}
