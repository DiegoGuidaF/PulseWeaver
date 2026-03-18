//go:build test

package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/matryer/is"
)

// ParseLevel

func TestParseLevel_Debug(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel("debug"), slog.LevelDebug)
}

func TestParseLevel_Info(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel("info"), slog.LevelInfo)
}

func TestParseLevel_Warn(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel("warn"), slog.LevelWarn)
}

func TestParseLevel_Error(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel("error"), slog.LevelError)
}

func TestParseLevel_EmptyString_DefaultsToInfo(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel(""), slog.LevelInfo)
}

func TestParseLevel_InvalidString_DefaultsToInfo(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel("verbose"), slog.LevelInfo)
}

func TestParseLevel_CaseInsensitive(t *testing.T) {
	is := is.New(t)
	is.Equal(logging.ParseLevel("DEBUG"), slog.LevelDebug)
	is.Equal(logging.ParseLevel("WARN"), slog.LevelWarn)
}

// contextHandler — verified via JSON output of a real logger

func logLine(ctx context.Context) string {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(logging.NewContextHandler(inner))
	logger.InfoContext(ctx, "test")
	return buf.String()
}

func TestContextHandler_StampsRequestID(t *testing.T) {
	is := is.New(t)
	ctx := logging.WithRequestID(context.Background(), "req-abc")
	line := logLine(ctx)
	is.True(strings.Contains(line, `"request_id":"req-abc"`))
}

func TestContextHandler_StampsClientIP(t *testing.T) {
	is := is.New(t)
	ctx := logging.WithClientIP(context.Background(), "1.2.3.4")
	line := logLine(ctx)
	is.True(strings.Contains(line, `"client_ip":"1.2.3.4"`))
}

func TestContextHandler_StampsOperation(t *testing.T) {
	is := is.New(t)
	ctx := logging.WithOperation(context.Background(), "HandleFoo")
	line := logLine(ctx)
	is.True(strings.Contains(line, `"operation":"HandleFoo"`))
}

func TestContextHandler_EmptyContext_NoExtraAttrs(t *testing.T) {
	is := is.New(t)
	line := logLine(context.Background())
	is.True(!strings.Contains(line, "request_id"))
	is.True(!strings.Contains(line, "client_ip"))
	is.True(!strings.Contains(line, "operation"))
}

// NewShortID

func TestNewShortID_Returns6CharHex(t *testing.T) {
	is := is.New(t)
	id := logging.NewShortID()
	is.Equal(len(id), 6)
	for _, c := range id {
		is.True((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
	}
}

func TestNewShortID_SuccessiveCallsDiffer(t *testing.T) {
	is := is.New(t)
	ids := make(map[string]struct{})
	for range 20 {
		ids[logging.NewShortID()] = struct{}{}
	}
	// With 3 random bytes (16M possibilities) collision in 20 draws is astronomically unlikely.
	is.True(len(ids) > 1)
}

// Context round-trips

func TestWithRequestID_RoundTrip(t *testing.T) {
	is := is.New(t)
	ctx := logging.WithRequestID(context.Background(), "my-id")
	id, ok := logging.RequestIDFromCtx(ctx)
	is.True(ok)
	is.Equal(id, "my-id")
}

func TestWithClientIP_RoundTrip(t *testing.T) {
	is := is.New(t)
	ctx := logging.WithClientIP(context.Background(), "10.0.0.1")
	ip, ok := logging.ClientIPFromCtx(ctx)
	is.True(ok)
	is.Equal(ip, "10.0.0.1")
}

func TestWithOperation_RoundTrip(t *testing.T) {
	is := is.New(t)
	ctx := logging.WithOperation(context.Background(), "SomeOp")
	op, ok := logging.OperationFromCtx(ctx)
	is.True(ok)
	is.Equal(op, "SomeOp")
}

func TestRequestIDFromCtx_MissingKey_ReturnsFalse(t *testing.T) {
	is := is.New(t)
	_, ok := logging.RequestIDFromCtx(context.Background())
	is.True(!ok)
}
