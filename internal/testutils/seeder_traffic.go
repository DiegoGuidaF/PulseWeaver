//go:build test

package testutils

import (
	"math/rand"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/geoip"
	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
)

// TrafficStream is a weighted family of access-log events sharing an outcome and
// attribution. generateTraffic emits Count events from it, sampling the per-event
// dimensions (client IP, host, method, URI, duration) uniformly.
//
// Attribution is mutually exclusive: a device stream (Devices set) produces
// access_log_contributors rows, a policy stream (PolicyName set) produces a
// network-policy contributor, and neither set produces an unattributed row (the
// internet-noise / unknown-IP path). For a device stream every listed device
// must own a seeded address equal to the sampled ClientIP, so the simplest and
// safest shape is one device with its own IP per stream.
type TrafficStream struct {
	Count      int                // number of events to emit
	ClientIPs  []string           // sampled per event; required
	Outcome    bool               // true = allow, false = deny
	DenyReason *policy.DenyReason // required when Outcome is false
	Hosts      []string           // target_host sampled per event; optional
	Methods    []string           // HTTP method sampled per event; optional
	URIs       []string           // target_uri sampled per event; optional
	Devices    []string           // contributor device names; optional
	PolicyName string             // network-policy attribution; optional
	Geo        *geoip.Result      // geoip child row written for every event; optional

	// Duration is sampled in [MinDurationUs, MaxDurationUs]; defaults to a small
	// decision-latency range when both are zero.
	MinDurationUs int64
	MaxDurationUs int64
}

// TrafficProfile configures a run of generated, time-distributed traffic. Its
// events spread across Window ending at now, so dashboard time-series widgets
// render a curve rather than the single spike that fixed-timestamp fixtures give.
type TrafficProfile struct {
	Window  time.Duration // spread ending now; defaults to 24h
	Diurnal bool          // weight timestamps toward daytime (07:00–22:00 UTC)
	Seed    int64         // RNG seed for deterministic output
	Streams []TrafficStream
}

// WithGeneratedTraffic registers a profile of weighted, time-spread access-log
// traffic to emit during Build. Opt-in — it does not run for plain integration
// tests. Attribution and geo are resolved against the seeded world, so devices,
// addresses and policies referenced by a stream must be seeded in the same Build.
func (s *Seeder) WithGeneratedTraffic(p TrafficProfile) *Seeder {
	s.trafficProfile = &p
	return s
}

// generateTraffic materialises a TrafficProfile into DecisionEvents with
// timestamps spread across the window. Deterministic for a given Seed.
func (s *Seeder) generateTraffic(p TrafficProfile, result *SeedResult, deviceOwnerByName map[string]string) []policy.DecisionEvent {
	window := p.Window
	if window <= 0 {
		window = 24 * time.Hour
	}
	rng := rand.New(rand.NewSource(p.Seed))
	now := time.Now().UTC()
	sampleTime := newTimeSampler(rng, now, window, p.Diurnal)

	total := 0
	for _, st := range p.Streams {
		total += st.Count
	}
	events := make([]policy.DecisionEvent, 0, total)

	for _, st := range p.Streams {
		if st.Count > 0 && len(st.ClientIPs) == 0 {
			s.t.Fatalf("Seeder.WithGeneratedTraffic: stream with Count=%d has no ClientIPs", st.Count)
		}
		minDur, maxDur := st.MinDurationUs, st.MaxDurationUs
		if minDur == 0 && maxDur == 0 {
			minDur, maxDur = 8, 140
		}
		for i := 0; i < st.Count; i++ {
			f := AccessLogEntryFixture{
				ClientIP:   sample(rng, st.ClientIPs),
				Outcome:    st.Outcome,
				DenyReason: st.DenyReason,
				PolicyName: st.PolicyName,
				Devices:    st.Devices,
				GeoIP:      st.Geo,
				DurationUs: randInt64(rng, minDur, maxDur),
			}
			if len(st.Hosts) > 0 {
				f.TargetHost = new(sample(rng, st.Hosts))
			}
			if len(st.Methods) > 0 {
				f.HTTPMethod = new(sample(rng, st.Methods))
			}
			if len(st.URIs) > 0 {
				f.TargetURI = new(sample(rng, st.URIs))
			}
			events = append(events, s.buildDecisionEvent(f, sampleTime(), result, deviceOwnerByName))
		}
	}
	return events
}

// buildDecisionEvent converts one access-log fixture into a DecisionEvent at the
// given time, resolving device-contributor or network-policy attribution against
// the seeded world. Shared by the explicit-entry and generated-traffic paths.
func (s *Seeder) buildDecisionEvent(
	f AccessLogEntryFixture,
	createdAt time.Time,
	result *SeedResult,
	deviceOwnerByName map[string]string,
) policy.DecisionEvent {
	e := policy.DecisionEvent{
		ClientIP:   f.ClientIP,
		Outcome:    f.Outcome,
		DenyReason: f.DenyReason,
		CreatedAt:  createdAt,
		Headers:    map[string][]string{},
		TargetHost: f.TargetHost,
		TargetURI:  f.TargetURI,
		HTTPMethod: f.HTTPMethod,
		DurationUs: f.DurationUs,
	}
	if f.GeoIP != nil {
		e.GeoIP = *f.GeoIP
	}
	switch {
	case f.PolicyName != "":
		policyID, ok := result.policies[f.PolicyName]
		if !ok {
			s.t.Fatalf("Seeder: access log entry references unknown policy %q", f.PolicyName)
		}
		e.MatchSource = policy.MatchSourceNetworkPolicy
		e.NetworkPolicyID = new(policyID)
		e.NetworkPolicyName = new(f.PolicyName)
	case len(f.Devices) > 0:
		contributors := make([]policy.IPContributor, 0, len(f.Devices))
		for _, devName := range f.Devices {
			deviceID, ok := result.devices[devName]
			if !ok {
				s.t.Fatalf("Seeder: access log entry references unknown device %q", devName)
			}
			addrID, ok := result.addresses[addressKey(devName, f.ClientIP)]
			if !ok {
				s.t.Fatalf("Seeder: access log entry for device %q: no address %q seeded — add WithAddress first", devName, f.ClientIP)
			}
			ownerName, ok := deviceOwnerByName[devName]
			if !ok {
				s.t.Fatalf("Seeder: access log entry: could not resolve owner for device %q", devName)
			}
			userID, ok := result.users[ownerName]
			if !ok {
				s.t.Fatalf("Seeder: access log entry: owner %q of device %q was not seeded", ownerName, devName)
			}
			contributors = append(contributors, policy.IPContributor{
				DeviceID: deviceID, AddressID: addrID, UserID: userID,
			})
		}
		e.IPContributors = contributors
		e.MatchSource = policy.MatchSourceDevice
	}
	return e
}

// newTimeSampler returns a closure that draws a timestamp within [now-window, now].
// When diurnal, hour-ago buckets are weighted toward daytime so the resulting
// series has a believable day/night shape; otherwise the draw is uniform.
func newTimeSampler(rng *rand.Rand, now time.Time, window time.Duration, diurnal bool) func() time.Time {
	if !diurnal {
		return func() time.Time {
			return now.Add(-time.Duration(rng.Int63n(int64(window))))
		}
	}
	hours := max(int(window.Hours()), 1)
	weights := make([]float64, hours)
	cumulative := 0.0
	for ago := range hours {
		h := now.Add(-time.Duration(ago) * time.Hour).Hour()
		w := 0.8
		if h >= 7 && h <= 22 {
			w = 4.0 + 2.0*(1-absf(14-float64(h))/8.0)
		}
		cumulative += w
		weights[ago] = cumulative
	}
	return func() time.Time {
		r := rng.Float64() * cumulative
		ago := 0
		for ago < hours-1 && r > weights[ago] {
			ago++
		}
		offset := time.Duration(ago)*time.Hour + time.Duration(rng.Int63n(int64(time.Hour)))
		if offset >= window {
			offset = window - time.Minute
		}
		return now.Add(-offset)
	}
}

func sample[T any](rng *rand.Rand, xs []T) T {
	return xs[rng.Intn(len(xs))]
}

func randInt64(rng *rand.Rand, lo, hi int64) int64 {
	if hi <= lo {
		return lo
	}
	return lo + rng.Int63n(hi-lo+1)
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
