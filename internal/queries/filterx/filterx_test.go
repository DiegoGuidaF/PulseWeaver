//go:build test

package filterx_test

import (
	"errors"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/matryer/is"

	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
)

func testRegistry() *filterx.Registry {
	return filterx.NewRegistry(
		map[string]filterx.ColumnSpec{
			"client_ip": {
				Expr: "ral.client_ip",
				Ops:  []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpContains, filterx.OpNotContains},
			},
			"target_host": {
				Expr:     "ral.target_host",
				Nullable: true,
				Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpContains, filterx.OpNotContains, filterx.OpIsNull, filterx.OpNotNull},
			},
			"country_code": {
				Expr:     "g.country_code",
				Nullable: true,
				Ops:      []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpIsNull, filterx.OpNotNull},
			},
			"device": {
				Rel: &filterx.Relational{
					Correlated: "SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id",
					ValueCol:   "c.device_id",
				},
				Ops: []filterx.Operator{filterx.OpIn, filterx.OpNotIn, filterx.OpIsNull, filterx.OpNotNull},
			},
			"user": {
				Rel: &filterx.Relational{
					Correlated: "SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id",
					ValueCol:   "c.user_id",
				},
				Ops: []filterx.Operator{filterx.OpIn, filterx.OpNotIn},
			},
		},
		map[string]filterx.SortSpec{
			"created_at": {Expr: "ral.created_at", Kind: filterx.KindTime},
		},
		"ral.id",
	)
}

func sqlOf(t *testing.T, s sq.Sqlizer) (string, []any) {
	t.Helper()
	query, args, err := s.ToSql()
	if err != nil {
		t.Fatalf("ToSql: %v", err)
	}
	return query, args
}

func TestParseOperator(t *testing.T) {
	is := is.New(t)

	op, err := filterx.ParseOperator("")
	is.NoErr(err)
	is.Equal(op, filterx.OpIn) // empty defaults to in

	op, err = filterx.ParseOperator("not_in")
	is.NoErr(err)
	is.Equal(op, filterx.OpNotIn)

	_, err = filterx.ParseOperator("regex")
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestCondition_In(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "country_code", Op: filterx.OpIn, Values: []any{"DE", "US"}})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, "g.country_code IN (?,?)")
	is.Equal(args, []any{"DE", "US"})
}

// not_in on a nullable column must include NULL rows, otherwise SQLite's
// three-valued logic silently drops them (the load-bearing correctness rule).
func TestCondition_NotIn_NullableIncludesNull(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "country_code", Op: filterx.OpNotIn, Values: []any{"ES"}})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, "(g.country_code IS NULL OR g.country_code NOT IN (?))")
	is.Equal(args, []any{"ES"})
}

func TestCondition_NotIn_NonNullable(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "client_ip", Op: filterx.OpNotIn, Values: []any{"1.1.1.1"}})
	is.NoErr(err)
	query, _ := sqlOf(t, cond)
	is.Equal(query, "ral.client_ip NOT IN (?)")
}

func TestCondition_Contains(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "target_host", Op: filterx.OpContains, Values: []any{"api"}})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, `(ral.target_host LIKE ? ESCAPE '\')`)
	is.Equal(args, []any{"%api%"})
}

func TestCondition_Contains_MultiValueOrs(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "target_host", Op: filterx.OpContains, Values: []any{"api", "web"}})
	is.NoErr(err)
	query, _ := sqlOf(t, cond)
	is.Equal(query, `(ral.target_host LIKE ? ESCAPE '\' OR ral.target_host LIKE ? ESCAPE '\')`)
}

func TestCondition_Contains_EscapesWildcards(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "client_ip", Op: filterx.OpContains, Values: []any{"5_1"}})
	is.NoErr(err)
	_, args := sqlOf(t, cond)
	is.Equal(args, []any{`%5\_1%`}) // underscore escaped so it matches literally
}

func TestCondition_NotContains_NullableIncludesNull(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "target_host", Op: filterx.OpNotContains, Values: []any{"internal"}})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, `(ral.target_host IS NULL OR (ral.target_host NOT LIKE ? ESCAPE '\'))`)
	is.Equal(args, []any{"%internal%"})
}

func TestCondition_IsNull(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "country_code", Op: filterx.OpIsNull})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, "g.country_code IS NULL")
	is.Equal(len(args), 0)
}

func TestCondition_NotNull(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "country_code", Op: filterx.OpNotNull})
	is.NoErr(err)
	query, _ := sqlOf(t, cond)
	is.Equal(query, "g.country_code IS NOT NULL")
}

func TestCondition_Relational_In(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "device", Op: filterx.OpIn, Values: []any{int64(7), int64(8)}})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, "EXISTS (SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id AND c.device_id IN (?,?))")
	is.Equal(args, []any{int64(7), int64(8)})
}

func TestCondition_Relational_NotIn(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "device", Op: filterx.OpNotIn, Values: []any{int64(7)}})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, "NOT EXISTS (SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id AND c.device_id IN (?))")
	is.Equal(args, []any{int64(7)})
}

func TestCondition_Relational_IsNull(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "device", Op: filterx.OpIsNull})
	is.NoErr(err)
	query, args := sqlOf(t, cond)
	is.Equal(query, "NOT EXISTS (SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id)")
	is.Equal(len(args), 0)
}

func TestCondition_Relational_NotNull(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	cond, err := reg.Condition(filterx.Filter{Column: "device", Op: filterx.OpNotNull})
	is.NoErr(err)
	query, _ := sqlOf(t, cond)
	is.Equal(query, "EXISTS (SELECT 1 FROM access_log_contributors c WHERE c.access_log_id = ral.id)")
}

func TestCondition_RejectsUnknownColumn(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	_, err := reg.Condition(filterx.Filter{Column: "secret", Op: filterx.OpIn, Values: []any{"x"}})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestCondition_RejectsDisallowedOperator(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	// client_ip does not allow is_null.
	_, err := reg.Condition(filterx.Filter{Column: "client_ip", Op: filterx.OpIsNull})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))

	// user does not allow is_null either.
	_, err = reg.Condition(filterx.Filter{Column: "user", Op: filterx.OpIsNull})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestCondition_RejectsEmptyValuesForIn(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	_, err := reg.Condition(filterx.Filter{Column: "country_code", Op: filterx.OpIn})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestCondition_RejectsOverCap(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	values := make([]any, filterx.MaxValues+1)
	for i := range values {
		values[i] = "x"
	}
	_, err := reg.Condition(filterx.Filter{Column: "country_code", Op: filterx.OpIn, Values: values})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestValidate_MatchesCondition(t *testing.T) {
	is := is.New(t)
	reg := testRegistry()

	is.NoErr(reg.Validate(filterx.Filter{Column: "country_code", Op: filterx.OpIn, Values: []any{"DE"}}))
	is.True(errors.Is(reg.Validate(filterx.Filter{Column: "client_ip", Op: filterx.OpIsNull}), filterx.ErrInvalidFilter))
}
