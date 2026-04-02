package httpserver

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// LoginRateLimitMiddleware rate limits POST /api/v1/auth/login by client IP.
func LoginRateLimitMiddleware(requests int, window time.Duration) func(http.Handler) http.Handler {
	return ipRateLimitMiddleware(httpapi.LoginEndpoint, http.MethodPost, requests, window,
		"Too many login attempts. Try again later.")
}

// HeartbeatRateLimitMiddleware rate limits POST /api/v1/heartbeat by client IP.
func HeartbeatRateLimitMiddleware(requests int, window time.Duration) func(http.Handler) http.Handler {
	return ipRateLimitMiddleware(httpapi.HeartbeatEndpoint, http.MethodPost, requests, window,
		"Too many heartbeat requests. Try again later.")
}

// RegistrationRateLimitMiddleware rate limits POST /api/v1/register by client IP.
func RegistrationRateLimitMiddleware(requests int, window time.Duration) func(http.Handler) http.Handler {
	return ipRateLimitMiddleware(httpapi.RegisterEndpoint, http.MethodPost, requests, window,
		"Too many registration attempts. Try again later.")
}

// ipRateLimitMiddleware creates a middleware that rate limits a specific path+method by client IP.
// The key is read from the request context (set by the IP middleware) with a fallback to RemoteAddr.
// When the limit is exceeded, a JSON 429 response is returned with the given message.
func ipRateLimitMiddleware(path, method string, requests int, window time.Duration, msg string) func(http.Handler) http.Handler {
	clientIP := func(r *http.Request) string {
		if ip, ok := httpapi.ClientIPFromContext(r.Context()); ok && ip != "" {
			return ip
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}

	limiter := httprate.NewRateLimiter(requests, window,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(httpapi.ErrorResponse{Error: &msg})
		}),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == path && r.Method == method {
				if limiter.RespondOnLimit(w, r, clientIP(r)) {
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySizeMiddleware limits request body size to prevent large payloads from exhausting memory.
func MaxBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIPFromRequestMiddleware is middleware that extracts the client IP from r.RemoteAddr
// and sets it in the request context. It ignores any X-Forwarded-For headers.
func ClientIPFromRequestMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := extractIPFromRemoteAddr(r)
			r = setClientIPInContext(r, clientIP)
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIPFromRealIPMiddleware extracts client IP from X-Real-IP only when the
// direct peer address is within trustedProxy.
//
// If the peer is not trusted, X-Real-IP is ignored to prevent spoofing and a
// security warning is logged when that header is present.
func ClientIPFromRealIPMiddleware(trustedProxy netip.Addr, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := extractIPFromRemoteAddr(r)

			peerAddr, err := netip.ParseAddr(clientIP)
			if err != nil {
				r = setClientIPInContext(r, clientIP)
				next.ServeHTTP(w, r)
				return
			}

			rawRealIP := strings.TrimSpace(r.Header.Get(httpapi.XRealIP))
			if trustedProxy.Compare(peerAddr) != 0 {
				if rawRealIP != "" {
					if logger != nil {
						logger.WarnContext(r.Context(), "ignored X-Real-IP from untrusted peer",
							slog.String("peer_ip", peerAddr.String()),
							slog.String("header_ip", rawRealIP),
						)
					}
				}
				r = setClientIPInContext(r, peerAddr.String())
				next.ServeHTTP(w, r)
				return
			}

			if rawRealIP == "" {
				r = setClientIPInContext(r, peerAddr.String())
				next.ServeHTTP(w, r)
				return
			}

			realAddr, err := netip.ParseAddr(rawRealIP)
			if err != nil {
				if logger != nil {
					logger.WarnContext(r.Context(), "invalid X-Real-IP from trusted peer",
						slog.String("peer_ip", peerAddr.String()),
						slog.String("header_ip", rawRealIP),
					)
				}
				r = setClientIPInContext(r, peerAddr.String())
				next.ServeHTTP(w, r)
				return
			}

			r = setClientIPInContext(r, realAddr.String())
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLoggerMiddleware stores the request ID in context so the custom
// slog.Handler can stamp it on every log record produced during this request.
// The base logger is no longer stored in context; each handler uses its own
// struct-level logger.
func RequestLoggerMiddleware(_ *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			ctx := logging.WithRequestID(r.Context(), reqID)
			next.ServeHTTP(w, r.WithContext(ctx))
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
	ctx := httpapi.WithClientIP(r.Context(), ip)
	ctx = logging.WithClientIP(ctx, ip)
	return r.WithContext(ctx)
}
