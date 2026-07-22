package queries

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) GetDeviceFleet(
	ctx context.Context,
	request httpapi.GetDeviceFleetRequestObject,
) (httpapi.GetDeviceFleetResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDeviceFleet")

	var ownerID *ids.UserID
	if request.Params.OwnerId != nil {
		ownerID = new(ids.UserID(*request.Params.OwnerId))
	}

	groups, err := h.fleet.Fleet(ctx, ownerID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to compose device fleet", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDeviceFleet500JSONResponse(errorMsgResponse("Failed to list device fleet")), nil
	}
	return httpapi.GetDeviceFleet200JSONResponse(groups), nil
}
