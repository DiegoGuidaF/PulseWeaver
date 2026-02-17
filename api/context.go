package api

import "context"

type ContextKey string

// ClientIPContextKey the standard context key for storing client IP in infrastructure layer
const clientIPContextKey ContextKey = "client_ip"

// WithClientIP returns a new context with the client IP address stored in it.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPContextKey, ip)
}

// ClientIPFromContext extracts the client IP address from the context.
func ClientIPFromContext(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(clientIPContextKey).(string)
	return ip, ok && ip != ""
}
