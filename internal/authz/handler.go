package authz

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

// Handler is the HTTP handler for forward-auth IP verification.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger.With(slog.String(logging.AttrKeyComponent, "authz"))}
}

// HandleForwardAuthIP serves GET /api/authz/verify-ip.
// Returns 200 if the IP in X-Real-IP is enabled, 403 otherwise.
// All failure paths return 403 (fail-closed) — never 401, to avoid leaking information.
func (h *Handler) HandleForwardAuthIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := h.logger.With(slog.String(logging.AttrKeyOperation, "HandleForwardAuthIP"))

	// 1. Secret must be configured
	secret := h.service.Secret()
	if secret == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 2. Validate Bearer token
	authHeader := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || token == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	// TODO: Check length issues and security
	if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
		logger.WarnContext(ctx, "authz: invalid bearer token")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 3. Read client IP from request context (set by global middleware)
	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "authz: missing client IP in request context")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// 4. Cache lookup
	if h.service.ContainsIP(clientIP) {
		w.WriteHeader(http.StatusOK)
		return
	}

	logger.DebugContext(ctx, "authz: IP not in enabled set",
		slog.String(AttrKeyRequestIP, clientIP),
	)
	w.WriteHeader(http.StatusForbidden)
}
