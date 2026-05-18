package queries

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) ListHosts(
	ctx context.Context,
	_ httpapi.ListHostsRequestObject,
) (httpapi.ListHostsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHosts")

	response, err := h.repo.GetAllHostsWithGroups(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list hosts failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHosts500JSONResponse(errorMsgResponse("Failed to list hosts")), nil
	}
	return httpapi.ListHosts200JSONResponse(response), nil
}

func (h *HTTPHandler) ListHostGroups(
	ctx context.Context,
	_ httpapi.ListHostGroupsRequestObject,
) (httpapi.ListHostGroupsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHostGroups")

	response, err := h.repo.GetHostGroupsDetails(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list host groups failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHostGroups500JSONResponse(errorMsgResponse("Failed to list host groups")), nil
	}
	return httpapi.ListHostGroups200JSONResponse(response), nil
}

func (h *HTTPHandler) ListHostSuggestions(
	ctx context.Context,
	_ httpapi.ListHostSuggestionsRequestObject,
) (httpapi.ListHostSuggestionsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListHostSuggestions")

	page, err := h.repo.GetHostSuggestionsPage(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list host suggestions failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListHostSuggestions500JSONResponse(errorMsgResponse("Failed to list host suggestions")), nil
	}
	return httpapi.ListHostSuggestions200JSONResponse(page), nil
}

func (h *HTTPHandler) ListUsersWithAccess(
	ctx context.Context,
	_ httpapi.ListUsersWithAccessRequestObject,
) (httpapi.ListUsersWithAccessResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListUsersWithAccess")

	rows, err := h.repo.ListUserAccessRows(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "list users host access failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListUsersWithAccess500JSONResponse(errorMsgResponse("Failed to list users host access")), nil
	}
	return httpapi.ListUsersWithAccess200JSONResponse(rows), nil
}

func (h *HTTPHandler) GetUserAccessDetail(
	ctx context.Context,
	request httpapi.GetUserAccessDetailRequestObject,
) (httpapi.GetUserAccessDetailResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetUserAccessDetail")
	userID := ids.UserID(request.UserId)

	accessDetail, err := h.repo.GetUserAccessDetail(ctx, userID)
	if err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			return httpapi.GetUserAccessDetail404JSONResponse(errorMsgResponse("User not found")), nil
		}
		h.logger.ErrorContext(ctx, "get user host details failed", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetUserAccessDetail500JSONResponse(errorMsgResponse("Failed to get user host details")), nil
	}
	return httpapi.GetUserAccessDetail200JSONResponse(accessDetail), nil
}
