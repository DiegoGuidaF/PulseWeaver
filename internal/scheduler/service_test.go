//go:build test

package scheduler_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/device"
	"github.com/DiegoGuidaF/PulseWeaver/internal/scheduler"
	"github.com/matryer/is"
)

// fakeExpiredFinder returns a preset list of expired address IDs.
type fakeExpiredFinder struct {
	ids []device.AddressID
	err error
}

func (f *fakeExpiredFinder) GetExpiredAddressIDs(_ context.Context) ([]device.AddressID, error) {
	return f.ids, f.err
}

// fakeAddressDisabler records the last DisableAddresses call.
type fakeAddressDisabler struct {
	calledWith []device.AddressID
	source     device.EventSource
	err        error
	calls      int
}

func (f *fakeAddressDisabler) DisableAddresses(_ context.Context, ids []device.AddressID, source device.EventSource) error {
	f.calls++
	f.calledWith = ids
	f.source = source
	return f.err
}

var _ scheduler.ExpiredAddressFinder = (*fakeExpiredFinder)(nil)
var _ scheduler.AddressDisabler = (*fakeAddressDisabler)(nil)

func noopLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

// NewService

func TestNewService_NilFinder_ReturnsError(t *testing.T) {
	is := is.New(t)
	_, err := scheduler.NewService(nil, &fakeAddressDisabler{}, nil, noopLogger())
	is.True(err != nil)
}

func TestNewService_NilDisabler_ReturnsError(t *testing.T) {
	is := is.New(t)
	_, err := scheduler.NewService(&fakeExpiredFinder{}, nil, nil, noopLogger())
	is.True(err != nil)
}

// ExecuteScheduledRules

func TestService_ExecuteScheduledRules_NoExpiredAddresses_SkipsDisabler(t *testing.T) {
	is := is.New(t)
	finder := &fakeExpiredFinder{ids: []device.AddressID{}}
	disabler := &fakeAddressDisabler{}
	svc, err := scheduler.NewService(finder, disabler, nil, noopLogger())
	is.NoErr(err)

	err = svc.ExecuteScheduledRules(context.Background())

	is.NoErr(err)
	is.Equal(disabler.calls, 0)
}

func TestService_ExecuteScheduledRules_ExpiredAddressesFound_DisablesAll(t *testing.T) {
	is := is.New(t)
	finder := &fakeExpiredFinder{ids: []device.AddressID{device.AddressID(1), device.AddressID(2)}}
	disabler := &fakeAddressDisabler{}
	svc, err := scheduler.NewService(finder, disabler, nil, noopLogger())
	is.NoErr(err)

	err = svc.ExecuteScheduledRules(context.Background())

	is.NoErr(err)
	is.Equal(disabler.calls, 1)
	is.Equal(len(disabler.calledWith), 2)
	is.Equal(disabler.calledWith[0], device.AddressID(1))
	is.Equal(disabler.calledWith[1], device.AddressID(2))
	is.Equal(disabler.source, device.EventSourceExpiry)
}

func TestService_ExecuteScheduledRules_FinderError_Propagates(t *testing.T) {
	is := is.New(t)
	finderErr := errors.New("db error")
	finder := &fakeExpiredFinder{err: finderErr}
	disabler := &fakeAddressDisabler{}
	svc, err := scheduler.NewService(finder, disabler, nil, noopLogger())
	is.NoErr(err)

	err = svc.ExecuteScheduledRules(context.Background())

	is.True(errors.Is(err, finderErr))
	is.Equal(disabler.calls, 0)
}

func TestService_ExecuteScheduledRules_DisablerError_Propagates(t *testing.T) {
	is := is.New(t)
	disablerErr := errors.New("disable error")
	finder := &fakeExpiredFinder{ids: []device.AddressID{device.AddressID(1)}}
	disabler := &fakeAddressDisabler{err: disablerErr}
	svc, err := scheduler.NewService(finder, disabler, nil, noopLogger())
	is.NoErr(err)

	err = svc.ExecuteScheduledRules(context.Background())

	is.True(errors.Is(err, disablerErr))
}

// RunSchedule

func TestService_RunSchedule_ContextCancellation_ExitsCleanly(t *testing.T) {
	is := is.New(t)
	finder := &fakeExpiredFinder{ids: []device.AddressID{}}
	disabler := &fakeAddressDisabler{}
	svc, err := scheduler.NewService(finder, disabler, nil, noopLogger())
	is.NoErr(err)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- svc.RunSchedule(ctx, time.Hour) // long interval: tick never fires in test
	}()

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
	finder := &fakeExpiredFinder{ids: []device.AddressID{device.AddressID(42)}}
	disabler := &fakeAddressDisabler{}
	svc, err := scheduler.NewService(finder, disabler, nil, noopLogger())
	is.NoErr(err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() { _ = svc.RunSchedule(ctx, 10*time.Millisecond) }()

	// Wait for at least one tick to fire and call the disabler.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if disabler.calls > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()

	is.True(disabler.calls > 0)
	is.Equal(disabler.source, device.EventSourceExpiry)
}
