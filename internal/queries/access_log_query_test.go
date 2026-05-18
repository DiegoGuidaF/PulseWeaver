//go:build test

package queries_test

import (
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/queries"
	"github.com/matryer/is"
)

func TestNewAccessLogQuery_Defaults(t *testing.T) {
	is := is.New(t)
	before := time.Now().UTC()
	q := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{})
	after := time.Now().UTC()

	// From defaults to approximately 24 hours ago.
	is.True(q.From.After(before.Add(-24*time.Hour - time.Second)))
	is.True(q.From.Before(after.Add(-24*time.Hour + time.Second)))

	// To defaults to approximately now.
	is.True(q.To.After(before.Add(-time.Second)))
	is.True(q.To.Before(after.Add(time.Second)))

	is.Equal(q.Limit, 50)
}

func TestNewAccessLogQuery_LimitZeroDefaultsFifty(t *testing.T) {
	is := is.New(t)
	zero := 0
	q := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Limit: &zero})
	is.Equal(q.Limit, 50)
}

func TestNewAccessLogQuery_LimitNegativeDefaultsFifty(t *testing.T) {
	is := is.New(t)
	neg := -1
	q := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Limit: &neg})
	is.Equal(q.Limit, 50)
}

func TestNewAccessLogQuery_LimitCappedAt200(t *testing.T) {
	is := is.New(t)
	big := 9999
	q := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{Limit: &big})
	is.Equal(q.Limit, 200)
}

func TestNewAccessLogQuery_ExplicitFromTo(t *testing.T) {
	is := is.New(t)
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	q := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{From: &from, To: &to})
	is.Equal(q.From, from)
	is.Equal(q.To, to)
}

func TestNewAccessLogQuery_FiltersPassedThrough(t *testing.T) {
	is := is.New(t)
	outcome := true
	ip := "1.2.3.4"
	host := "example.com"
	reason := "no_device_match"
	devID := httpapi.ID(42)
	limit := 10

	q := queries.NewAccessLogQuery(httpapi.GetAccessLogParams{
		Outcome:    &outcome,
		Ip:         &ip,
		Host:       &host,
		DenyReason: &reason,
		DeviceId:   &devID,
		Limit:      &limit,
	})

	is.True(q.Outcome != nil)
	is.Equal(*q.Outcome, true)
	is.True(q.ClientIP != nil)
	is.Equal(*q.ClientIP, "1.2.3.4")
	is.True(q.TargetHost != nil)
	is.Equal(*q.TargetHost, "example.com")
	is.True(q.DenyReason != nil)
	is.Equal(*q.DenyReason, "no_device_match")
	is.True(q.DeviceID != nil)
	is.Equal(*q.DeviceID, ids.DeviceID(42))
	is.Equal(q.Limit, 10)
}
