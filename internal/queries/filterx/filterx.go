// Package filterx is the reusable column-allowlist registry for list queries.
//
// It translates caller-supplied filter values and a chosen sort into squirrel
// conditions, ORDER BY clauses, and keyset pagination predicates. Column
// identifiers and SQL expressions live in the registry as fixed constants — the
// registry *is* the allowlist (ADR-007): callers supply only values, which
// squirrel parameterises. Views declare one Registry and route their filters,
// sort, and cursor through it instead of hand-writing the SQL each time.
package filterx

import (
	"errors"
	"fmt"
	"slices"

	sq "github.com/Masterminds/squirrel"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// ErrInvalidFilter is returned for an unknown column, an operator a column does
// not allow, a missing value list where one is required, or an over-cap value
// list. Callers map it to a 400.
var ErrInvalidFilter = errors.New("invalid filter")

// MaxValues caps the number of values accepted in a single multi-value filter.
// It keeps request URLs small and the bound-parameter count far below SQLite's
// per-statement variable limit. Callers needing more should switch to a POST
// search endpoint with a JSON body.
const MaxValues = 200

// Operator is the closed set of filter operators a column may permit.
type Operator string

const (
	OpIn          Operator = "in"
	OpNotIn       Operator = "not_in"
	OpContains    Operator = "contains"
	OpNotContains Operator = "not_contains"
	OpIsNull      Operator = "is_null"
	OpNotNull     Operator = "not_null"
)

// ParseOperator maps a request string to an Operator, defaulting to OpIn when
// empty. An unrecognised value is rejected with ErrInvalidFilter.
func ParseOperator(s string) (Operator, error) {
	switch op := Operator(s); op {
	case "":
		return OpIn, nil
	case OpIn, OpNotIn, OpContains, OpNotContains, OpIsNull, OpNotNull:
		return op, nil
	default:
		return "", fmt.Errorf("%w: unknown operator %q", ErrInvalidFilter, s)
	}
}

// Relational describes a column filtered via an EXISTS subquery correlated to
// the parent row, rather than a scalar comparison — e.g. matching any
// access_log_contributors row for the parent access_log entry.
type Relational struct {
	// Correlated is the subquery up to but excluding the value predicate, e.g.
	//   "SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id"
	// in / not_in append "AND <ValueCol> IN (…)"; is_null / not_null wrap it in
	// NOT EXISTS / EXISTS as-is.
	Correlated string
	// ValueCol is the column the IN list is matched against, e.g. "c.device_id".
	ValueCol string
}

// ColumnSpec defines how one filterable column translates to SQL.
type ColumnSpec struct {
	// Expr is the scalar SQL column expression, e.g. "g.country_code". Empty when
	// Rel is set.
	Expr string
	// Nullable enables the NULL-safe rewrite for not_in / not_contains (so NULL
	// rows are not silently dropped) and is required for is_null / not_null.
	Nullable bool
	// Rel, when set, translates the column as an EXISTS subquery instead of a
	// scalar comparison.
	Rel *Relational
	// Ops is the set of operators this column allows.
	Ops []Operator
}

func (s ColumnSpec) allows(op Operator) bool {
	return slices.Contains(s.Ops, op)
}

// Filter is a request to constrain one column. Values is empty for is_null /
// not_null. For scalar string operators the values must be strings; for
// relational and scalar-id columns they are the column's value type.
type Filter struct {
	Column string
	Op     Operator
	Values []any
}

// Registry is the per-view allowlist of filterable columns and sortable columns.
type Registry struct {
	cols       map[string]ColumnSpec
	sorts      map[string]SortSpec
	tieBreaker string
}

// NewRegistry builds a registry from a column allowlist, a sortable-column
// allowlist, and the tiebreaker column appended to every ORDER BY and keyset
// predicate (a unique, non-null column such as the primary key).
func NewRegistry(cols map[string]ColumnSpec, sorts map[string]SortSpec, tieBreaker string) *Registry {
	return &Registry{cols: cols, sorts: sorts, tieBreaker: tieBreaker}
}

// check validates a filter against the registry and returns its column spec.
func (r *Registry) check(f Filter) (ColumnSpec, error) {
	spec, ok := r.cols[f.Column]
	if !ok {
		return ColumnSpec{}, fmt.Errorf("%w: unknown column %q", ErrInvalidFilter, f.Column)
	}
	if !spec.allows(f.Op) {
		return ColumnSpec{}, fmt.Errorf("%w: operator %q not allowed on %q", ErrInvalidFilter, f.Op, f.Column)
	}
	if f.Op == OpIsNull || f.Op == OpNotNull {
		return spec, nil
	}
	if len(f.Values) == 0 {
		return ColumnSpec{}, fmt.Errorf("%w: operator %q on %q requires at least one value", ErrInvalidFilter, f.Op, f.Column)
	}
	if len(f.Values) > MaxValues {
		return ColumnSpec{}, fmt.Errorf("%w: %q has %d values, limit is %d", ErrInvalidFilter, f.Column, len(f.Values), MaxValues)
	}
	return spec, nil
}

// Validate reports whether a filter is acceptable without building its SQL.
func (r *Registry) Validate(f Filter) error {
	_, err := r.check(f)
	return err
}

// Condition validates the filter and returns its squirrel condition.
func (r *Registry) Condition(f Filter) (sq.Sqlizer, error) {
	spec, err := r.check(f)
	if err != nil {
		return nil, err
	}
	if spec.Rel != nil {
		return relationalCondition(spec.Rel, f)
	}
	return scalarCondition(spec, f)
}

func scalarCondition(spec ColumnSpec, f Filter) (sq.Sqlizer, error) {
	switch f.Op {
	case OpIn:
		return sq.Eq{spec.Expr: f.Values}, nil
	case OpNotIn:
		notIn := sq.NotEq{spec.Expr: f.Values}
		if spec.Nullable {
			return sq.Or{sq.Expr(spec.Expr + " IS NULL"), notIn}, nil
		}
		return notIn, nil
	case OpContains, OpNotContains:
		cond, err := likeCondition(spec.Expr, f.Values, f.Op == OpNotContains)
		if err != nil {
			return nil, err
		}
		if f.Op == OpNotContains && spec.Nullable {
			return sq.Or{sq.Expr(spec.Expr + " IS NULL"), cond}, nil
		}
		return cond, nil
	case OpIsNull:
		return sq.Expr(spec.Expr + " IS NULL"), nil
	case OpNotNull:
		return sq.Expr(spec.Expr + " IS NOT NULL"), nil
	}
	return nil, fmt.Errorf("%w: unsupported operator %q", ErrInvalidFilter, f.Op)
}

// likeCondition builds a substring match across values: an OR of LIKEs for
// contains (match any) and an AND of NOT LIKEs for not_contains (match none).
// Wildcards in the input are escaped so they match literally.
func likeCondition(expr string, values []any, negate bool) (sq.Sqlizer, error) {
	parts := make([]sq.Sqlizer, 0, len(values))
	for _, v := range values {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%w: contains requires string values", ErrInvalidFilter)
		}
		pattern := "%" + database.EscapeLIKE(s) + "%"
		if negate {
			parts = append(parts, sq.Expr(expr+` NOT LIKE ? ESCAPE '\'`, pattern))
		} else {
			parts = append(parts, sq.Expr(expr+` LIKE ? ESCAPE '\'`, pattern))
		}
	}
	if negate {
		return sq.And(parts), nil
	}
	return sq.Or(parts), nil
}

func relationalCondition(rel *Relational, f Filter) (sq.Sqlizer, error) {
	switch f.Op {
	case OpIn, OpNotIn:
		inSQL, args, err := sq.Eq{rel.ValueCol: f.Values}.ToSql()
		if err != nil {
			return nil, fmt.Errorf("build relational IN: %w", err)
		}
		exists := "EXISTS"
		if f.Op == OpNotIn {
			exists = "NOT EXISTS"
		}
		return sq.Expr(exists+" ("+rel.Correlated+" AND "+inSQL+")", args...), nil
	case OpIsNull:
		// No correlated child row at all.
		return sq.Expr("NOT EXISTS (" + rel.Correlated + ")"), nil
	case OpNotNull:
		return sq.Expr("EXISTS (" + rel.Correlated + ")"), nil
	}
	return nil, fmt.Errorf("%w: unsupported operator %q for relational column", ErrInvalidFilter, f.Op)
}
