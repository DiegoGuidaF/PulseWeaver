package device

import (
	"net/http"
)

// ClientIPContextMiddleware Retrieves the client IP from the request and injects it into request context.
func ClientIPContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP from request (RealIP middleware sets RemoteAddr)
			clientIP := r.RemoteAddr
			ctx := WithClientIP(r.Context(), clientIP)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
