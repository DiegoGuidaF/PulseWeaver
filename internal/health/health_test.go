//go:build test

package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/health"
	"github.com/matryer/is"
)

func TestHandler_Returns200WithStatusOK(t *testing.T) {
	is := is.New(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	health.Handler(w, req)

	is.Equal(w.Code, http.StatusOK)
	is.Equal(w.Header().Get("Content-Type"), "application/json")

	var resp health.Response
	err := json.NewDecoder(w.Body).Decode(&resp)
	is.NoErr(err)
	is.Equal(resp.Status, "ok")
}

func TestHandler_TimestampIsRFC3339AndNonEmpty(t *testing.T) {
	is := is.New(t)

	before := time.Now().UTC().Truncate(time.Second)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	health.Handler(w, req)

	var resp health.Response
	err := json.NewDecoder(w.Body).Decode(&resp)
	is.NoErr(err)
	is.True(resp.Timestamp != "")

	ts, err := time.Parse(time.RFC3339, resp.Timestamp)
	is.NoErr(err)
	is.True(!ts.Before(before))
}
