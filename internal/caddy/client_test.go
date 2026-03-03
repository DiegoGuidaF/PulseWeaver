package caddy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPNotifier_SendsPostOnSignal(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewReloaderClient(server.URL, "")
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- notifier.Run(ctx)
	}()

	notifier.NotifyChange(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&callCount) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error from Run, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}
}

func TestHTTPNotifier_IncludesTokenHeader(t *testing.T) {
	const token = "secret-authToken"

	headerCh := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerCh <- r.Header.Get("X-Reloader-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewReloaderClient(server.URL, token)
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- notifier.Run(ctx)
	}()

	notifier.NotifyChange(context.Background())

	select {
	case header := <-headerCh:
		if header != token {
			t.Fatalf("expected authToken header %q, got %q", token, header)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive request with authToken header")
	}

	cancel()
	<-done
}

func TestHTTPNotifier_RetriesOnServerError(t *testing.T) {
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&callCount, 1)
		if current == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewReloaderClient(server.URL, "")
	if notifier == nil {
		t.Fatal("expected non-nil notifier")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- notifier.Run(ctx)
	}()

	notifier.NotifyChange(context.Background())

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&callCount) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if atomic.LoadInt32(&callCount) < 2 {
		t.Fatalf("expected at least 2 calls due to retry, got %d", callCount)
	}

	cancel()
	<-done
}
