package device

import "context"

type contextKey string

const clientIPCtxKey contextKey = "client_ip"

// WithClientIP returns a new context with the client IP address stored in it.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPCtxKey, ip)
}

// ClientIPFromContext extracts the client IP address from the context.
func ClientIPFromContext(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(clientIPCtxKey).(string)
	return ip, ok && ip != ""
}
