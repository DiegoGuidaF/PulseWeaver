package queries

import (
	"context"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

// GetDashboardPosture returns the current-state posture counts for the dashboard
// landing page, reduced from the policy cache snapshot.
func (h *HTTPHandler) GetDashboardPosture(
	ctx context.Context,
	_ httpapi.GetDashboardPostureRequestObject,
) (httpapi.GetDashboardPostureResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetDashboardPosture")

	posture, err := h.repo.BuildDashboardPosture(ctx, h.ipProvider)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to build dashboard posture", slog.Any(logging.AttrKeyError, err))
		return httpapi.GetDashboardPosture500JSONResponse(errorMsgResponse("Failed to load dashboard posture")), nil
	}
	return httpapi.GetDashboardPosture200JSONResponse(posture), nil
}
