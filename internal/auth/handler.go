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
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUsernameTaken) {
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

func (h *HTTPHandler) GetCurrentUser(ctx context.Context, request api.GetCurrentUserRequestObject) (api.GetCurrentUserResponseObject, error) {
	//TODO implement me
	panic("implement me")
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
		s := string(*request.Body.Email)
		email = &s
	}
	user, err := h.service.CreateUserByAdmin(ctx, request.Body.Username, request.Body.DisplayName, email, request.Body.Password, &principal)

	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			return api.CreateUser409JSONResponse(errorMsgResponse("User with that username already exists")), nil
		}
		if errors.Is(err, ErrInvalidDisplayName) || errors.Is(err, ErrInvalidUsername) {
			return api.CreateUser400JSONResponse(errorMsgResponse("Invalid input")), nil
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

func (h *HTTPHandler) Authenticator() Authenticator {
	return h.service
}

func toUserResponse(d *User) api.User {
	var email *openapi_types.Email

	if d.Email != nil { // Check if email exists
		e := openapi_types.Email(*d.Email)
		email = &e
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
