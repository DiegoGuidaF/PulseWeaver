package queries

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

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
