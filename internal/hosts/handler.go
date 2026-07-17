package hosts

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	service *Service
	repo    *Repository
	logger  *slog.Logger
}

func NewHTTPHandler(service *Service, repo *Repository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		repo:    repo,
		logger:  logger.With(slog.String(logging.AttrKeyComponent, "hosts")),
	}
}

func (h *HTTPHandler) ListHosts(
	ctx context.Context,
	_ httpapi.ListHostsRequestObject,
) (httpapi.ListHostsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHosts")

	response, err := h.repo.GetAllHostsWithGroups(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list hosts failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHosts500JSONResponse(errResp("Failed to list hosts")), nil
	}
	return httpapi.ListHosts200JSONResponse(response), nil
}

func (h *HTTPHandler) ListHostGroups(
	ctx context.Context,
	_ httpapi.ListHostGroupsRequestObject,
) (httpapi.ListHostGroupsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHostGroups")

	response, err := h.repo.GetHostGroupsList(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list host groups failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHostGroups500JSONResponse(errResp("Failed to list host groups")), nil
	}
	return httpapi.ListHostGroups200JSONResponse(response), nil
}

func (h *HTTPHandler) GetHostGroup(
	ctx context.Context,
	request httpapi.GetHostGroupRequestObject,
) (httpapi.GetHostGroupResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetHostGroup")

	id := ids.HostGroupID(request.GroupId)
	detail, err := h.repo.GetHostGroupDetail(ctx, id)
	if err != nil {
		if errors.Is(err, ErrHostGroupNotFound) {
			return httpapi.GetHostGroup404JSONResponse(errResp("Host group not found")), nil
		}
		h.logger.ErrorContext(ctx, "get host group failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetHostGroup500JSONResponse(errResp("Failed to get host group")), nil
	}
	return httpapi.GetHostGroup200JSONResponse(*detail), nil
}

func (h *HTTPHandler) ReconcileHosts(
	ctx context.Context,
	req httpapi.ReconcileHostsRequestObject,
) (httpapi.ReconcileHostsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ReconcileHosts")

	in := ReconcileHostsInput{
		Hosts: make([]DesiredHost, 0, len(req.Body.Hosts)),
	}
	for _, item := range req.Body.Hosts {
		desired := DesiredHost{
			FQDN:     item.Fqdn,
			GroupIDs: make([]ids.HostGroupID, len(item.GroupIds)),
		}
		if item.Id != nil {
			desired.ID = new(ids.HostID(*item.Id))
		}
		for i, gid := range item.GroupIds {
			desired.GroupIDs[i] = ids.HostGroupID(gid)
		}
		in.Hosts = append(in.Hosts, desired)
	}

	if err := h.service.ReconcileHosts(ctx, in); err != nil {
		switch {
		case errors.Is(err, ErrBadRequest),
			errors.Is(err, ErrDuplicateHostID),
			errors.Is(err, ErrDuplicateHostFQDN),
			errors.Is(err, ErrHostFQDNImmutable):
			return httpapi.ReconcileHosts400JSONResponse(errResp(err.Error())), nil
		case errors.Is(err, ErrHostNotFound), errors.Is(err, ErrReferenceNotFound):
			return httpapi.ReconcileHosts404JSONResponse(errResp(err.Error())), nil
		case errors.Is(err, ErrHostConflict):
			return httpapi.ReconcileHosts409JSONResponse(errResp("FQDN already exists")), nil
		default:
			h.logger.ErrorContext(ctx, "reconcile hosts failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.ReconcileHosts500JSONResponse(errResp("Failed to reconcile hosts")), nil
		}
	}
	h.logger.InfoContext(ctx, "hosts reconciled", slog.Int("host_count", len(in.Hosts)))
	return httpapi.ReconcileHosts204Response{}, nil
}

func (h *HTTPHandler) ReconcileHostGroups(
	ctx context.Context,
	req httpapi.ReconcileHostGroupsRequestObject,
) (httpapi.ReconcileHostGroupsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ReconcileHostGroups")

	in := ReconcileHostGroupsInput{
		Groups: make([]DesiredHostGroup, 0, len(req.Body.Groups)),
	}
	for _, g := range req.Body.Groups {
		desired := DesiredHostGroup{
			Name:        g.Name,
			Color:       g.Color,
			Description: g.Description,
			Icon:        g.Icon,
		}
		if g.Id != nil {
			desired.ID = new(ids.HostGroupID(*g.Id))
		}
		desired.HostIDs = make([]ids.HostID, len(g.HostIds))
		for i, raw := range g.HostIds {
			desired.HostIDs[i] = ids.HostID(raw)
		}
		in.Groups = append(in.Groups, desired)
	}

	if err := h.service.ReconcileHostGroups(ctx, in); err != nil {
		switch {
		case errors.Is(err, ErrGroupNameRequired),
			errors.Is(err, ErrInvalidGroupColor),
			errors.Is(err, ErrDuplicateGroupID):
			return httpapi.ReconcileHostGroups400JSONResponse(errResp(err.Error())), nil
		case errors.Is(err, ErrHostGroupNotFound), errors.Is(err, ErrReferenceNotFound):
			return httpapi.ReconcileHostGroups404JSONResponse(errResp(err.Error())), nil
		case errors.Is(err, ErrHostGroupConflict):
			return httpapi.ReconcileHostGroups409JSONResponse(errResp("Host group name already taken")), nil
		default:
			h.logger.ErrorContext(ctx, "reconcile host groups failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.ReconcileHostGroups500JSONResponse(errResp("Failed to reconcile host groups")), nil
		}
	}
	h.logger.InfoContext(ctx, "host groups reconciled", slog.Int("group_count", len(in.Groups)))
	return httpapi.ReconcileHostGroups204Response{}, nil
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
	h.logger.InfoContext(ctx, "host suggestion ignored", slog.Int64("suggestion_id", s.ID))
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

	if err := h.service.RemoveIgnoredSuggestionByFQDN(ctx, req.Fqdn); err != nil {
		if errors.Is(err, ErrSuggestionNotFound) {
			return httpapi.UnignoreSuggestion404JSONResponse(errResp("Ignored suggestion not found")), nil
		}
		h.logger.ErrorContext(ctx, "unignore suggestion failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.UnignoreSuggestion500JSONResponse(errResp("Failed to unignore suggestion")), nil
	}
	h.logger.InfoContext(ctx, "host suggestion unignored", slog.String("fqdn", req.Fqdn))
	return httpapi.UnignoreSuggestion204Response{}, nil
}

func errResp(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
