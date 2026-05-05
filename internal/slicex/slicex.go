// Package slicex provides slice operations absent from the stdlib slices package.
package slicex

import "cmp"

// Dedup returns a new slice with duplicate elements removed, preserving order.
// The first occurrence of each element is kept.
func Dedup[T comparable](s []T) []T {
	if s == nil {
		return nil
	}
	seen := make(map[T]struct{}, len(s))
	out := make([]T, 0, len(s))
	for _, v := range s {
		if _, dup := seen[v]; dup {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// Intersect returns elements present in both a and b.
// Both slices must be sorted in ascending order. Runs in O(n+m).
func Intersect[T cmp.Ordered](a, b []T) []T {
	result := make([]T, 0)
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch cmp.Compare(a[i], b[j]) {
		case 0:
			result = append(result, a[i])
			i++
			j++
		case -1:
			i++
		default:
			j++
		}
	}
	return result
}

// Diff returns elements in a that are NOT present in b.
// Both slices must be sorted in ascending order. Runs in O(n+m).
func Diff[T cmp.Ordered](a, b []T) []T {
	result := make([]T, 0)
	i, j := 0, 0
	for i < len(a) {
		if j >= len(b) || cmp.Compare(a[i], b[j]) < 0 {
			result = append(result, a[i])
			i++
		} else if a[i] == b[j] {
			i++
			j++
		} else {
			j++
		}
	}
	return result
}
