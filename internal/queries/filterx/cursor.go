package filterx

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
)

// ValueKind classifies a sortable column's value so an opaque cursor can
// round-trip it through JSON without losing type fidelity.
type ValueKind int

const (
	KindString ValueKind = iota
	KindInt
	KindTime
)

// SortSpec describes a sortable column. Expr is the ORDER BY expression and the
// keyset comparison column; Kind drives cursor (de)serialization; Nullable
// selects NULL-aware keyset predicates (SQLite orders NULLs first in ASC, last
// in DESC, which the predicates below mirror).
type SortSpec struct {
	Expr     string
	Kind     ValueKind
	Nullable bool
}

func (s SortSpec) decodeValue(raw json.RawMessage) (any, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	switch s.Kind {
	case KindInt:
		var n int64
		if err := json.Unmarshal(raw, &n); err != nil {
			return nil, fmt.Errorf("%w: bad cursor int value", ErrInvalidFilter)
		}
		return n, nil
	case KindTime:
		var t time.Time
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, fmt.Errorf("%w: bad cursor time value", ErrInvalidFilter)
		}
		return t, nil
	default:
		var str string
		if err := json.Unmarshal(raw, &str); err != nil {
			return nil, fmt.Errorf("%w: bad cursor string value", ErrInvalidFilter)
		}
		return str, nil
	}
}

// Cursor is the decoded form of an opaque pagination token: the sort it was
// issued under, the last row's sort value (nil for a NULL value), and the last
// row's tiebreaker id.
type Cursor struct {
	Sort  string
	Order string
	ID    int64
	value any
}

type cursorToken struct {
	Sort  string          `json:"s"`
	Order string          `json:"o"`
	Value json.RawMessage `json:"v"`
	ID    int64           `json:"id"`
}

// SortableExists reports whether sortKey is an allowlisted sortable column.
func (r *Registry) SortableExists(sortKey string) bool {
	_, ok := r.sorts[sortKey]
	return ok
}

func normalizeOrder(order string) (string, error) {
	switch order {
	case "asc", "desc":
		return order, nil
	default:
		return "", fmt.Errorf("%w: order must be asc or desc", ErrInvalidFilter)
	}
}

// OrderBy returns the ORDER BY clause for a sort/order pair, always appending
// the tiebreaker so non-unique sort columns page deterministically.
func (r *Registry) OrderBy(sortKey, order string) (string, error) {
	spec, ok := r.sorts[sortKey]
	if !ok {
		return "", fmt.Errorf("%w: unknown sort column %q", ErrInvalidFilter, sortKey)
	}
	ord, err := normalizeOrder(order)
	if err != nil {
		return "", err
	}
	dir := strings.ToUpper(ord)
	return fmt.Sprintf("%s %s, %s %s", spec.Expr, dir, r.tieBreaker, dir), nil
}

// EncodeCursor produces an opaque token from the last row of a page: the active
// sort, order, the row's sort value, and its tiebreaker id. The token is
// server-issued only; clients pass it back verbatim.
func (r *Registry) EncodeCursor(sortKey, order string, value any, id int64) (string, error) {
	if _, ok := r.sorts[sortKey]; !ok {
		return "", fmt.Errorf("%w: unknown sort column %q", ErrInvalidFilter, sortKey)
	}
	if _, err := normalizeOrder(order); err != nil {
		return "", err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal cursor value: %w", err)
	}
	b, err := json.Marshal(cursorToken{Sort: sortKey, Order: order, Value: raw, ID: id})
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeCursor parses a token, validating the embedded sort/order against the
// registry and coercing the value to the sort column's kind.
func (r *Registry) DecodeCursor(token string) (Cursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Cursor{}, fmt.Errorf("%w: malformed cursor", ErrInvalidFilter)
	}
	var t cursorToken
	if err := json.Unmarshal(b, &t); err != nil {
		return Cursor{}, fmt.Errorf("%w: malformed cursor", ErrInvalidFilter)
	}
	spec, ok := r.sorts[t.Sort]
	if !ok {
		return Cursor{}, fmt.Errorf("%w: unknown sort %q in cursor", ErrInvalidFilter, t.Sort)
	}
	if _, err := normalizeOrder(t.Order); err != nil {
		return Cursor{}, err
	}
	val, err := spec.decodeValue(t.Value)
	if err != nil {
		return Cursor{}, err
	}
	return Cursor{Sort: t.Sort, Order: t.Order, ID: t.ID, value: val}, nil
}

// Keyset returns the WHERE predicate that selects rows strictly after the cursor
// row under its sort and order. The tiebreaker id breaks ties on the (non-unique)
// sort column so adjacent pages never skip or repeat rows. For nullable sort
// columns the predicate mirrors SQLite's default NULL placement (NULLs first in
// ASC, last in DESC).
func (r *Registry) Keyset(c Cursor) (sq.Sqlizer, error) {
	spec, ok := r.sorts[c.Sort]
	if !ok {
		return nil, fmt.Errorf("%w: unknown sort %q in cursor", ErrInvalidFilter, c.Sort)
	}
	expr := spec.Expr
	desc := c.Order == "desc"

	if c.value == nil {
		// The cursor row's sort value is NULL.
		if desc {
			// NULLs sort last in DESC, so only further NULLs remain.
			return sq.Expr(expr+" IS NULL AND "+r.tieBreaker+" < ?", c.ID), nil
		}
		// NULLs sort first in ASC: all non-NULL rows follow, plus later NULLs.
		return sq.Or{
			sq.Expr(expr + " IS NOT NULL"),
			sq.Expr(expr+" IS NULL AND "+r.tieBreaker+" > ?", c.ID),
		}, nil
	}

	if desc {
		ors := sq.Or{
			sq.Lt{expr: c.value},
			sq.And{sq.Eq{expr: c.value}, sq.Lt{r.tieBreaker: c.ID}},
		}
		if spec.Nullable {
			// NULLs sort after any non-NULL value in DESC.
			ors = append(ors, sq.Expr(expr+" IS NULL"))
		}
		return ors, nil
	}
	// ASC with a non-NULL value: any NULLs already preceded this row.
	return sq.Or{
		sq.Gt{expr: c.value},
		sq.And{sq.Eq{expr: c.value}, sq.Gt{r.tieBreaker: c.ID}},
	}, nil
}
