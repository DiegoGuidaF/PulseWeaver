package anomaly

import "slices"

// minHistory is the smallest number of trailing buckets that makes a median
// trustworthy. Below it a series is too young to have a baseline and never
// flags — a brand-new host's or user's first day is not "abnormal".
const minHistory = 24

// Preset holds the sensitivity-scaled thresholds every detector family evaluates
// against. The operator picks a preset name, never raw numbers, so a
// misconfiguration cannot land in a state they can't reason about; every
// sensitivity number in the system lives in this one table.
type Preset struct {
	// Windowed fixed thresholds (rule/probing family, task 02).
	ProbingDistinctHosts     int
	AddressChurnNewAddresses int
	// Statistical dual thresholds (volume family). Deny and allow series use
	// different pairs: an allow spike means legit-load-or-compromise, a deny
	// spike means scan/probe — same math, different floors.
	DenyMultiplier  int64
	DenyFloor       int64
	AllowMultiplier int64
	AllowFloor      int64
}

// denyThreshold / allowThreshold expose the pair the caller's outcome selects.
func (p Preset) denyThreshold() (multiplier, floor int64)  { return p.DenyMultiplier, p.DenyFloor }
func (p Preset) allowThreshold() (multiplier, floor int64) { return p.AllowMultiplier, p.AllowFloor }

// presetFor resolves a sensitivity name into its threshold set. Config
// validation already rejects unknown names, so the medium fallback is defensive
// rather than a supported path.
func presetFor(sensitivity string) Preset {
	switch sensitivity {
	case "low":
		return Preset{
			ProbingDistinctHosts: 8, AddressChurnNewAddresses: 15,
			DenyMultiplier: 6, DenyFloor: 40, AllowMultiplier: 8, AllowFloor: 100,
		}
	case "high":
		return Preset{
			ProbingDistinctHosts: 3, AddressChurnNewAddresses: 6,
			DenyMultiplier: 3, DenyFloor: 10, AllowMultiplier: 4, AllowFloor: 25,
		}
	default: // medium
		return Preset{
			ProbingDistinctHosts: 5, AddressChurnNewAddresses: 10,
			DenyMultiplier: 4, DenyFloor: 20, AllowMultiplier: 6, AllowFloor: 50,
		}
	}
}

// Verdict carries the numbers behind a flag so evidence copy — "48 denials vs a
// median of 3" — comes straight from it.
type Verdict struct {
	Observed  int64
	Baseline  int64
	Threshold int64
}

// Evaluate reports whether observed is anomalous against history's median.
// ok is false — never a flag — when history is shorter than minHistory (the
// silence rule) so a young series is left alone. Flag when
// observed > max(floor, multiplier × median): the floor stops a 1→8 blip
// flagging like a 1k→8k surge; the multiplier scales with the baseline.
func Evaluate(observed int64, history []int64, multiplier, floor int64) (Verdict, bool) {
	if len(history) < minHistory {
		return Verdict{}, false
	}
	baseline := median(history)
	threshold := max(floor, multiplier*baseline)
	if observed > threshold {
		return Verdict{Observed: observed, Baseline: baseline, Threshold: threshold}, true
	}
	return Verdict{}, false
}

// median returns the middle value of a copy of values (even counts average the
// two middles). Median, not mean, so one prior spike cannot drag the baseline up
// and mask the next one.
func median(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	s := slices.Clone(values)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
