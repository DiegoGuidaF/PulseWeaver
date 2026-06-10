//go:build test

package queries_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries/filterx"
	"github.com/matryer/is"
)

func TestNewAccessLogQuery_Defaults(t *testing.T) {
	is := is.New(t)
	before := time.Now().UTC()
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{})
	is.NoErr(err)
	after := time.Now().UTC()

	// From defaults to approximately 24 hours ago.
	is.True(q.From.After(before.Add(-24*time.Hour - time.Second)))
	is.True(q.From.Before(after.Add(-24*time.Hour + time.Second)))

	// To defaults to approximately now.
	is.True(q.To.After(before.Add(-time.Second)))
	is.True(q.To.Before(after.Add(time.Second)))

	is.Equal(q.Limit, 50)
	is.Equal(q.Sort, "created_at")
	is.Equal(q.Order, "desc")
	is.True(q.Cursor == nil)
	is.Equal(len(q.Filters), 0)
}

func TestNewAccessLogQuery_LimitZeroDefaultsFifty(t *testing.T) {
	is := is.New(t)
	zero := 0
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Limit: &zero})
	is.NoErr(err)
	is.Equal(q.Limit, 50)
}

func TestNewAccessLogQuery_LimitNegativeDefaultsFifty(t *testing.T) {
	is := is.New(t)
	neg := -1
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Limit: &neg})
	is.NoErr(err)
	is.Equal(q.Limit, 50)
}

func TestNewAccessLogQuery_LimitCappedAt200(t *testing.T) {
	is := is.New(t)
	big := 9999
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Limit: &big})
	is.NoErr(err)
	is.Equal(q.Limit, 200)
}

func TestNewAccessLogQuery_ExplicitFromTo(t *testing.T) {
	is := is.New(t)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{From: &from, To: &to})
	is.NoErr(err)
	is.Equal(q.From, from)
	is.Equal(q.To, to)
}

func TestNewAccessLogQuery_ValueFiltersBuilt(t *testing.T) {
	is := is.New(t)
	codes := []string{"DE", "US"}
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{
		CountryCode: &codes,
	})
	is.NoErr(err)
	is.Equal(len(q.Filters), 1)
	is.Equal(q.Filters[0].Column, "country_code")
	is.Equal(q.Filters[0].Op, filterx.OpIn) // default operator
	is.Equal(len(q.Filters[0].Values), 2)
}

func TestNewAccessLogQuery_NullOperatorNeedsNoValues(t *testing.T) {
	is := is.New(t)
	op := httpapi.AccessLogFilterOperator(filterx.OpIsNull)
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{
		CountryCodeOp: &op,
	})
	is.NoErr(err)
	is.Equal(len(q.Filters), 1)
	is.Equal(q.Filters[0].Op, filterx.OpIsNull)
}

func TestNewAccessLogQuery_OperatorWithoutValuesIgnored(t *testing.T) {
	is := is.New(t)
	op := httpapi.AccessLogFilterOperator(filterx.OpContains)
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{
		TargetHostOp: &op,
	})
	is.NoErr(err)
	is.Equal(len(q.Filters), 0) // contains with no values is a no-op
}

func TestNewAccessLogQuery_RejectsDisallowedOperator(t *testing.T) {
	is := is.New(t)
	// client_ip does not allow is_null.
	op := httpapi.AccessLogFilterOperator(filterx.OpIsNull)
	ips := []string{"1.2.3.4"}
	_, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{
		ClientIp:   &ips,
		ClientIpOp: &op,
	})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestNewAccessLogQuery_RejectsBadCursor(t *testing.T) {
	is := is.New(t)
	bad := "not-a-real-cursor"
	_, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Cursor: &bad})
	is.True(errors.Is(err, filterx.ErrInvalidFilter))
}

func TestNewAccessLogQuery_SortAndOrder(t *testing.T) {
	is := is.New(t)
	sort := httpapi.GetAccessLogParamsSort("duration_us")
	order := httpapi.GetAccessLogParamsOrder("asc")
	q, err := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Sort: &sort, Order: &order})
	is.NoErr(err)
	is.Equal(q.Sort, "duration_us")
	is.Equal(q.Order, "asc")
}
