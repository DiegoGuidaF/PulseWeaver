package policy

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"
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
// This handler is not managed via openapi spec nor its related validations. It doesn't need authentication (API_KEY|cookies).
// Returns 200 if the IP in X-Real-IP is enabled, 403 otherwise.
// All failure paths return 403 (fail-closed) — never 401, to avoid leaking information.
func (h *HTTPHandler) HandleForwardAuthIP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = logging.WithOperation(ctx, "HandleForwardAuthIP")
	h.logger.DebugContext(ctx, "Verify request received")

	// Reject QUIC 0-RTT early data: the remote IP is unavailable before the TLS
	// handshake completes, so we cannot reliably identify the client. The client
	// should retry over a fully established connection (RFC 8470).
	if r.Header.Get("Early-Data") == "1" {
		h.logger.WarnContext(ctx, "rejected 0-RTT early data request")
		w.WriteHeader(http.StatusTooEarly)
		return
	}

	authHeader := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || token == "" {
		h.logger.DebugContext(ctx, "invalid authorization header")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	clientIP, ok := httpapi.ClientIPFromContext(ctx)
	if !ok {
		h.logger.WarnContext(ctx, "failed to get client IP from context")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	addr, err := netip.ParseAddr(clientIP)
	if err != nil {
		h.logger.WarnContext(ctx, "invalid client IP in context", slog.String(logging.AttrKeyClientIP, clientIP))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := h.service.VerifyAccess(ctx, new(NewVerifyRequest(token, addr.Unmap(), r))); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SimulatePolicyAccess Allows simulating a request for a given host and IP to see if it would be authorized (200) or not (403)
func (h *HTTPHandler) SimulatePolicyAccess(
	ctx context.Context,
	request httpapi.SimulatePolicyAccessRequestObject,
) (httpapi.SimulatePolicyAccessResponseObject, error) {
	ctx = logging.WithOperation(ctx, "SimulatePolicyAccess")

	ip := request.Params.Ip
	host := request.Params.Host

	// Parse the operator-supplied IP into a canonical address. An unparseable
	// value yields the zero Addr, which Decide treats as a fail-closed deny.
	addr, _ := netip.ParseAddr(ip)
	addr = addr.Unmap()
	if addr.IsValid() {
		ip = addr.String() // echo the canonical form that was actually evaluated
	}

	result := h.service.Decide(ctx, addr, host)
	denyReason := toAPIDenyReason(result.DenyReason)

	var matchSource *httpapi.PolicyMatchSource
	if result.Allowed {
		matchSource = new(httpapi.PolicyMatchSource(result.MatchSource))
	}

	var networkPolicyIDInt *int64
	if result.NetworkPolicyID != nil {
		networkPolicyIDInt = new(result.NetworkPolicyID.Int64())
	}

	return httpapi.SimulatePolicyAccess200JSONResponse(httpapi.PolicySimulateResult{
		Ip:                ip,
		Host:              host,
		Allowed:           result.Allowed,
		DenyReason:        denyReason,
		MatchSource:       matchSource,
		NetworkPolicyId:   networkPolicyIDInt,
		NetworkPolicyName: result.NetworkPolicyName,
	}), nil
}

func toAPIDenyReason(reason *DenyReason) *httpapi.PolicyDenyReason {
	if reason == nil {
		return nil
	}
	return new(httpapi.PolicyDenyReason(*reason))
}
