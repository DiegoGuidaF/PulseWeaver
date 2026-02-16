package auth

import (
	"context"
	"errors"
	"log/slog"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type HTTPHandler struct {
	service      *Service
	logger       *slog.Logger
	cookieConfig CookieConfig
}

func (h *HTTPHandler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	rawToken, user, err := h.service.Login(ctx, request.Body.Username, request.Body.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserNotFound) {
			return api.Login401JSONResponse(errorMsgResponse("Invalid credentials")), nil
		}
		return api.Login500JSONResponse(errorMsgResponse("Login failure")), nil
	}

	cookie := NewSessionCookie(rawToken, h.cookieConfig)

	headers := api.Login200ResponseHeaders{SetCookie: cookie.String()}

	return api.Login200JSONResponse{Body: toUserResponse(user), Headers: headers}, nil
}

func (h *HTTPHandler) Logout(ctx context.Context, _ api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	principal, ok := PrincipalFromContext(ctx)
	if ok {
		err := h.service.RevokeSession(ctx, principal.SessionID)
		if err != nil {
			h.logger.Error("Failed to revoke session", "error", err)
		}
	}

	cookie := ExpireSessionCookie(h.cookieConfig)

	headers := api.Logout204ResponseHeaders{SetCookie: cookie.String()}

	return api.Logout204Response{Headers: headers}, nil
}

func (h *HTTPHandler) GetCurrentUser(ctx context.Context, _ api.GetCurrentUserRequestObject) (api.GetCurrentUserResponseObject, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return api.GetCurrentUser500JSONResponse(errorMsgResponse("Couldn't retrieve current principal from context")), nil
	}
	user, err := h.service.GetUserFromPrincipal(ctx, principal)
	if err != nil {
		var errorMsg string
		if errors.Is(err, ErrUserNotFound) {
			errorMsg = "User not found"
		} else {
			errorMsg = "Failed to retrieve current user"
		}
		return api.GetCurrentUser500JSONResponse(errorMsgResponse(errorMsg)), nil
	}

	return api.GetCurrentUser200JSONResponse(toUserResponse(user)), nil
}

func (h *HTTPHandler) CreateUser(ctx context.Context, request api.CreateUserRequestObject) (api.CreateUserResponseObject, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		h.logger.Error("failed to extract principal from request")
		return api.CreateUser403Response{}, nil
	}

	// Email is inside an openapi validator, we need to turn it into a valid string or nil
	var email *string
	if request.Body.Email != nil {
		email = new(string(*request.Body.Email))
	}
	user, err := h.service.CreateUserByAdmin(
		ctx,
		request.Body.Username,
		request.Body.DisplayName,
		email,
		request.Body.Password,
		principal,
	)

	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			return api.CreateUser409JSONResponse(errorMsgResponse("User with that username already exists")), nil
		}
		if errors.Is(err, ErrEmailTaken) {
			return api.CreateUser409JSONResponse(errorMsgResponse("User with that email already exists")), nil
		}
		if errors.Is(err, ErrInvalidDisplayName) || errors.Is(err, ErrInvalidUsername) || errors.Is(err, ErrInvalidPassword) {
			return api.CreateUser400JSONResponse(errorMsgResponse("Invalid input")), nil
		}
		if errors.Is(err, ErrAdminCredentialsRequired) {
			return api.CreateUser403Response{}, nil
		}
		return api.CreateUser500JSONResponse(errorMsgResponse("Failed to create user")), nil
	}

	return api.CreateUser201JSONResponse(toUserResponse(user)), nil

}

func NewHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	cfg := DefaultCookieConfig

	return &HTTPHandler{
		service:      service,
		logger:       logger,
		cookieConfig: cfg,
	}
}

func (h *HTTPHandler) UserAuthenticator() UserAuthenticator {
	return h.service
}

func toUserResponse(d *User) api.User {
	var email *openapi_types.Email

	if d.Email != nil { // Check if email exists
		email = new(openapi_types.Email(*d.Email))
	}

	return api.User{
		Id:          d.ID.Int64(),
		Username:    d.Username,
		DisplayName: d.DisplayName,
		Email:       email,
		CreatedAt:   d.CreatedAt,
	}
}

func errorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: &errorMsg}
}
