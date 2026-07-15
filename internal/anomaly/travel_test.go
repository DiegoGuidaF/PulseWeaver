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

// travelScope pins the clock; TravelSameContinent defaults off.
func travelScope(now time.Time) Scope {
	return Scope{Now: now, Sensitivity: "medium"}
}

func TestTravelDetector_DualCountryEnabled_OneFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "198.51.100.1", true, now.Add(-2*time.Hour))
	seedAddress(t, db, 2, 1, "203.0.113.1", true, now.Add(-2*time.Hour))

	geo := fakeGeo{byIP: map[string]geoip.Result{
		"198.51.100.1": {CountryCode: "US", ContinentCode: "NA"},
		"203.0.113.1":  {CountryCode: "DE", ContinentCode: "EU"},
	}}
	det := travelDetector{reader: repo, geo: geo}
	findings, err := det.Detect(context.Background(), travelScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindImpossibleTravel)
	is.Equal(findings[0].Severity, SeverityWarning)
	is.Equal(findings[0].Evidence["signal"], "concurrent_presence")
	is.Equal(*findings[0].DeviceID, ids.DeviceID(1))

	// Stable across passes: the same country pair keeps one fingerprint.
	second, err := det.Detect(context.Background(), travelScope(now))
	is.NoErr(err)
	is.Equal(len(second), 1)
	is.Equal(second[0].Fingerprint, findings[0].Fingerprint)
}

func TestTravelDetector_SingleCountry_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "198.51.100.1", true, now.Add(-2*time.Hour))
	seedAddress(t, db, 2, 1, "198.51.100.2", true, now.Add(-2*time.Hour))

	geo := fakeGeo{byIP: map[string]geoip.Result{
		"198.51.100.1": {CountryCode: "US", ContinentCode: "NA"},
		"198.51.100.2": {CountryCode: "US", ContinentCode: "NA"},
	}}
	det := travelDetector{reader: repo, geo: geo}
	findings, err := det.Detect(context.Background(), travelScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}

func TestTravelDetector_CrossContinentHop_Flags(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	// First address disabled, second created shortly after in another continent.
	seedAddress(t, db, 1, 1, "198.51.100.1", false, now.Add(-3*time.Hour))
	seedEnableEvent(t, db, 1, now.Add(-3*time.Hour))
	seedAddress(t, db, 2, 1, "203.0.113.1", true, now.Add(-2*time.Hour))
	seedEnableEvent(t, db, 2, now.Add(-2*time.Hour))

	geo := fakeGeo{byIP: map[string]geoip.Result{
		"198.51.100.1": {CountryCode: "US", ContinentCode: "NA"},
		"203.0.113.1":  {CountryCode: "DE", ContinentCode: "EU"},
	}}
	det := travelDetector{reader: repo, geo: geo}
	findings, err := det.Detect(context.Background(), travelScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 1)
	is.Equal(findings[0].Kind, KindImpossibleTravel)
	is.Equal(findings[0].Evidence["signal"], "country_hop")
	is.Equal(findings[0].Evidence["same_continent"], false)
}

// TestTravelDetector_SameContinentHop_RespectsFlag: a same-continent hop is silent
// by default and flags only when the flag is on.
func TestTravelDetector_SameContinentHop_RespectsFlag(t *testing.T) {
	seed := func(t *testing.T) (travelDetector, time.Time) {
		repo, db := newRepo(t)
		now := time.Now()
		seedUser(t, db)
		seedDevice(t, db, 1, "laptop")
		seedAddress(t, db, 1, 1, "198.51.100.1", false, now.Add(-3*time.Hour))
		seedEnableEvent(t, db, 1, now.Add(-3*time.Hour))
		seedAddress(t, db, 2, 1, "203.0.113.1", true, now.Add(-2*time.Hour))
		seedEnableEvent(t, db, 2, now.Add(-2*time.Hour))
		geo := fakeGeo{byIP: map[string]geoip.Result{
			"198.51.100.1": {CountryCode: "FR", ContinentCode: "EU"},
			"203.0.113.1":  {CountryCode: "DE", ContinentCode: "EU"},
		}}
		return travelDetector{reader: repo, geo: geo}, now
	}

	t.Run("flag off — silent", func(t *testing.T) {
		is := is.New(t)
		det, now := seed(t)
		sc := travelScope(now)
		sc.TravelSameContinent = false
		findings, err := det.Detect(context.Background(), sc)
		is.NoErr(err)
		is.Equal(len(findings), 0)
	})

	t.Run("flag on — flags", func(t *testing.T) {
		is := is.New(t)
		det, now := seed(t)
		sc := travelScope(now)
		sc.TravelSameContinent = true
		findings, err := det.Detect(context.Background(), sc)
		is.NoErr(err)
		is.Equal(len(findings), 1)
		is.Equal(findings[0].Evidence["same_continent"], true)
	})
}

func TestTravelDetector_NilResolver_NoFinding(t *testing.T) {
	is := is.New(t)
	repo, db := newRepo(t)
	now := time.Now()
	seedUser(t, db)
	seedDevice(t, db, 1, "laptop")
	seedAddress(t, db, 1, 1, "198.51.100.1", true, now.Add(-2*time.Hour))
	seedAddress(t, db, 2, 1, "203.0.113.1", true, now.Add(-2*time.Hour))

	det := travelDetector{reader: repo, geo: nil}
	findings, err := det.Detect(context.Background(), travelScope(now))

	is.NoErr(err)
	is.Equal(len(findings), 0)
}
