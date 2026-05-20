package queries

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	"github.com/DiegoGuidaF/PulseWeaver/internal/networkpolicies"
)

func (h *HTTPHandler) ListNetworkPolicies(
	ctx context.Context,
	_ httpapi.ListNetworkPoliciesRequestObject,
) (httpapi.ListNetworkPoliciesResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListNetworkPolicies")

	summaries, err := h.repo.GetNetworkPolicySummaries(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list network policies", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListNetworkPolicies500JSONResponse(errorMsgResponse("Failed to list network policies")), nil
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

	id := ids.NetworkPolicyID(request.Id)
	detail, err := h.repo.GetNetworkPolicyDetail(ctx, id)
	if err != nil {
		if errors.Is(err, networkpolicies.ErrNotFound) {
			return httpapi.GetNetworkPolicy404JSONResponse(errorMsgResponse("Network policy not found")), nil
		}
		h.logger.ErrorContext(ctx, "failed to get network policy", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetNetworkPolicy500JSONResponse(errorMsgResponse("Failed to get network policy")), nil
	}
	return httpapi.GetNetworkPolicy200JSONResponse(toNetworkPolicyDetailResponse(*detail)), nil
}

func toNetworkPolicySummaryResponse(s NetworkPolicySummaryView) httpapi.NetworkPolicyListItem {
	groups := make([]httpapi.GroupRef, len(s.Groups))
	for i, g := range s.Groups {
		groups[i] = httpapi.GroupRef{Id: g.ID.Int64(), Name: g.Name}
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
			Id:              g.ID,
			Name:            g.Name,
			Color:           color,
			Icon:            icon,
			Hosts:           hosts,
			Granted:         g.Assigned,
			NetworkPolicies: []httpapi.NetworkPolicyRef{},
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
		CreatedAt:       httpapi.UTCTime(d.CreatedAt),
		UpdatedAt:       httpapi.UTCTime(d.UpdatedAt),
	}
}
