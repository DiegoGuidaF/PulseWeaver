//go:build test

package audit

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/policy"
	"github.com/matryer/is"
)

// fakeRepo is an in-memory fake that implements the repository interface.
type fakeRepo struct {
	mu     sync.Mutex
	events []policy.DecisionEvent
	err    error
}

func (f *fakeRepo) BatchInsert(_ context.Context, events []policy.DecisionEvent) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeRepo) storedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func noopSinkLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func makeEvent(ip string, outcome bool) policy.DecisionEvent {
	return policy.DecisionEvent{
		ClientIP:  ip,
		Outcome:   outcome,
		CreatedAt: time.Now().UTC(),
		Headers:   map[string][]string{},
	}
}

func TestSink_Submit_EventsBufferedAndFlushed(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	sink := NewSink(repo, noopSinkLogger())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = sink.Run(ctx)
	}()

	sink.OnDecision(context.Background(), makeEvent("1.2.3.4", true))
	sink.OnDecision(context.Background(), makeEvent("5.6.7.8", false))

	// Cancel context to trigger graceful drain and final flush.
	cancel()
	<-done

	is.Equal(repo.storedCount(), 2)
}

func TestSink_Submit_NonBlockingWhenBufferFull(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	sink := NewSink(repo, noopSinkLogger())

	// Fill the channel beyond its capacity (500) without starting Run.
	// All submissions must return immediately without blocking.
	submitted := make(chan struct{})
	go func() {
		defer close(submitted)
		for i := range 600 {
			sink.OnDecision(context.Background(), makeEvent("1.2.3.4", i%2 == 0))
		}
	}()

	select {
	case <-submitted:
		// Good: all 600 calls returned without blocking.
	case <-time.After(2 * time.Second):
		t.Fatal("Submit blocked when buffer was full")
	}

	// The channel holds at most 500 events; the rest were dropped.
	is.True(len(sink.ch) <= 500)
}

func TestSink_GracefulDrain_OnContextCancellation(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	sink := NewSink(repo, noopSinkLogger())

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = sink.Run(ctx)
	}()

	// Submit several events and then immediately cancel so they get drained.
	const numEvents = 10
	for i := range numEvents {
		sink.OnDecision(context.Background(), makeEvent("1.2.3.4", i%2 == 0))
	}
	cancel()
	<-done

	// All events submitted before cancellation must have been flushed.
	is.Equal(repo.storedCount(), numEvents)
}

func TestSink_BatchFill_FlushesEarly(t *testing.T) {
	is := is.New(t)
	repo := &fakeRepo{}
	sink := NewSink(repo, noopSinkLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = sink.Run(ctx)
	}()

	// Submit more than one full batch (50 events) to trigger an early flush.
	for i := range 55 {
		sink.OnDecision(context.Background(), makeEvent("1.2.3.4", i%2 == 0))
	}

	// Wait for the batch flush to happen.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if repo.storedCount() >= 50 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	<-done

	is.Equal(repo.storedCount(), 55)
}
