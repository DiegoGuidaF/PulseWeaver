//go:build test

package filterx_test

import (
	"errors"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
)

func sortRegistry() *filterx.Registry {
	return filterx.NewRegistry(
		map[string]filterx.ColumnSpec{},
		map[string]filterx.SortSpec{
			"created_at":  {Expr: "ral.created_at", Kind: filterx.KindTime},
			"duration_us": {Expr: "ral.duration_us", Kind: filterx.KindInt},
			"client_ip":   {Expr: "ral.client_ip", Kind: filterx.KindString},
			"target_host": {Expr: "ral.target_host", Kind: filterx.KindString, Nullable: true},
		},
		"ral.id",
	)
}

func TestOrderBy_AppendsTiebreaker(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	clause, err := reg.OrderBy("duration_us", "desc")
	is.NoErr(err)
	is.Equal(clause, "ral.duration_us DESC, ral.id DESC")

	clause, err = reg.OrderBy("client_ip", "asc")
	is.NoErr(err)
	is.Equal(clause, "ral.client_ip ASC, ral.id ASC")
}

func TestOrderBy_RejectsUnknownSortOrBadOrder(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	_, err := reg.OrderBy("nope", "desc")
	is.True(errors.Is(err, filterx.ErrInvalidFilter))

	_, err = reg.OrderBy("created_at", "sideways")
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestCursor_RoundTrip(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	token, err := reg.EncodeCursor("duration_us", "desc", int64(42), 1001)
	is.NoErr(err)

	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)
	is.Equal(cur.Sort, "duration_us")
	is.Equal(cur.Order, "desc")
	is.Equal(cur.ID, int64(1001))
}

func TestCursor_RoundTripTime(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	when := time.Date(2026, 5, 26, 19, 11, 15, 0, time.UTC)
	token, err := reg.EncodeCursor("created_at", "desc", when, 5)
	is.NoErr(err)

	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)
	// The decoded time value drives the keyset predicate's bound arg.
	pred, err := reg.Keyset(cur)
	is.NoErr(err)
	_, args, err := pred.ToSql()
	is.NoErr(err)
	is.True(len(args) > 0)
}

func TestDecodeCursor_RejectsGarbage(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	_, err := reg.DecodeCursor("not-base64!!")
	is.True(errors.Is(err, filterx.ErrInvalidFilter))

	_, err = reg.DecodeCursor("YWJj") // valid base64, not valid JSON token
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestKeyset_Desc(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	token, err := reg.EncodeCursor("duration_us", "desc", int64(42), 1001)
	is.NoErr(err)
	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)

	pred, err := reg.Keyset(cur)
	is.NoErr(err)
	query, args := sqlOf(t, pred)
	is.Equal(query, "(ral.duration_us < ? OR (ral.duration_us = ? AND ral.id < ?))")
	is.Equal(args, []any{int64(42), int64(42), int64(1001)})
}

func TestKeyset_Asc(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	token, err := reg.EncodeCursor("duration_us", "asc", int64(42), 1001)
	is.NoErr(err)
	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)

	pred, err := reg.Keyset(cur)
	is.NoErr(err)
	query, args := sqlOf(t, pred)
	is.Equal(query, "(ral.duration_us > ? OR (ral.duration_us = ? AND ral.id > ?))")
	is.Equal(args, []any{int64(42), int64(42), int64(1001)})
}

// A nullable sort column in DESC must also surface the NULL block (which sorts
// last) once paging past the non-NULL values.
func TestKeyset_NullableDesc_WithValue(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	token, err := reg.EncodeCursor("target_host", "desc", "example.com", 1001)
	is.NoErr(err)
	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)

	pred, err := reg.Keyset(cur)
	is.NoErr(err)
	query, _ := sqlOf(t, pred)
	is.Equal(query, "(ral.target_host < ? OR (ral.target_host = ? AND ral.id < ?) OR ral.target_host IS NULL)")
}

// When the cursor's value is NULL in DESC, only further NULL rows remain.
func TestKeyset_NullValueDesc(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	token, err := reg.EncodeCursor("target_host", "desc", nil, 1001)
	is.NoErr(err)
	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)

	pred, err := reg.Keyset(cur)
	is.NoErr(err)
	query, args := sqlOf(t, pred)
	is.Equal(query, "ral.target_host IS NULL AND ral.id < ?")
	is.Equal(args, []any{int64(1001)})
}

// When the cursor's value is NULL in ASC (NULLs first), all non-NULL rows follow
// plus later NULL rows.
func TestKeyset_NullValueAsc(t *testing.T) {
	is := is.New(t)
	reg := sortRegistry()

	token, err := reg.EncodeCursor("target_host", "asc", nil, 1001)
	is.NoErr(err)
	cur, err := reg.DecodeCursor(token)
	is.NoErr(err)

	pred, err := reg.Keyset(cur)
	is.NoErr(err)
	query, _ := sqlOf(t, pred)
	is.Equal(query, "(ral.target_host IS NOT NULL OR ral.target_host IS NULL AND ral.id > ?)")
}
