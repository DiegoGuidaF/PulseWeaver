package buildmeta

import (
	"context"
	"log/slog"
	"time"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	logger *slog.Logger
}

func NewHTTPHandler(logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		logger: logger.With(slog.String(logging.AttrKeyComponent, "version")),
	}
}

func (h *HTTPHandler) GetVersion(
	ctx context.Context,
	_ httpapi.GetVersionRequestObject,
) (httpapi.GetVersionResponseObject, error) {
	_ = logging.WithOperation(ctx, "GetVersion")

	info := Get()
	resp := httpapi.VersionInfo{
		Version: info.Version,
		Commit:  info.Commit,
	}
	// An un-stamped build has no build time; leave the field absent rather
	// than emitting a zero timestamp the UI would have to special-case.
	if info.BuildTime != "" {
		if t, err := time.Parse(time.RFC3339, info.BuildTime); err == nil {
			resp.BuildTime = &t
		}
	}

	return httpapi.GetVersion200JSONResponse(resp), nil
}
