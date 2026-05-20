//go:build test

package scheduler_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/scheduler"
	"github.com/matryer/is"
)

func noopLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// RetentionJob

type fakeAccessLogPruner struct {
	deleted int64
	err     error
	calls   int
}

func (f *fakeAccessLogPruner) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	f.calls++
	return f.deleted, f.err
}

type fakeAddressEventPruner struct {
	deleted int64
	err     error
	calls   int
}

func (f *fakeAddressEventPruner) DeleteAddressEventsOlderThan(_ context.Context, _ time.Time) (int64, error) {
	f.calls++
	return f.deleted, f.err
}

var _ scheduler.AccessLogPruner = (*fakeAccessLogPruner)(nil)
var _ scheduler.AddressEventPruner = (*fakeAddressEventPruner)(nil)

func TestRetentionJob_ZeroRetentionDays_SkipsBothPruners(t *testing.T) {
	is := is.New(t)
	alp := &fakeAccessLogPruner{}
	aep := &fakeAddressEventPruner{}
	job := scheduler.NewRetentionJob(alp, aep, 0, noopLogger())

	err := job.Run(context.Background())

	is.NoErr(err)
	is.Equal(alp.calls, 0)
	is.Equal(aep.calls, 0)
}

func TestRetentionJob_CallsBothPrunersOnFirstRun(t *testing.T) {
	is := is.New(t)
	alp := &fakeAccessLogPruner{deleted: 5}
	aep := &fakeAddressEventPruner{deleted: 3}
	job := scheduler.NewRetentionJob(alp, aep, 30, noopLogger())

	err := job.Run(context.Background())

	is.NoErr(err)
	is.Equal(alp.calls, 1)
	is.Equal(aep.calls, 1)
}

func TestRetentionJob_DailyGuard_DoesNotRunTwiceInSameDay(t *testing.T) {
	is := is.New(t)
	alp := &fakeAccessLogPruner{}
	aep := &fakeAddressEventPruner{}
	job := scheduler.NewRetentionJob(alp, aep, 30, noopLogger())

	_ = job.Run(context.Background())
	err := job.Run(context.Background())

	is.NoErr(err)
	is.Equal(alp.calls, 1)
	is.Equal(aep.calls, 1)
}

func TestRetentionJob_AccessLogPrunerError_Propagates(t *testing.T) {
	is := is.New(t)
	pruneErr := errors.New("db error")
	alp := &fakeAccessLogPruner{err: pruneErr}
	aep := &fakeAddressEventPruner{}
	job := scheduler.NewRetentionJob(alp, aep, 30, noopLogger())

	err := job.Run(context.Background())

	is.True(errors.Is(err, pruneErr))
	is.Equal(aep.calls, 0)
}

func TestRetentionJob_AddressEventPrunerError_Propagates(t *testing.T) {
	is := is.New(t)
	pruneErr := errors.New("db error")
	alp := &fakeAccessLogPruner{}
	aep := &fakeAddressEventPruner{err: pruneErr}
	job := scheduler.NewRetentionJob(alp, aep, 30, noopLogger())

	err := job.Run(context.Background())

	is.True(errors.Is(err, pruneErr))
}

// RunSchedule

type countingJob struct{ runs atomic.Int32 }

func (j *countingJob) Run(_ context.Context) error {
	j.runs.Add(1)
	return nil
}

func TestService_RunSchedule_ContextCancellation_ExitsCleanly(t *testing.T) {
	is := is.New(t)
	svc := scheduler.NewService(noopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.RunSchedule(ctx, time.Hour) }()
	cancel()

	select {
	case err := <-done:
		is.NoErr(err)
	case <-time.After(1 * time.Second):
		t.Fatal("RunSchedule did not exit after context cancellation")
	}
}

func TestService_RunSchedule_TickFiresExecuteScheduledRules(t *testing.T) {
	is := is.New(t)
	job := &countingJob{}
	svc := scheduler.NewService(noopLogger())
	svc.AddJob(job)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = svc.RunSchedule(ctx, 10*time.Millisecond) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if job.runs.Load() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()

	is.True(job.runs.Load() > 0)
}
