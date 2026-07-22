package device

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) ListDeviceRefs(
	ctx context.Context,
	_ httpapi.ListDeviceRefsRequestObject,
) (httpapi.ListDeviceRefsResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListDeviceRefs")

	refs, err := h.repo.GetDeviceRefs(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list device refs", slog.Any(AttrKeyError, err))
		return httpapi.ListDeviceRefs500JSONResponse(errorMsgResponse("Failed to list device refs")), nil
	}
	return httpapi.ListDeviceRefs200JSONResponse(refs), nil
}
