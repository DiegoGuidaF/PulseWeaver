package hostaccess

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "hostaccess")),
	}
}

// ── Known hosts ───────────────────────────────────────────────────────────────

func (h *HTTPHandler) CreateKnownHosts(
	ctx context.Context,
	req httpapi.CreateKnownHostsRequestObject,
) (httpapi.CreateKnownHostsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateKnownHosts")

	hosts, err := h.service.BulkCreateKnownHosts(ctx, req.Body.Fqdns)
	if err != nil {
		if errors.Is(err, ErrKnownHostConflict) {
			return httpapi.CreateKnownHosts409JSONResponse(errResp("One or more FQDNs are already registered")), nil
		}
		if isValidationError(err) {
			return httpapi.CreateKnownHosts400JSONResponse(errResp(err.Error())), nil
		}
		h.logger.ErrorContext(ctx, "bulk create known hosts failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.CreateKnownHosts500JSONResponse(errResp("Failed to create hosts")), nil
	}

	resp := make([]httpapi.KnownHost, len(hosts))
	for i, h := range hosts {
		resp[i] = toKnownHostDTO(h)
	}
	return httpapi.CreateKnownHosts201JSONResponse(resp), nil
}

func (h *HTTPHandler) UpdateKnownHost(
	ctx context.Context,
	req httpapi.UpdateKnownHostRequestObject,
) (httpapi.UpdateKnownHostResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UpdateKnownHost")
	id := KnownHostID(req.HostId)

	host, err := h.service.UpdateKnownHost(ctx, id, req.Body.Icon.Value)
	if err != nil {
		if errors.Is(err, ErrKnownHostNotFound) {
			return httpapi.UpdateKnownHost404JSONResponse(errResp("Host not found")), nil
		}
		h.logger.ErrorContext(ctx, "update known host failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.UpdateKnownHost500JSONResponse(errResp("Failed to update host")), nil
	}
	return httpapi.UpdateKnownHost200JSONResponse(toKnownHostDTO(host)), nil
}

func (h *HTTPHandler) DeleteKnownHost(
	ctx context.Context,
	req httpapi.DeleteKnownHostRequestObject,
) (httpapi.DeleteKnownHostResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteKnownHost")
	id := KnownHostID(req.HostId)

	if err := h.service.DeleteKnownHost(ctx, id); err != nil {
		if errors.Is(err, ErrKnownHostNotFound) {
			return httpapi.DeleteKnownHost404JSONResponse(errResp("Host not found")), nil
		}
		h.logger.ErrorContext(ctx, "delete known host failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.DeleteKnownHost500JSONResponse(errResp("Failed to delete host")), nil
	}
	return httpapi.DeleteKnownHost204Response{}, nil
}

// ── Host groups ───────────────────────────────────────────────────────────────

func (h *HTTPHandler) ListHostGroups(
	ctx context.Context,
	_ httpapi.ListHostGroupsRequestObject,
) (httpapi.ListHostGroupsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHostGroups")

	groups, err := h.service.ListHostGroupsWithMembers(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list host groups failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHostGroups500JSONResponse(errResp("Failed to list host groups")), nil
	}

	resp := make([]httpapi.HostGroupWithMembers, len(groups))
	for i, g := range groups {
		resp[i] = toHostGroupWithMembersDTO(g)
	}
	return httpapi.ListHostGroups200JSONResponse(resp), nil
}

func (h *HTTPHandler) CreateHostGroup(
	ctx context.Context,
	req httpapi.CreateHostGroupRequestObject,
) (httpapi.CreateHostGroupResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateHostGroup")

	group, err := h.service.CreateHostGroup(ctx, req.Body.Name, req.Body.Description, req.Body.Icon)
	if err != nil {
		if errors.Is(err, ErrHostGroupConflict) {
			return httpapi.CreateHostGroup409JSONResponse(errResp("Host group name already exists")), nil
		}
		h.logger.ErrorContext(ctx, "create host group failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.CreateHostGroup500JSONResponse(errResp("Failed to create host group")), nil
	}
	return httpapi.CreateHostGroup201JSONResponse(toHostGroupDTO(group)), nil
}

func (h *HTTPHandler) UpdateHostGroup(
	ctx context.Context,
	req httpapi.UpdateHostGroupRequestObject,
) (httpapi.UpdateHostGroupResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UpdateHostGroup")
	id := HostGroupID(req.GroupId)

	group, err := h.service.UpdateHostGroup(ctx, id, req.Body.Name, req.Body.Description.Value, req.Body.Icon.Value)
	if err != nil {
		switch {
		case errors.Is(err, ErrHostGroupNotFound):
			return httpapi.UpdateHostGroup404JSONResponse(errResp("Host group not found")), nil
		case errors.Is(err, ErrHostGroupConflict):
			return httpapi.UpdateHostGroup409JSONResponse(errResp("Host group name already taken")), nil
		default:
			h.logger.ErrorContext(ctx, "update host group failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.UpdateHostGroup500JSONResponse(errResp("Failed to update host group")), nil
		}
	}
	return httpapi.UpdateHostGroup200JSONResponse(toHostGroupDTO(group)), nil
}

func (h *HTTPHandler) DeleteHostGroup(
	ctx context.Context,
	req httpapi.DeleteHostGroupRequestObject,
) (httpapi.DeleteHostGroupResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteHostGroup")
	id := HostGroupID(req.GroupId)

	if err := h.service.DeleteHostGroup(ctx, id); err != nil {
		if errors.Is(err, ErrHostGroupNotFound) {
			return httpapi.DeleteHostGroup404JSONResponse(errResp("Host group not found")), nil
		}
		h.logger.ErrorContext(ctx, "delete host group failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.DeleteHostGroup500JSONResponse(errResp("Failed to delete host group")), nil
	}
	return httpapi.DeleteHostGroup204Response{}, nil
}

func (h *HTTPHandler) SetHostGroupMembers(
	ctx context.Context,
	req httpapi.SetHostGroupMembersRequestObject,
) (httpapi.SetHostGroupMembersResponseObject, error) {
	ctx = logging.WithOperation(ctx, "SetHostGroupMembers")
	groupID := HostGroupID(req.GroupId)

	hostIDs := make([]KnownHostID, len(req.Body.HostIds))
	for i, id := range req.Body.HostIds {
		hostIDs[i] = KnownHostID(id)
	}

	if err := h.service.SetHostGroupMembers(ctx, groupID, hostIDs); err != nil {
		switch {
		case errors.Is(err, ErrReferenceNotFound):
			return httpapi.SetHostGroupMembers404JSONResponse(errResp("Group or one of the hosts not found")), nil
		default:
			h.logger.ErrorContext(ctx, "set host group members failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.SetHostGroupMembers500JSONResponse(errResp("Failed to set host group members")), nil
		}
	}
	return httpapi.SetHostGroupMembers204Response{}, nil
}

// ── User host grants ──────────────────────────────────────────────────────────

func (h *HTTPHandler) GetUserHostGrants(
	ctx context.Context,
	req httpapi.GetUserHostGrantsRequestObject,
) (httpapi.GetUserHostGrantsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetUserHostGrants")
	userID := auth.UserID(req.UserId)

	grants, err := h.service.GetFullUserGrants(ctx, userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return httpapi.GetUserHostGrants404JSONResponse(errResp("User not found")), nil
		}
		h.logger.ErrorContext(ctx, "get user host grants failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetUserHostGrants500JSONResponse(errResp("Failed to get user grants")), nil
	}

	hostIDs := make([]int64, len(grants.Hosts))
	for i, host := range grants.Hosts {
		hostIDs[i] = host.ID.Int64()
	}
	groupIDs := make([]int64, len(grants.Groups))
	for i, group := range grants.Groups {
		groupIDs[i] = group.ID.Int64()
	}

	return httpapi.GetUserHostGrants200JSONResponse(httpapi.UserHostGrants{
		Bypass:   grants.Bypass,
		HostIds:  hostIDs,
		GroupIds: groupIDs,
	}), nil
}

func (h *HTTPHandler) SetUserHostGrants(
	ctx context.Context,
	req httpapi.SetUserHostGrantsRequestObject,
) (httpapi.SetUserHostGrantsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "SetUserHostGrants")
	userID := auth.UserID(req.UserId)

	var hostIDs []KnownHostID
	if req.Body.HostIds != nil {
		hostIDs = make([]KnownHostID, len(*req.Body.HostIds))
		for i, id := range *req.Body.HostIds {
			hostIDs[i] = KnownHostID(id)
		}
	}

	var groupIDs []HostGroupID
	if req.Body.GroupIds != nil {
		groupIDs = make([]HostGroupID, len(*req.Body.GroupIds))
		for i, id := range *req.Body.GroupIds {
			groupIDs[i] = HostGroupID(id)
		}
	}

	if err := h.service.SetFullUserGrants(ctx, userID, req.Body.Bypass, hostIDs, groupIDs); err != nil {
		switch {
		case errors.Is(err, ErrReferenceNotFound), errors.Is(err, auth.ErrUserNotFound):
			return httpapi.SetUserHostGrants404JSONResponse(errResp("User or one of the referenced hosts/groups not found")), nil
		default:
			h.logger.ErrorContext(ctx, "set user host grants failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.SetUserHostGrants500JSONResponse(errResp("Failed to set user grants")), nil
		}
	}

	return httpapi.SetUserHostGrants204Response{}, nil
}

// ── Ignored suggestions ───────────────────────────────────────────────────────

func (h *HTTPHandler) ListIgnoredSuggestions(
	ctx context.Context,
	_ httpapi.ListIgnoredSuggestionsRequestObject,
) (httpapi.ListIgnoredSuggestionsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListIgnoredSuggestions")

	suggestions, err := h.service.ListIgnoredSuggestions(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list ignored suggestions failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListIgnoredSuggestions500JSONResponse(errResp("Failed to list ignored suggestions")), nil
	}

	resp := make([]httpapi.IgnoredHostSuggestion, len(suggestions))
	for i, s := range suggestions {
		resp[i] = httpapi.IgnoredHostSuggestion{
			Id:        s.ID,
			Fqdn:      s.FQDN,
			CreatedAt: httpapi.UTCTime(s.CreatedAt),
		}
	}
	return httpapi.ListIgnoredSuggestions200JSONResponse(resp), nil
}

func (h *HTTPHandler) IgnoreSuggestion(
	ctx context.Context,
	req httpapi.IgnoreSuggestionRequestObject,
) (httpapi.IgnoreSuggestionResponseObject, error) {
	ctx = logging.WithOperation(ctx, "IgnoreSuggestion")

	s, err := h.service.AddIgnoredSuggestion(ctx, req.Body.Fqdn)
	if err != nil {
		if errors.Is(err, ErrSuggestionConflict) {
			return httpapi.IgnoreSuggestion409JSONResponse(errResp("FQDN already ignored")), nil
		}
		h.logger.ErrorContext(ctx, "ignore suggestion failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.IgnoreSuggestion500JSONResponse(errResp("Failed to ignore suggestion")), nil
	}
	return httpapi.IgnoreSuggestion201JSONResponse(httpapi.IgnoredHostSuggestion{
		Id:        s.ID,
		Fqdn:      s.FQDN,
		CreatedAt: httpapi.UTCTime(s.CreatedAt),
	}), nil
}

func (h *HTTPHandler) UnignoreSuggestion(
	ctx context.Context,
	req httpapi.UnignoreSuggestionRequestObject,
) (httpapi.UnignoreSuggestionResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UnignoreSuggestion")

	suggestion, err := h.service.FindIgnoredSuggestionByFQDN(ctx, req.Fqdn)
	if err != nil {
		if errors.Is(err, ErrSuggestionNotFound) {
			return httpapi.UnignoreSuggestion404JSONResponse(errResp("Ignored suggestion not found")), nil
		}
		h.logger.ErrorContext(ctx, "find ignored suggestion failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.UnignoreSuggestion500JSONResponse(errResp("Failed to unignore suggestion")), nil
	}

	if err := h.service.RemoveIgnoredSuggestion(ctx, suggestion.ID); err != nil {
		h.logger.ErrorContext(ctx, "unignore suggestion failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.UnignoreSuggestion500JSONResponse(errResp("Failed to unignore suggestion")), nil
	}
	return httpapi.UnignoreSuggestion204Response{}, nil
}

// ── DTO mappers ───────────────────────────────────────────────────────────────

func toKnownHostDTO(h KnownHost) httpapi.KnownHost {
	return httpapi.KnownHost{
		Id:        h.ID.Int64(),
		Fqdn:      h.FQDN,
		Icon:      h.Icon,
		CreatedAt: httpapi.UTCTime(h.CreatedAt),
	}
}

func toHostGroupDTO(g HostGroup) httpapi.HostGroup {
	return httpapi.HostGroup{
		Id:          g.ID.Int64(),
		Name:        g.Name,
		Description: g.Description,
		Icon:        g.Icon,
		CreatedAt:   httpapi.UTCTime(g.CreatedAt),
	}
}

func toHostGroupWithMembersDTO(g HostGroupWithMembers) httpapi.HostGroupWithMembers {
	hostIDs := make([]int64, len(g.HostIDs))
	for i, id := range g.HostIDs {
		hostIDs[i] = id.Int64()
	}
	return httpapi.HostGroupWithMembers{
		Id:          g.ID.Int64(),
		Name:        g.Name,
		Description: g.Description,
		Icon:        g.Icon,
		CreatedAt:   httpapi.UTCTime(g.CreatedAt),
		HostIds:     hostIDs,
	}
}

func errResp(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrBadRequest)
}
