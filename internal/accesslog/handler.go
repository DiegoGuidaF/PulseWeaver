package accesslog

import (
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	repo   *Repository
	logger *slog.Logger
}

func NewHTTPHandler(repo *Repository, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:   repo,
		logger: logger.With(slog.String(logging.AttrKeyComponent, "accesslog")),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
