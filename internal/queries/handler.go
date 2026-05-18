package queries

import (
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

type HTTPHandler struct {
	repo         *Repository
	policyReader PolicyMapReader
	npProvider   AuditNetworkPoliciesProvider
	logger       *slog.Logger
}

func NewHTTPHandler(repo *Repository, policyReader PolicyMapReader, npProvider AuditNetworkPoliciesProvider, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		repo:         repo,
		policyReader: policyReader,
		npProvider:   npProvider,
		logger:       logger.With(slog.String(logging.AttrKeyComponent, "queries")),
	}
}

func errorMsgResponse(msg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &msg}
}
