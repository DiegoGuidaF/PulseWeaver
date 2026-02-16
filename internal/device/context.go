package device

import "context"

type contextKey string

const clientIPCtxKey contextKey = "client_ip"
const principalContextKey contextKey = "devicePrincipal"

// WithClientIP returns a new context with the client IP address stored in it.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPCtxKey, ip)
}

// ClientIPFromContext extracts the client IP address from the context.
func ClientIPFromContext(ctx context.Context) (string, bool) {
	ip, ok := ctx.Value(clientIPCtxKey).(string)
	return ip, ok && ip != ""
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return &principal, ok
}
