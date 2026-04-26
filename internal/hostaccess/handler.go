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
	for i, kh := range hosts {
		resp[i] = toKnownHostDTO(kh)
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

// ── Host groups (reconcile) ───────────────────────────────────────────────────

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
			id := HostGroupID(*g.Id)
			desired.ID = &id
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

// ── Ignored suggestions ───────────────────────────────────────────────────────

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

func toKnownHostDTO(h KnownHost) httpapi.KnownHost {
	return httpapi.KnownHost{
		Id:        h.ID.Int64(),
		Fqdn:      h.FQDN,
		Icon:      h.Icon,
		CreatedAt: httpapi.UTCTime(h.CreatedAt),
	}
}

func errResp(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}

func isValidationError(err error) bool {
	return errors.Is(err, ErrBadRequest)
}
