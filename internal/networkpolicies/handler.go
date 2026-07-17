package networkpolicies

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// HTTPHandler implements all of the network-policies operations of httpapi.StrictServerInterface,
// both writes (via service) and the read views (via repo).
type HTTPHandler struct {
	service *Service
	repo    *Repository
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, repo *Repository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		repo:    repo,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "networkpolicies")),
	}
}

func (h *HTTPHandler) ListNetworkPolicies(
	ctx context.Context,
	_ httpapi.ListNetworkPoliciesRequestObject,
) (httpapi.ListNetworkPoliciesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListNetworkPolicies")

	summaries, err := h.repo.GetNetworkPolicySummaries(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list network policies", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListNetworkPolicies500JSONResponse(errMsg("Failed to list network policies")), nil
	}

	resp := make(httpapi.ListNetworkPolicies200JSONResponse, len(summaries))
	for i, s := range summaries {
		resp[i] = toNetworkPolicySummaryResponse(s)
	}
	return resp, nil
}

func (h *HTTPHandler) GetNetworkPolicy(
	ctx context.Context,
	request httpapi.GetNetworkPolicyRequestObject,
) (httpapi.GetNetworkPolicyResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetNetworkPolicy")

	id := ids.NetworkPolicyID(request.PolicyId)
	detail, err := h.repo.GetNetworkPolicyDetail(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return httpapi.GetNetworkPolicy404JSONResponse(errMsg("Network policy not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to get network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetNetworkPolicy500JSONResponse(errMsg("Failed to get network policy")), nil
	}
	return httpapi.GetNetworkPolicy200JSONResponse(toNetworkPolicyDetailResponse(*detail)), nil
}

func toNetworkPolicySummaryResponse(s NetworkPolicySummaryView) httpapi.NetworkPolicyListItem {
	groups := make([]httpapi.GroupSummary, len(s.Groups))
	for i, g := range s.Groups {
		groups[i] = httpapi.GroupSummary{Id: g.ID.Int64(), Name: g.Name, Color: g.Color, Icon: g.Icon}
	}
	return httpapi.NetworkPolicyListItem{
		Id:              s.ID.Int64(),
		Name:            s.Name,
		Cidr:            s.CIDR,
		Enabled:         s.Enabled,
		BypassHostCheck: s.BypassHostCheck,
		HostCount:       s.EffectiveHostCount,
		CreatedAt:       httpapi.UTCTime(s.CreatedAt),
		Groups:          groups,
	}
}

func toNetworkPolicyDetailResponse(d NetworkPolicyDetailView) httpapi.NetworkPolicyDetail {
	groups := make([]httpapi.SubjectGroupDetail, len(d.HostGroups))
	for i, g := range d.HostGroups {
		hosts := make([]httpapi.HostSummary, len(g.Hosts))
		for j, h := range g.Hosts {
			hosts[j] = httpapi.HostSummary{
				Id:   h.ID,
				Fqdn: h.FQDN,
			}
		}
		var color string
		if g.Color != nil {
			color = *g.Color
		}
		var icon string
		if g.Icon != nil {
			icon = *g.Icon
		}
		groups[i] = httpapi.SubjectGroupDetail{
			Id:      g.ID,
			Name:    g.Name,
			Color:   color,
			Icon:    icon,
			Hosts:   hosts,
			Granted: g.Assigned,
		}
	}

	return httpapi.NetworkPolicyDetail{
		Id:              d.ID.Int64(),
		Name:            d.Name,
		Cidr:            d.CIDR,
		Description:     d.Description,
		Enabled:         d.Enabled,
		BypassHostCheck: d.BypassHostCheck,
		Groups:          groups,
		UpdatedAt:       httpapi.UTCTime(d.UpdatedAt),
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
	h.logger.InfoContext(ctx, "network policy created", policyLogAttrs(p)...)
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

	updated, err := h.service.UpdatePolicy(ctx, id, fields)
	if err != nil {
		return h.mapUpdateError(ctx, err), nil
	}
	h.logger.InfoContext(ctx, "network policy updated", policyLogAttrs(updated)...)
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

// policyLogAttrs builds the audit-log attributes for a policy mutation, flagging
// large-but-allowed CIDRs with broad_cidr=true so they stay distinguishable.
func policyLogAttrs(p NetworkPolicy) []any {
	attrs := []any{slog.Int64("policy_id", p.ID.Int64())}
	if broadCIDR(p.CIDR) {
		attrs = append(attrs, slog.Bool("broad_cidr", true))
	}
	return attrs
}

func toNetworkPolicyResponse(p NetworkPolicy) httpapi.NetworkPolicyDetail {
	return httpapi.NetworkPolicyDetail{
		Id:              p.ID.Int64(),
		Name:            p.Name,
		Cidr:            p.CIDR,
		Description:     p.Description,
		Enabled:         p.Enabled,
		BypassHostCheck: p.BypassHostCheck,
		// A freshly created policy has no group assignments; the detail view
		// refetch populates the authoritative all-groups list. Stay non-nil so the
		// response serializes "groups": [] rather than null (a required array field).
		Groups:    []httpapi.SubjectGroupDetail{},
		UpdatedAt: httpapi.UTCTime(p.UpdatedAt),
	}
}

func (h *HTTPHandler) mapCreateError(ctx context.Context, err error) httpapi.CreateNetworkPolicyResponseObject {
	switch {
	case errors.Is(err, ErrCIDRConflict):
		return httpapi.CreateNetworkPolicy409JSONResponse(errMsg("A policy with this CIDR already exists"))
	case errors.Is(err, ErrInvalidCIDR), errors.Is(err, ErrCIDRTooBroad):
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
	case errors.Is(err, ErrInvalidCIDR), errors.Is(err, ErrCIDRTooBroad):
		return httpapi.UpdateNetworkPolicy400JSONResponse(errMsg(err.Error()))
	default:
		h.logger.ErrorContext(ctx, "failed to update network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.UpdateNetworkPolicy500JSONResponse(errMsg("Failed to update network policy"))
	}
}

func errMsg(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
