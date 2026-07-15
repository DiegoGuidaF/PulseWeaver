//go:build test

package anomaly

import (
	"testing"

	"github.com/matryer/is"
)

func repeat(v int64, n int) []int64 {
	s := make([]int64, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func TestEvaluate(t *testing.T) {
	// A quiet baseline of 2 over 23 buckets plus one prior spike of 1000: the
	// median stays 2, proving one past spike can't poison the baseline.
	robustHistory := append(repeat(2, 23), 1000)

	cases := []struct {
		name          string
		observed      int64
		history       []int64
		multiplier    int64
		floor         int64
		wantFlag      bool
		wantBaseline  int64
		wantThreshold int64
	}{
		{"floor dominates small baseline", 25, repeat(2, 24), 4, 20, true, 2, 20},
		{"below floor no flag", 15, repeat(2, 24), 4, 20, false, 0, 0},
		{"multiplier dominates large baseline", 500, repeat(50, 24), 4, 20, true, 50, 200},
		{"below multiplier threshold", 150, repeat(50, 24), 4, 20, false, 0, 0},
		{"median robust to prior spike", 25, robustHistory, 4, 20, true, 2, 20},
		{"silence under min history", 1000, repeat(2, 23), 4, 20, false, 0, 0},
		{"empty history silenced", 1000, nil, 4, 20, false, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			verdict, flagged := Evaluate(tc.observed, tc.history, tc.multiplier, tc.floor)
			is.Equal(flagged, tc.wantFlag)
			if tc.wantFlag {
				is.Equal(verdict.Observed, tc.observed)
				is.Equal(verdict.Baseline, tc.wantBaseline)
				is.Equal(verdict.Threshold, tc.wantThreshold)
			}
		})
	}
}

func TestMedian(t *testing.T) {
	is := is.New(t)
	is.Equal(median([]int64{5}), int64(5))
	is.Equal(median([]int64{1, 3}), int64(2))         // even → average of middles
	is.Equal(median([]int64{9, 1, 5}), int64(5))      // unsorted input
	is.Equal(median([]int64{1, 2, 3, 100}), int64(2)) // outlier does not move it
}

func TestPresetFor_AllSensitivities(t *testing.T) {
	is := is.New(t)
	low, medium, high := presetFor("low"), presetFor("medium"), presetFor("high")

	is.Equal(low.DenyMultiplier, int64(6))
	is.Equal(low.DenyFloor, int64(40))
	is.Equal(medium.DenyMultiplier, int64(4))
	is.Equal(medium.AllowFloor, int64(50))
	is.Equal(high.DenyFloor, int64(10))
	is.Equal(high.AllowMultiplier, int64(4))

	// Unknown value falls back to medium (config validation rejects it upstream).
	is.Equal(presetFor("nonsense"), medium)
}
