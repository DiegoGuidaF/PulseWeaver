package caddy

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_SendsPostOnSignal(t *testing.T) {
	callCh := make(chan struct{}, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		select {
		case callCh <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewReloaderClient(server.URL, "", slog.Default())
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
	case <-callCh:
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive reload request")
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

func TestClient_IncludesTokenHeader(t *testing.T) {
	const token = "secret-authToken"

	headerCh := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerCh <- r.Header.Get("X-Reloader-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewReloaderClient(server.URL, token, slog.Default())
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

func TestClient_DisabledWhenNoEndpoint(t *testing.T) {
	notifier := NewReloaderClient("", "", slog.Default())

	// NotifyChange must be a no-op: no signal should be enqueued
	notifier.NotifyChange(context.Background())
	select {
	case <-notifier.signals:
		t.Fatal("expected no signal enqueued when disabled")
	default:
	}

	// Run must return nil immediately without blocking
	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- notifier.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil error from Run when disabled, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not return immediately when disabled")
	}
}

func TestClient_RetriesOnServerError(t *testing.T) {
	callCh := make(chan struct{}, 10)
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		select {
		case callCh <- struct{}{}:
		default:
		}

		if callCount == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewReloaderClient(server.URL, "", slog.Default())
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

	// Expect at least two calls due to retry behavior.
	select {
	case <-callCh:
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive initial request")
	}

	select {
	case <-callCh:
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive retry request")
	}

	cancel()
	<-done
}
