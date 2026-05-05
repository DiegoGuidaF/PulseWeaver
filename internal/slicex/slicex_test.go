//go:build test

package slicex_test

import (
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/slicex"
	"github.com/matryer/is"
)

func TestDedup(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want []int
	}{
		{"nil input", nil, nil},
		{"no duplicates", []int{1, 2, 3}, []int{1, 2, 3}},
		{"all duplicates", []int{1, 1, 1}, []int{1}},
		{"preserves order", []int{3, 1, 2, 1, 3}, []int{3, 1, 2}},
		{"first occurrence kept", []int{2, 1, 2, 3}, []int{2, 1, 3}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(slicex.Dedup(tc.in), tc.want)
		})
	}
}

func TestDedup_Strings(t *testing.T) {
	is := is.New(t)
	is.Equal(slicex.Dedup([]string{"a", "b", "a", "c", "b"}), []string{"a", "b", "c"})
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{"both empty", nil, nil, []string{}},
		{"a empty", nil, []string{"x"}, []string{}},
		{"b empty", []string{"x"}, nil, []string{}},
		{"disjoint", []string{"a", "b"}, []string{"c", "d"}, []string{}},
		{"partial", []string{"a", "b", "c"}, []string{"b", "c", "d"}, []string{"b", "c"}},
		{"identical", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(slicex.Intersect(tc.a, tc.b), tc.want)
		})
	}
}

func TestIntersect_Ints(t *testing.T) {
	is := is.New(t)
	is.Equal(slicex.Intersect([]int{1, 3, 5}, []int{2, 3, 4, 5}), []int{3, 5})
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{"both empty", nil, nil, []string{}},
		{"a empty", nil, []string{"x"}, []string{}},
		{"b empty", []string{"x", "y"}, nil, []string{"x", "y"}},
		{"all in b", []string{"a", "b"}, []string{"a", "b", "c"}, []string{}},
		{"partial", []string{"a", "b", "c"}, []string{"b"}, []string{"a", "c"}},
		{"disjoint", []string{"a", "b"}, []string{"c", "d"}, []string{"a", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			is.Equal(slicex.Diff(tc.a, tc.b), tc.want)
		})
	}
}

func TestDiff_Ints(t *testing.T) {
	is := is.New(t)
	is.Equal(slicex.Diff([]int{1, 2, 3, 5}, []int{2, 4, 5}), []int{1, 3})
}
