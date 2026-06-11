package queries

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
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

func toUserViewResponse(u *UserView) httpapi.User {
	var email *openapi_types.Email
	if u.Email != nil {
		email = new(openapi_types.Email(*u.Email))
	}
	return httpapi.User{
		Id:                 u.ID.Int64(),
		Username:           u.Username,
		DisplayName:        u.DisplayName,
		Email:              email,
		Role:               httpapi.UserRole(u.Role),
		MustChangePassword: new(u.MustChangePassword),
		BypassHostCheck:    u.BypassHostCheck,
		CreatedAt:          httpapi.UTCTime(u.CreatedAt),
	}
}
