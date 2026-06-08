//go:build test

package httpserver

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/database"
)

// TestResponseErrorHandler_Contention verifies the central response error handler
// degrades write contention to 503 + Retry-After (even when ErrContended is
// wrapped, as withinTx returns it) while leaving other errors as 500.
func TestResponseErrorHandler_Contention(t *testing.T) {
	handler := createResponseErrorHandler(slog.New(slog.DiscardHandler))

	t.Run("wrapped contention → 503 with Retry-After", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)

		handler(rec, req, fmt.Errorf("create session: %w", database.ErrContended))

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("expected a Retry-After header")
		}
	})

	t.Run("other error → 500", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)

		handler(rec, req, errors.New("something else"))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if rec.Header().Get("Retry-After") != "" {
			t.Fatal("did not expect a Retry-After header on a 500")
		}
	})
}
