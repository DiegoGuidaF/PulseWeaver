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

func (h *HTTPHandler) ReconcileKnownHosts(
	ctx context.Context,
	req httpapi.ReconcileKnownHostsRequestObject,
) (httpapi.ReconcileKnownHostsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ReconcileKnownHosts")

	in := ReconcileKnownHostsInput{
		Hosts: make([]DesiredKnownHost, 0, len(req.Body.Hosts)),
	}
	for _, h := range req.Body.Hosts {
		desired := DesiredKnownHost{
			FQDN:     h.Fqdn,
			Icon:     h.Icon,
			GroupIDs: make([]HostGroupID, len(h.GroupIds)),
		}
		if h.Id != nil {
			desired.ID = new(KnownHostID(*h.Id))
		}
		for i, gid := range h.GroupIds {
			desired.GroupIDs[i] = HostGroupID(gid)
		}
		in.Hosts = append(in.Hosts, desired)
	}

	if err := h.service.ReconcileKnownHosts(ctx, in); err != nil {
		switch {
		case errors.Is(err, ErrBadRequest),
			errors.Is(err, ErrDuplicateKnownHostID),
			errors.Is(err, ErrDuplicateKnownHostFQDN),
			errors.Is(err, ErrKnownHostFQDNImmutable):
			return httpapi.ReconcileKnownHosts400JSONResponse(errResp(err.Error())), nil
		case errors.Is(err, ErrKnownHostNotFound), errors.Is(err, ErrReferenceNotFound):
			return httpapi.ReconcileKnownHosts404JSONResponse(errResp(err.Error())), nil
		case errors.Is(err, ErrKnownHostConflict):
			return httpapi.ReconcileKnownHosts409JSONResponse(errResp("FQDN already exists")), nil
		default:
			h.logger.ErrorContext(ctx, "reconcile known hosts failed", slog.Any(logging.AttrKeyError, err))
			return httpapi.ReconcileKnownHosts500JSONResponse(errResp("Failed to reconcile known hosts")), nil
		}
	}
	return httpapi.ReconcileKnownHosts204Response{}, nil
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
			desired.ID = new(HostGroupID(*g.Id))
		}
		if g.HostIds != nil {
			desired.HostIDs = make([]KnownHostID, len(*g.HostIds))
			for i, raw := range *g.HostIds {
				desired.HostIDs[i] = KnownHostID(raw)
			}
		}
		in.Groups = append(in.Groups, desired)
	}

	if err := h.service.ReconcileHostGroups(ctx, in); err != nil {
		switch {
		case errors.Is(err, ErrGroupNameRequired), errors.Is(err, ErrDuplicateGroupID):
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
	return httpapi.ReconcileHostGroups204Response{}, nil
}

// ── User host grants ──────────────────────────────────────────────────────────

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

	if err := h.service.RemoveIgnoredSuggestionByFQDN(ctx, req.Fqdn); err != nil {
		if errors.Is(err, ErrSuggestionNotFound) {
			return httpapi.UnignoreSuggestion404JSONResponse(errResp("Ignored suggestion not found")), nil
		}
		h.logger.ErrorContext(ctx, "unignore suggestion failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.UnignoreSuggestion500JSONResponse(errResp("Failed to unignore suggestion")), nil
	}
	return httpapi.UnignoreSuggestion204Response{}, nil
}

// ── DTO mappers ───────────────────────────────────────────────────────────────
func errResp(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
