package queries

import (
	"context"
	"log/slog"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
)

func (h *HTTPHandler) ListUsers(
	ctx context.Context,
	_ httpapi.ListUsersRequestObject,
) (httpapi.ListUsersResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ListUsers")

	users, err := h.repo.GetAllUsers(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to list users", slog.Any(logging.AttrKeyError, err))
		return httpapi.ListUsers500JSONResponse(errorMsgResponse("Failed to list users")), nil
	}

	response := make(httpapi.ListUsers200JSONResponse, 0, len(users))
	for _, u := range users {
		response = append(response, toUserViewResponse(&u))
	}
	return response, nil
}

func toUserViewResponse(u *UserView) httpapi.User {
	return httpapi.User{
		Id:                 u.ID.Int64(),
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Email:              openapi_types.Email(u.Email),
		Role:               httpapi.UserRole(u.Role),
		MustChangePassword: new(u.MustChangePassword),
		BypassHostCheck:    u.BypassHostCheck,
		CreatedAt:          httpapi.UTCTime(u.CreatedAt),
	}
}
