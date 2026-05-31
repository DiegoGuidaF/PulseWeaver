package networkpolicies

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// HTTPHandler implements the network-policies write operations of httpapi.StrictServerInterface.
// Read operations (ListNetworkPolicies, GetNetworkPolicy) are handled by queries.HTTPHandler.
type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "networkpolicies")),
	}
}

func (h *HTTPHandler) CreateNetworkPolicy(
	ctx context.Context,
	request httpapi.CreateNetworkPolicyRequestObject,
) (httpapi.CreateNetworkPolicyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateNetworkPolicy")

	body := request.Body
	p, err := h.service.CreatePolicy(ctx, body.Name, body.Cidr, body.Description)
	if err != nil {
		return h.mapCreateError(ctx, err), nil
	}
	h.logger.InfoContext(ctx, "network policy created", slog.Int64("policy_id", p.ID.Int64()))
	return httpapi.CreateNetworkPolicy201JSONResponse(toNetworkPolicyResponse(p)), nil
}

func (h *HTTPHandler) UpdateNetworkPolicy(
	ctx context.Context,
	request httpapi.UpdateNetworkPolicyRequestObject,
) (httpapi.UpdateNetworkPolicyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UpdateNetworkPolicy")

	id := ids.NetworkPolicyID(request.PolicyId)
	body := request.Body

	fields := UpdateFields{
		Name:        body.Name,
		CIDR:        body.Cidr,
		Enabled:     body.Enabled,
		Description: body.Description,
	}

	_, err := h.service.UpdatePolicy(ctx, id, fields)
	if err != nil {
		return h.mapUpdateError(ctx, err), nil
	}
	h.logger.InfoContext(ctx, "network policy updated", slog.Int64("policy_id", id.Int64()))
	return httpapi.UpdateNetworkPolicy204Response{}, nil
}

func (h *HTTPHandler) DeleteNetworkPolicy(
	ctx context.Context,
	request httpapi.DeleteNetworkPolicyRequestObject,
) (httpapi.DeleteNetworkPolicyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteNetworkPolicy")

	id := ids.NetworkPolicyID(request.PolicyId)
	err := h.service.DeletePolicy(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return httpapi.DeleteNetworkPolicy404JSONResponse(errMsg("Network policy not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to delete network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.DeleteNetworkPolicy500JSONResponse(errMsg("Failed to delete network policy")), nil
	}
	h.logger.InfoContext(ctx, "network policy deleted", slog.Int64("policy_id", id.Int64()))
	return httpapi.DeleteNetworkPolicy204Response{}, nil
}

func (h *HTTPHandler) UpdateNetworkPolicyAccess(
	ctx context.Context,
	request httpapi.UpdateNetworkPolicyAccessRequestObject,
) (httpapi.UpdateNetworkPolicyAccessResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UpdateNetworkPolicyAccess")

	id := ids.NetworkPolicyID(request.PolicyId)
	body := request.Body

	groupIDs := make([]ids.HostGroupID, len(body.GroupIds))
	for i, groupID := range body.GroupIds {
		groupIDs[i] = ids.HostGroupID(groupID)
	}

	err := h.service.SetHostAccess(ctx, id, body.BypassHostCheck, groupIDs)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return httpapi.UpdateNetworkPolicyAccess404JSONResponse(errMsg("Network policy not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to update host access", slog.Any(logging.AttrKeyError, err))
		return httpapi.UpdateNetworkPolicyAccess500JSONResponse(errMsg("Failed to update host access")), nil
	}
	h.logger.InfoContext(ctx, "network policy access updated", slog.Int64("policy_id", id.Int64()))
	return httpapi.UpdateNetworkPolicyAccess204Response{}, nil
}

// ── mapping helpers ────────────────────────────────────────────────────────────

func toNetworkPolicyResponse(p NetworkPolicy) httpapi.NetworkPolicyDetail {
	return httpapi.NetworkPolicyDetail{
		Id:              p.ID.Int64(),
		Name:            p.Name,
		Cidr:            p.CIDR,
		Description:     p.Description,
		Enabled:         p.Enabled,
		BypassHostCheck: p.BypassHostCheck,
		CreatedAt:       httpapi.UTCTime(p.CreatedAt),
		UpdatedAt:       httpapi.UTCTime(p.UpdatedAt),
	}
}

func (h *HTTPHandler) mapCreateError(ctx context.Context, err error) httpapi.CreateNetworkPolicyResponseObject {
	switch {
	case errors.Is(err, ErrCIDRConflict):
		return httpapi.CreateNetworkPolicy409JSONResponse(errMsg("A policy with this CIDR already exists"))
	case errors.Is(err, ErrInvalidCIDR):
		return httpapi.CreateNetworkPolicy400JSONResponse(errMsg(err.Error()))
	default:
		h.logger.ErrorContext(ctx, "failed to create network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.CreateNetworkPolicy500JSONResponse(errMsg("Failed to create network policy"))
	}
}

func (h *HTTPHandler) mapUpdateError(ctx context.Context, err error) httpapi.UpdateNetworkPolicyResponseObject {
	switch {
	case errors.Is(err, ErrNotFound):
		return httpapi.UpdateNetworkPolicy404JSONResponse(errMsg("Network policy not found"))
	case errors.Is(err, ErrCIDRConflict):
		return httpapi.UpdateNetworkPolicy409JSONResponse(errMsg("A policy with this CIDR already exists"))
	case errors.Is(err, ErrInvalidCIDR):
		return httpapi.UpdateNetworkPolicy400JSONResponse(errMsg(err.Error()))
	default:
		h.logger.ErrorContext(ctx, "failed to update network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.UpdateNetworkPolicy500JSONResponse(errMsg("Failed to update network policy"))
	}
}

func errMsg(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
