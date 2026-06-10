package queries

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
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
