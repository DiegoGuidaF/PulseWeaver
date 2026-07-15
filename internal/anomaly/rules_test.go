//go:build test

package anomaly

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

func TestExpiredAccessDetector_DenyInsideGrace_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	disableAt := time.Now().Add(-2 * time.Hour)
	seedAddress(t, db, 1, 1, "203.0.113.5", false, disableAt.Add(-24*time.Hour))
	seedDisableEvent(t, db, 1, disableAt)
	seedDeny(t, db, "203.0.113.5", "app.example.com", "ip_not_registered", disableAt.Add(10*time.Minute))

	findings, err := expiredAccessDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindExpiredAccess)
	is.Equal(findings[0].Severity, SeverityWarning)
	is.Equal(*findings[0].DeviceID, ids.DeviceID(1))
	is.Equal(*findings[0].UserID, ids.UserID(1))
	is.Equal(*findings[0].ClientIP, "203.0.113.5")
}

func TestExpiredAccessDetector_DenyOutsideGrace_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	disableAt := time.Now().Add(-3 * time.Hour)
	seedAddress(t, db, 1, 1, "203.0.113.5", false, disableAt.Add(-24*time.Hour))
	seedDisableEvent(t, db, 1, disableAt)
	// Deny 90 minutes after the disable — outside the 60-minute grace.
	seedDeny(t, db, "203.0.113.5", "app.example.com", "ip_not_registered", disableAt.Add(90*time.Minute))

	findings, err := expiredAccessDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

func TestExpiredAccessDetector_AllowedRow_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	disableAt := time.Now().Add(-2 * time.Hour)
	seedAddress(t, db, 1, 1, "203.0.113.5", false, disableAt.Add(-24*time.Hour))
	seedDisableEvent(t, db, 1, disableAt)
	seedAllow(t, db, "203.0.113.5", "app.example.com", disableAt.Add(10*time.Minute))

	findings, err := expiredAccessDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

// TestExpiredAccessDetector_SharedIP_OneFindingPerDevice: an IP that maps to two
// devices produces one finding per device, not one per deny row.
func TestExpiredAccessDetector_SharedIP_OneFindingPerDevice(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedDevice(t, db, 2, "phone")
	disableAt := time.Now().Add(-2 * time.Hour)
	seedAddress(t, db, 1, 1, "203.0.113.5", false, disableAt.Add(-24*time.Hour))
	seedAddress(t, db, 2, 2, "203.0.113.5", false, disableAt.Add(-24*time.Hour))
	seedDisableEvent(t, db, 1, disableAt)
	seedDisableEvent(t, db, 2, disableAt)
	seedDeny(t, db, "203.0.113.5", "app.example.com", "ip_not_registered", disableAt.Add(10*time.Minute))

	findings, err := expiredAccessDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 2)
}

func TestInvalidTokenDetector_SingleDeny_CriticalFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	seedDeny(t, db, "198.51.100.7", "app.example.com", "invalid_token", time.Now())

	findings, err := invalidTokenDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindInvalidToken)
	is.Equal(findings[0].Severity, SeverityCritical)
	is.Equal(*findings[0].ClientIP, "198.51.100.7")
}

// TestInvalidTokenDetector_RepeatSameDay_OneFinding: repeated denies from one
// source on one day collapse into a single (ip, day)-fingerprinted finding.
func TestInvalidTokenDetector_RepeatSameDay_OneFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	day := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	seedDeny(t, db, "198.51.100.7", "a.example.com", "invalid_token", day)
	seedDeny(t, db, "198.51.100.7", "b.example.com", "invalid_token", day.Add(2*time.Hour))

	findings, err := invalidTokenDetector{reader: repo}.Detect(context.Background(), scopeAll("medium"))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Fingerprint, "invalid_token:198.51.100.7:2026-03-01")
	is.Equal(findings[0].Evidence["deny_count"], int64(2))
}
