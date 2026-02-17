package device

import "context"

type contextKey string

const clientIPCtxKey contextKey = "client_ip"
const principalContextKey contextKey = "devicePrincipal"

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func PrincipalFromContext(ctx context.Context) (*Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return &principal, ok
}
