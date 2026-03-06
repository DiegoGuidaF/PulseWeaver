package auth

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DiegoGuidaF/WallyDex/internal/httpapi"
	"github.com/DiegoGuidaF/WallyDex/internal/logging"
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
		return httpapi.CreateUser403Response{}, nil
	}

	// Email is inside an openapi validator, we need to turn it into a valid string or nil
	var email *string
	if request.Body.Email != nil {
		email = new(string(*request.Body.Email))
	}
	user, err := h.service.CreateUserByAdmin(
		ctx,
		username,
		request.Body.DisplayName,
		email,
		request.Body.Password,
		principal,
	)

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
		if errors.Is(err, ErrAdminCredentialsRequired) {
			logger.WarnContext(ctx, "admin credentials required")
			return httpapi.CreateUser403Response{}, nil
		}
		logger.ErrorContext(ctx, "failed to create user", slog.Any(AttrKeyError, err))
		return httpapi.CreateUser500JSONResponse(errorMsgResponse("Failed to create user")), nil
	}

	logger.InfoContext(ctx, "new user created", slog.Int64(AttrKeyUserID, user.ID.Int64()))

	return httpapi.CreateUser201JSONResponse(toUserResponse(user)), nil
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
	var email *openapi_types.Email

	if d.Email != nil { // Check if email exists
		email = new(openapi_types.Email(*d.Email))
	}

	return httpapi.User{
		Id:          d.ID.Int64(),
		Username:    d.Username,
		DisplayName: d.DisplayName,
		Email:       email,
		CreatedAt:   httpapi.UTCTime(d.CreatedAt),
	}
}

func errorMsgResponse(errorMsg string) httpapi.ErrorResponse {
	return httpapi.ErrorResponse{Error: &errorMsg}
}
