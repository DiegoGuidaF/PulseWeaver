package logging

import (
	"context"
	"log/slog"
)

// contextHandler is a slog.Handler that automatically stamps the request_id
// (or flow ID) stored in context onto every log record before delegating to
// the inner handler.
type contextHandler struct {
	inner slog.Handler
}

// NewContextHandler wraps inner with a handler that reads request-scoped values
// from context (request_id, client_ip, component, operation) and stamps them on
// every log record when present.
func NewContextHandler(inner slog.Handler) slog.Handler {
	return &contextHandler{inner: inner}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := RequestIDFromCtx(ctx); ok {
		r.AddAttrs(slog.String(AttrKeyRequestID, id))
	}
	if ip, ok := ClientIPFromCtx(ctx); ok {
		r.AddAttrs(slog.String(AttrKeyClientIP, ip))
	}
	if operation, ok := OperationFromCtx(ctx); ok {
		r.AddAttrs(slog.String(AttrKeyOperation, operation))
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}
