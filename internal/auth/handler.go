package auth

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/logging"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type HTTPHandler struct {
	service      *Service
	cookieConfig CookieConfig
	logger       *slog.Logger
}

func (h *HTTPHandler) Login(ctx context.Context, request httpapi.LoginRequestObject) (httpapi.LoginResponseObject, error) {
	ctx = logging.WithOperation(ctx, "Login")
	username := request.Body.Username
	logger := h.logger.With(slog.String(AttrKeyUsername, username))

	rawToken, user, err := h.service.Login(ctx, username, request.Body.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserNotFound) {
			logger.WarnContext(ctx, "invalid credentials")
			return httpapi.Login401JSONResponse(errorMsgResponse("Invalid credentials")), nil
		}
		logger.ErrorContext(ctx, "login failed", slog.Any(AttrKeyError, err))
		return httpapi.Login500JSONResponse(errorMsgResponse("Login failure")), nil
	}

	cookie := NewSessionCookie(rawToken, h.cookieConfig)

	headers := httpapi.Login200ResponseHeaders{SetCookie: cookie.String()}

	return httpapi.Login200JSONResponse{Body: toUserResponse(user), Headers: headers}, nil
}

func (h *HTTPHandler) Logout(ctx context.Context, _ httpapi.LogoutRequestObject) (httpapi.LogoutResponseObject, error) {
	ctx = logging.WithOperation(ctx, "Logout")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if ok {
		sessionLogger := logger.With(slog.Int64(AttrKeySessionID, principal.SessionID.Int64()))
		err := h.service.RevokeSession(ctx, principal.SessionID)
		if err != nil {
			sessionLogger.ErrorContext(ctx, "failed to revoke session", slog.Any(AttrKeyError, err))
		}
	}

	cookie := ExpireSessionCookie(h.cookieConfig)

	headers := httpapi.Logout204ResponseHeaders{SetCookie: cookie.String()}

	return httpapi.Logout204Response{Headers: headers}, nil
}

func (h *HTTPHandler) GetCurrentUser(ctx context.Context, _ httpapi.GetCurrentUserRequestObject) (httpapi.GetCurrentUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "GetCurrentUser")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "principal not in context")
		return httpapi.GetCurrentUser500JSONResponse(errorMsgResponse("Couldn't retrieve current principal from context")), nil
	}
	logger = logger.With(slog.Int64(AttrKeyUserID, principal.UserID.Int64()))

	user, err := h.service.GetUserFromPrincipal(ctx, principal)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			logger.WarnContext(ctx, "user not found")
			return httpapi.GetCurrentUser500JSONResponse(errorMsgResponse("User not found")), nil
		}
		logger.ErrorContext(ctx, "failed to retrieve current user", slog.Any(AttrKeyError, err))
		return httpapi.GetCurrentUser500JSONResponse(errorMsgResponse("Failed to retrieve current user")), nil
	}

	return httpapi.GetCurrentUser200JSONResponse(toUserResponse(user)), nil
}

func (h *HTTPHandler) CreateUser(ctx context.Context, request httpapi.CreateUserRequestObject) (httpapi.CreateUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "CreateUser")
	username := request.Body.Username
	logger := h.logger.With(
		slog.String(AttrKeyUsername, username),
		slog.String(AttrKeyDisplayName, request.Body.DisplayName),
	)

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.ErrorContext(ctx, "principal not in context")
		return httpapi.CreateUser401JSONResponse{}, nil
	}

	var email *string
	if request.Body.Email != nil {
		email = new(string(*request.Body.Email))
	}
	user, err := h.service.CreateUser(ctx, username, request.Body.DisplayName, email, principal)

	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			logger.WarnContext(ctx, "username already taken")
			return httpapi.CreateUser409JSONResponse(errorMsgResponse("User with that username already exists")), nil
		}
		if errors.Is(err, ErrEmailTaken) {
			logger.WarnContext(ctx, "email already taken")
			return httpapi.CreateUser409JSONResponse(errorMsgResponse("User with that email already exists")), nil
		}
		if errors.Is(err, ErrInvalidDisplayName) || errors.Is(err, ErrInvalidUsername) || errors.Is(err, ErrInvalidPassword) {
			logger.WarnContext(ctx, "invalid input")
			return httpapi.CreateUser400JSONResponse(errorMsgResponse("Invalid input")), nil
		}
		logger.ErrorContext(ctx, "failed to create user", slog.Any(AttrKeyError, err))
		return httpapi.CreateUser500JSONResponse(errorMsgResponse("Failed to create user")), nil
	}

	logger.InfoContext(ctx, "new user created", slog.Int64(AttrKeyUserID, user.ID.Int64()))

	return httpapi.CreateUser201JSONResponse(toUserResponse(user)), nil
}

func (h *HTTPHandler) UpdateMe(ctx context.Context, request httpapi.UpdateMeRequestObject) (httpapi.UpdateMeResponseObject, error) {
	ctx = logging.WithOperation(ctx, "UpdateMe")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "principal not in context")
		return httpapi.UpdateMe401JSONResponse(errorMsgResponse("Not authenticated")), nil
	}

	updates := ProfileUpdates{
		Username:    request.Body.Username,
		DisplayName: request.Body.DisplayName,
	}
	if request.Body.Email.Set {
		updates.Email = &request.Body.Email.Value
	}

	user, err := h.service.UpdateOwnProfile(ctx, principal.UserID, updates)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidDisplayName), errors.Is(err, ErrInvalidUsername), errors.Is(err, ErrNoUpdateFields):
			logger.WarnContext(ctx, "invalid profile update input")
			return httpapi.UpdateMe400JSONResponse(errorMsgResponse("Invalid input")), nil
		case errors.Is(err, ErrUsernameTaken):
			logger.WarnContext(ctx, "username already taken")
			return httpapi.UpdateMe409JSONResponse(errorMsgResponse("User with that username already exists")), nil
		case errors.Is(err, ErrEmailTaken):
			logger.WarnContext(ctx, "email already taken")
			return httpapi.UpdateMe409JSONResponse(errorMsgResponse("User with that email already exists")), nil
		default:
			logger.ErrorContext(ctx, "failed to update profile", slog.Any(AttrKeyError, err))
			return httpapi.UpdateMe500JSONResponse(errorMsgResponse("Failed to update profile")), nil
		}
	}

	return httpapi.UpdateMe200JSONResponse(toUserResponse(user)), nil
}

func (h *HTTPHandler) ChangePassword(ctx context.Context, request httpapi.ChangePasswordRequestObject) (httpapi.ChangePasswordResponseObject, error) {
	ctx = logging.WithOperation(ctx, "ChangePassword")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "principal not in context")
		return httpapi.ChangePassword401JSONResponse(errorMsgResponse("Not authenticated")), nil
	}

	err := h.service.ChangePassword(ctx, principal.UserID, principal.SessionID, request.Body.CurrentPassword, request.Body.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInvalidPassword):
			logger.WarnContext(ctx, "invalid password change request")
			return httpapi.ChangePassword400JSONResponse(errorMsgResponse("Invalid password change request")), nil
		default:
			logger.ErrorContext(ctx, "failed to change password", slog.Any(AttrKeyError, err))
			return httpapi.ChangePassword500JSONResponse(errorMsgResponse("Failed to change password")), nil
		}
	}

	return httpapi.ChangePassword204Response{}, nil
}

func (h *HTTPHandler) PromoteUser(ctx context.Context, request httpapi.PromoteUserRequestObject) (httpapi.PromoteUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "PromoteUser")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "principal not in context")
		return httpapi.PromoteUser401JSONResponse(errorMsgResponse("Not authenticated")), nil
	}

	user, err := h.service.PromoteUser(ctx, principal, ids.UserID(request.UserId), request.Body.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrSelfRoleChangeForbidden) || errors.Is(err, ErrPromoteAlreadyAdmin):
			logger.WarnContext(ctx, "forbidden already an admin or self-promotion")
			return httpapi.PromoteUser403JSONResponse(errorMsgResponse("Cannot promote an admin or yourself")), nil
		case errors.Is(err, ErrAdminCredentialsRequired):
			logger.WarnContext(ctx, "admin credentials required")
			return httpapi.PromoteUser403JSONResponse(errorMsgResponse("admin credentials required")), nil
		case errors.Is(err, ErrInvalidPassword):
			logger.WarnContext(ctx, "invalid password for promotion")
			return httpapi.PromoteUser400JSONResponse(errorMsgResponse("Invalid password")), nil
		case errors.Is(err, ErrUserNotFound):
			logger.WarnContext(ctx, "target user not found")
			return httpapi.PromoteUser404JSONResponse(errorMsgResponse("User not found")), nil
		default:
			logger.ErrorContext(ctx, "failed to promote user", slog.Any(AttrKeyError, err))
			return httpapi.PromoteUser500JSONResponse(errorMsgResponse("Failed to promote user")), nil
		}
	}

	return httpapi.PromoteUser200JSONResponse(toUserResponse(user)), nil
}

func (h *HTTPHandler) DemoteUser(ctx context.Context, request httpapi.DemoteUserRequestObject) (httpapi.DemoteUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DemoteUser")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "principal not in context")
		return httpapi.DemoteUser401JSONResponse(errorMsgResponse("Not authenticated")), nil
	}

	user, err := h.service.DemoteUser(ctx, principal, ids.UserID(request.UserId))
	if err != nil {
		switch {
		case errors.Is(err, ErrSelfRoleChangeForbidden):
			logger.WarnContext(ctx, "forbidden demotion")
			return httpapi.DemoteUser403JSONResponse(errorMsgResponse("Forbidden role change")), nil
		case errors.Is(err, ErrAdminCredentialsRequired):
			logger.WarnContext(ctx, "admin credentials required")
			return httpapi.DemoteUser403JSONResponse(errorMsgResponse("admin credentials required")), nil
		case errors.Is(err, ErrUserNotFound):
			logger.WarnContext(ctx, "target user not found")
			return httpapi.DemoteUser404JSONResponse(errorMsgResponse("User not found")), nil
		default:
			logger.ErrorContext(ctx, "failed to demote user", slog.Any(AttrKeyError, err))
			return httpapi.DemoteUser500JSONResponse(errorMsgResponse("Failed to demote user")), nil
		}
	}

	return httpapi.DemoteUser200JSONResponse(toUserResponse(user)), nil
}

func (h *HTTPHandler) DeleteUser(ctx context.Context, request httpapi.DeleteUserRequestObject) (httpapi.DeleteUserResponseObject, error) {
	ctx = logging.WithOperation(ctx, "DeleteUser")
	logger := h.logger

	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		logger.WarnContext(ctx, "principal not in context")
		return httpapi.DeleteUser401JSONResponse(errorMsgResponse("Not authenticated")), nil
	}

	err := h.service.DeleteUser(ctx, principal, ids.UserID(request.UserId))
	if err != nil {
		switch {
		case errors.Is(err, ErrAdminCredentialsRequired), errors.Is(err, ErrSelfDeleteForbidden):
			logger.WarnContext(ctx, "forbidden user delete")
			return httpapi.DeleteUser403JSONResponse(errorMsgResponse("Forbidden user delete")), nil
		case errors.Is(err, ErrUserNotFound):
			logger.WarnContext(ctx, "target user not found")
			return httpapi.DeleteUser404JSONResponse(errorMsgResponse("User not found")), nil
		default:
			logger.ErrorContext(ctx, "failed to delete user", slog.Any(AttrKeyError, err))
			return httpapi.DeleteUser500JSONResponse(errorMsgResponse("Failed to delete user")), nil
		}
	}

	return httpapi.DeleteUser204Response{}, nil
}

func NewHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	cfg := DefaultCookieConfig

	return &HTTPHandler{
		service:      service,
		cookieConfig: cfg,
		logger:       logger.With(slog.String(logging.AttrKeyComponent, "auth")),
	}
}

func (h *HTTPHandler) UserAuthenticator() UserAuthenticator {
	return h.service
}

func toUserResponse(d *User) httpapi.User {
	resp := httpapi.User{
		Id:                 d.ID.Int64(),
		Username:           d.Username,
		DisplayName:        d.DisplayName,
		Role:               httpapi.UserRole(d.Role),
		MustChangePassword: new(d.MustChangePassword),
		CreatedAt:          httpapi.UTCTime(d.CreatedAt),
	}
	if d.Email != nil {
		resp.Email = new(openapi_types.Email(*d.Email))
	}
	return resp
}

func errorMsgResponse(errorMsg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &errorMsg}
}
