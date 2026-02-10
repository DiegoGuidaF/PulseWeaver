package auth

import "context"

type Principal struct {
	UserID   string
	DeviceID *string // nil for browser session
	TokenID  *string // device token id, for audit
}

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
