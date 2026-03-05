package authz

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
)

// HTTPHandler is the HTTP handler for forward-auth IP verification.
type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{service: service, logger: logger.With(slog.String(logging.AttrKeyComponent, "authz"))}
}

// HandleForwardAuthIP serves GET /api/authz/verify-ip.
// Returns 200 if the IP in X-Real-IP is enabled, 403 otherwise.
// All failure paths return 403 (fail-closed) — never 401, to avoid leaking information.
func (h *HTTPHandler) HandleForwardAuthIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := h.logger.With(slog.String(logging.AttrKeyOperation, "HandleForwardAuthIP"))

	authHeader := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || token == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "authz: missing client IP in request context")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := h.service.VerifyAccess(ctx, token, clientIP); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
}
