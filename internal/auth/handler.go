package auth

import (
	"context"
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
	rawToken, user, err := h.service.Login(ctx, string(request.Body.Email), request.Body.Password)
	if err != nil {
		return api.Login401JSONResponse(errorMsgResponse("Error authenticating")), nil
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

func (h *HTTPHandler) Signup(ctx context.Context, request api.SignupRequestObject) (api.SignupResponseObject, error) {
	rawToken, user, err := h.service.SignUp(ctx, request.Body.Name, string(request.Body.Email), request.Body.Password)

	if err != nil {
		return api.Signup409JSONResponse(errorMsgResponse("Error signing up")), nil
	}

	cookie := NewSessionCookie(rawToken, h.cookieConfig)
	headers := api.Signup201ResponseHeaders{SetCookie: cookie.String()}

	return api.Signup201JSONResponse{Body: toUserResponse(user), Headers: headers}, nil

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
	return api.User{
		ID:        d.ID.Int64(),
		Email:     openapi_types.Email(d.Email),
		CreatedAt: d.CreatedAt,
	}
}

func errorMsgResponse(errorMsg string) api.ErrorResponse {
	return api.ErrorResponse{Error: &errorMsg}
}
