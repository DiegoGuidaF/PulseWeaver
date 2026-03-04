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

// NewContextHandler wraps inner with a handler that reads RequestIDFromCtx
// and appends a "request_id" attribute to every record that has one.
func NewContextHandler(inner slog.Handler) slog.Handler {
	return &contextHandler{inner: inner}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := RequestIDFromCtx(ctx); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	if ip, ok := ClientIPFromCtx(ctx); ok {
		r.AddAttrs(slog.String("client_ip", ip))
	}
	return h.inner.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{inner: h.inner.WithGroup(name)}
}
