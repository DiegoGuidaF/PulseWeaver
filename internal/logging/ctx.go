package logging

import (
	"context"
)

// requestIDKeyType is an unexported type for the request ID context key.
type requestIDKeyType struct{}

var requestIDKey = requestIDKeyType{}

// WithRequestID stores a request or flow ID string in the context.
// The custom slog handler reads this value and stamps it on every log record.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromCtx retrieves the request/flow ID stored by WithRequestID.
// Returns the empty string and false if no ID is present.
func RequestIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}

// clientIPKeyType is an unexported type for the client IP context key.
type clientIPKeyType struct{}

var clientIPKey = clientIPKeyType{}

// WithClientIP stores the client IP string in the context for the slog handler to stamp automatically.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

// ClientIPFromCtx retrieves the client IP stored by WithClientIP.
// Returns empty string and false if not present.
func ClientIPFromCtx(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(clientIPKey).(string)
	return ip, ok
}
