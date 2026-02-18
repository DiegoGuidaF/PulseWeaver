package httpserver

import (
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"github.com/go-chi/httprate"
)

// LoginRateLimitMiddleware creates a middleware that rate limits only POST /api/v1/auth/login requests.
// Other endpoints are not affected. Uses a custom key function that reads from context.
func LoginRateLimitMiddleware(requests int, window time.Duration) func(http.Handler) http.Handler {
	// Custom key function that reads client IP from context
	keyFunc := func(r *http.Request) (string, error) {
		ip, ok := api.ClientIPFromContext(r.Context())
		if !ok || ip == "" {
			// Fallback to RemoteAddr if context doesn't have IP
			clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				clientIP = r.RemoteAddr
			}
			return clientIP, nil
		}
		return ip, nil
	}

	limiter := httprate.NewRateLimiter(requests, window,
		httprate.WithKeyFuncs(keyFunc),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			msg := "Too many login attempts. Try again later."
			_ = json.NewEncoder(w).Encode(api.ErrorResponse{Error: &msg})
		}),
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only rate limit login endpoint
			if r.URL.Path == api.LoginEndpoint && r.Method == http.MethodPost {
				// Extract key using the custom key function
				key, err := keyFunc(r)
				if err != nil {
					// If we can't extract IP, allow the request through
					next.ServeHTTP(w, r)
					return
				}
				// Check limit and respond with 429 if exceeded
				if limiter.RespondOnLimit(w, r, key) {
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize limits request body size to prevent large payloads from exhausting memory.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIpFromRequest is middleware that extracts the client IP from r.RemoteAddr
// and sets it in the request context. It ignores any X-Forwarded-For headers.
func ClientIpFromRequest() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := extractIPFromRemoteAddr(r)
			r = setClientIPInContext(r, clientIP)
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIPFromXFFHeader is middleware that extracts the client IP from X-Forwarded-For headers
// only when the direct connection is from a trusted proxy (the given IP address).
// Otherwise forwarded headers are ignored to prevent spoofing.
//
// Algorithm:
// 1. Collects all IPs from ALL X-Forwarded-For headers (prevents header injection attacks)
// 2. Selects the rightmost IP from XFF headers (more secure - prevents spoofing)
// 3. Stores client IP in context (does NOT modify r.RemoteAddr to avoid port issues)
//
// Note: This middleware assumes trustedProxy.IsValid() is true. Use ClientIpFromRequest()
// when trusted proxy is not configured.
func ClientIPFromXFFHeader(trustedProxy netip.Addr) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract direct peer IP (the proxy we're connected to)
			clientIP := extractIPFromRemoteAddr(r)

			peerAddr, err := netip.ParseAddr(clientIP)
			if err != nil {
				// Invalid peer IP, don't trust forwarded headers
				// Store original RemoteAddr IP in context as fallback
				r = setClientIPInContext(r, clientIP)
				next.ServeHTTP(w, r)
				return
			}

			// Check if direct peer equals the trusted proxy IP
			if peerAddr != trustedProxy {
				// Peer is not trusted, use original RemoteAddr IP
				r = setClientIPInContext(r, peerAddr.String())
				next.ServeHTTP(w, r)
				return
			}

			// Only trust X-Forwarded-For if peer is trusted
			// Collect all IPs from all X-Forwarded-For headers
			xffIPs := collectXFFIPs(r)
			if len(xffIPs) == 0 {
				// No XFF headers, use peer IP
				r = setClientIPInContext(r, peerAddr.String())
				next.ServeHTTP(w, r)
				return
			}

			// Use the rightmost IP from XFF headers (most secure - prevents spoofing)
			selectedIP := xffIPs[len(xffIPs)-1]

			// Store selected client IP in context
			r = setClientIPInContext(r, selectedIP.String())
			next.ServeHTTP(w, r)
		})
	}
}

// extractIPFromRemoteAddr extracts the IP address from r.RemoteAddr.
// Handles both "host:port" and plain address formats.
func extractIPFromRemoteAddr(r *http.Request) string {
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return clientIP
}

// setClientIPInContext sets the client IP in the request context.
func setClientIPInContext(r *http.Request, ip string) *http.Request {
	ctx := api.WithClientIP(r.Context(), ip)
	return r.WithContext(ctx)
}

// collectXFFIPs collects all IP addresses from all X-Forwarded-For headers.
// Handles multiple headers to prevent header injection attacks.
func collectXFFIPs(r *http.Request) []netip.Addr {
	var ips []netip.Addr
	// Use Values() to get all X-Forwarded-For headers (not just the first one)
	for _, headerValue := range r.Header.Values(api.XForwardedFor) {
		// Split comma-separated values in each header
		parts := strings.Split(headerValue, ",")
		for _, part := range parts {
			candidate := strings.TrimSpace(part)
			if candidate == "" {
				continue
			}
			// Try to parse as IP address
			if addr, err := netip.ParseAddr(candidate); err == nil {
				ips = append(ips, addr)
			}
			// Invalid IP entries are ignored
		}
	}
	return ips
}
