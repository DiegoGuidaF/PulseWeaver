package logging

import (
	"context"
	"log/slog"
)

// Use unexported struct type for context key to avoid collisions across packages
type ctxKey struct{}

// ToCtx injects a logger into context
func ToCtx(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromCtx extracts logger from context, returns fallback logger if not found
// Always pass an explicit fallback logger - never use slog.Default() as it reintroduces implicit global dependency
func FromCtx(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// Enrich extracts the logger from context, adds attributes with With(),
// stores the enriched logger back in context, and returns both.
// This ensures progressive enrichment flows to all downstream layers.
func Enrich(ctx context.Context, attrs ...any) (context.Context, *slog.Logger) {
	logger := FromCtx(ctx).With(attrs...)
	return ToCtx(ctx, logger), logger
}
