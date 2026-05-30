// Package collate folds flat SQL rows into nested DTOs. It replaces the hand-written
// "seen map[ID]int" collapse idiom and the per-parent child-indexing maps used by the
// gather-and-merge read models in internal/queries.
package collate

// Collapse folds parent×child rows (e.g. a LEFT JOIN result) into one parent per key,
// in first-seen order. newParent builds the parent from the first row seen for a key;
// child extracts that row's optional child (return ok=false to skip, e.g. a NULL join);
// attach appends the child to its parent.
func Collapse[Row any, K comparable, Parent any, Child any](
	rows []Row,
	key func(Row) K,
	newParent func(Row) Parent,
	child func(Row) (Child, bool),
	attach func(*Parent, Child),
) []Parent {
	seen := make(map[K]int, len(rows))
	out := []Parent{}
	for _, r := range rows {
		k := key(r)
		idx, ok := seen[k]
		if !ok {
			idx = len(out)
			seen[k] = idx
			out = append(out, newParent(r))
		}
		if c, has := child(r); has {
			// Safe: &out[idx] is used only within this iteration and never retained
			// across an append (which could reallocate the backing array).
			attach(&out[idx], c)
		}
	}
	return out
}

// GroupByMap indexes items by key, transforming each into a value — the side-query
// "xByGroup" pattern. Order within each slice follows input order.
func GroupByMap[T any, K comparable, V any](items []T, key func(T) K, val func(T) V) map[K][]V {
	out := make(map[K][]V)
	for _, it := range items {
		k := key(it)
		out[k] = append(out[k], val(it))
	}
	return out
}

// OrEmpty returns a non-nil slice so JSON marshals to [] rather than null.
func OrEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
