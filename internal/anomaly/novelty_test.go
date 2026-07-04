//go:build test

package anomaly

import (
	"context"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/matryer/is"
)

func TestNormalizeUA(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"version digits collapse", "Firefox/128.0.1", "Firefox/x.x.x"},
		{"different patch same shape", "Firefox/130.2.5", "Firefox/x.x.x"},
		{"distinct product differs", "Chrome/120.0", "Chrome/x.x"},
		{"whitespace collapses", "Mozilla/5.0   (X11;  Linux x86_64)", "Mozilla/x.x (Xx; Linux xx_x)"},
		{"no digits unchanged", "curl", "curl"},
		{"empty", "", ""},
		{"whitespace only", "   \t ", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(normalizeUA(tc.in), tc.want)
		})
	}
}

// TestNormalizeUA_VersionsCollapse_ProductsDoNot pins the core contract: two
// versions of one product share a fingerprint; two products do not.
func TestNormalizeUA_VersionsCollapse_ProductsDoNot(t *testing.T) {
	is := is.New(t)
	is.Equal(normalizeUA("Firefox/128.0"), normalizeUA("Firefox/131.0"))
	is.True(normalizeUA("Firefox/128.0") != normalizeUA("Chrome/128.0"))
}

const learningWindow = 7 * 24 * time.Hour

// noveltyScope covers every raw row and pins the clock plus the learning window.
func noveltyScope(now time.Time) Scope {
	return Scope{FromAccessLogID: 0, ToAccessLogID: 1 << 62, Now: now, Sensitivity: "medium", LearningWindow: learningWindow}
}

// warmProfileFirstSeen is old enough that the seeded device is past learning.
func warmProfileFirstSeen(now time.Time) time.Time { return now.Add(-8 * 24 * time.Hour) }

func TestNoveltyDetector_NewUA_WarmDevice_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	// An old profile of a different UA makes the device warm.
	seedProfile(t, db, 1, dimUserAgent, uaFingerprint(normalizeUA("OldBrowser/1.0")), warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	const ua = "Mozilla/5.0 Firefox/130.0"
	id := seedAllowUA(t, db, ua, now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)

	det := &noveltyDetector{reader: repo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindNewUserAgent)
	is.Equal(findings[0].Severity, SeverityWarning)
	is.Equal(findings[0].Evidence["user_agent"], ua)
	is.Equal(*findings[0].DeviceID, ids.DeviceID(1))
	// The sighting is reported for the job to persist regardless of the finding.
	is.Equal(len(det.ProfileObservations()), 1)
}

// TestNoveltyDetector_LearningDevice_NoFinding_ButProfiles: a device with no
// established baseline stays silent while its profile still populates.
func TestNoveltyDetector_LearningDevice_NoFinding_ButProfiles(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	id := seedAllowUA(t, db, "Mozilla/5.0 Firefox/130.0", now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)

	det := &noveltyDetector{reader: repo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
	// The profile is still learned during the learning window.
	is.Equal(len(det.ProfileObservations()), 1)
}

// TestNoveltyDetector_KnownUA_NoFinding: a UA already in the device's profile is
// familiar — no finding, but its recurrence is still recorded.
func TestNoveltyDetector_KnownUA_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	const ua = "Mozilla/5.0 Firefox/130.0"
	seedProfile(t, db, 1, dimUserAgent, uaFingerprint(normalizeUA(ua)), warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	id := seedAllowUA(t, db, ua, now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)

	det := &noveltyDetector{reader: repo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
	is.Equal(len(det.ProfileObservations()), 1)
}

// TestNoveltyDetector_UAVersionBump_NoFinding: a browser auto-update (same product,
// new version) normalizes to the known fingerprint, so it does not flag.
func TestNoveltyDetector_UAVersionBump_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimUserAgent, uaFingerprint(normalizeUA("Mozilla/5.0 Firefox/128.0")), warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	id := seedAllowUA(t, db, "Mozilla/5.0 Firefox/131.0", now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)

	det := &noveltyDetector{reader: repo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

// TestNoveltyDetector_NoUserAgentHeader_Skips: a row with no User-Agent yields no
// finding and no profile.
func TestNoveltyDetector_NoUserAgentHeader_Skips(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimUserAgent, uaFingerprint(normalizeUA("OldBrowser/1.0")), warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	id := seedAllow(t, db, "203.0.113.5", "app.example.com", now.Add(-time.Minute)) // headers '{}'
	seedContributor(t, db, id, 1, 1)

	det := &noveltyDetector{reader: repo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
	is.Equal(len(det.ProfileObservations()), 0)
}

func TestNoveltyDetector_NewCountry_AddressFeed_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimCountry, "US", warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "198.51.100.7", true, now.Add(-time.Hour))
	seedEnableEvent(t, db, 1, now.Add(-30*time.Minute))

	geo := fakeGeo{byIP: map[string]geoip.Result{"198.51.100.7": {CountryCode: "DE", ContinentCode: "EU"}}}
	det := &noveltyDetector{reader: repo, geo: geo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindNewCountry)
	is.Equal(*findings[0].CountryCode, "DE")
}

func TestNoveltyDetector_NewCountry_TrafficFeed_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimCountry, "US", warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "203.0.113.9", true, now.Add(-time.Hour))
	id := seedAllow(t, db, "203.0.113.9", "app.example.com", now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)
	seedGeoip(t, db, id, "FR")

	geo := fakeGeo{byIP: map[string]geoip.Result{}}
	det := &noveltyDetector{reader: repo, geo: geo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindNewCountry)
	is.Equal(*findings[0].CountryCode, "FR")
}

// TestNoveltyDetector_NilResolver_NoCountry: without GeoIP the country kind is
// skipped entirely — no findings and no country observations — while UA is
// unaffected (none seeded here).
func TestNoveltyDetector_NilResolver_NoCountry(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimCountry, "US", warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "198.51.100.7", true, now.Add(-time.Hour))
	seedEnableEvent(t, db, 1, now.Add(-30*time.Minute))

	det := &noveltyDetector{reader: repo, geo: nil}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
	is.Equal(len(det.ProfileObservations()), 0)
}

// TestNoveltyDetector_EmptyCountryResolution_Skips: an IP the resolver can't place
// (private range, gap) produces no country finding.
func TestNoveltyDetector_EmptyCountryResolution_Skips(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedProfile(t, db, 1, dimCountry, "US", warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "10.0.0.4", true, now.Add(-time.Hour))
	seedEnableEvent(t, db, 1, now.Add(-30*time.Minute))

	geo := fakeGeo{byIP: map[string]geoip.Result{}} // resolves to empty Result
	det := &noveltyDetector{reader: repo, geo: geo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
	is.Equal(len(det.ProfileObservations()), 0)
}

// TestNoveltyDetector_SharedIP_PerDevice: one allowed row with two contributing
// devices yields a finding and an observation for each device.
func TestNoveltyDetector_SharedIP_PerDevice(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedDevice(t, db, 2, "desktop")
	seedProfile(t, db, 1, dimUserAgent, uaFingerprint(normalizeUA("OldBrowser/1.0")), warmProfileFirstSeen(now))
	seedProfile(t, db, 2, dimUserAgent, uaFingerprint(normalizeUA("OldBrowser/1.0")), warmProfileFirstSeen(now))
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	seedAddress(t, db, 2, 2, "203.0.113.5", true, now.Add(-time.Hour))
	id := seedAllowUA(t, db, "Mozilla/5.0 Firefox/130.0", now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)
	seedContributor(t, db, id, 2, 2)

	det := &noveltyDetector{reader: repo}
	findings, err := det.Detect(context.Background(), noveltyScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 2)
	is.Equal(len(det.ProfileObservations()), 2)
	devices := map[ids.DeviceID]bool{}
	for _, f := range findings {
		is.Equal(f.Kind, KindNewUserAgent)
		devices[*f.DeviceID] = true
	}
	is.True(devices[ids.DeviceID(1)])
	is.True(devices[ids.DeviceID(2)])
}

// TestNoveltyDetector_ObservationsPersist drives the detector's observations
// through the repository upsert and confirms the profile row lands.
func TestNoveltyDetector_ObservationsPersist(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "203.0.113.5", true, now.Add(-time.Hour))
	id := seedAllowUA(t, db, "Mozilla/5.0 Firefox/130.0", now.Add(-time.Minute))
	seedContributor(t, db, id, 1, 1)

	det := &noveltyDetector{reader: repo}
	_, err := det.Detect(context.Background(), noveltyScope(now))
	is.NoErr(err)

	for _, o := range det.ProfileObservations() {
		is.NoErr(repo.UpsertDeviceProfile(context.Background(), o))
	}
	profiles, err := repo.DeviceProfiles(context.Background(), []int64{1})
	is.NoErr(err)
	is.Equal(len(profiles), 1)
	is.Equal(profiles[0].Dimension, dimUserAgent)
}
