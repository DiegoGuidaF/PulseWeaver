package policy

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// HTTPHandler is the HTTP handler for forward-auth IP verification.
type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{service: service, logger: logger.With(slog.String(logging.AttrKeyComponent, "policy"))}
}

// HandleForwardAuthIP serves GET /api/policy-engine/verify-ip.
// Returns 200 if the IP in X-Real-IP is enabled, 403 otherwise.
// All failure paths return 403 (fail-closed) — never 401, to avoid leaking information.
func (h *HTTPHandler) HandleForwardAuthIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = logging.WithOperation(ctx, "HandleForwardAuthIP")
	h.logger.DebugContext(ctx, "Request received with headers", slog.Any(logging.AttrKeyHeaders, r.Header))

	authHeader := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || token == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		h.logger.WarnContext(ctx, "policy: missing client IP in request context")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	req := NewVerifyRequest(token, clientIP, r)

	if err := h.service.VerifyAccess(ctx, req); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
}
