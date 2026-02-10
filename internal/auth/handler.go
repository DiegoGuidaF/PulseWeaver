package auth

import (
	"context"
	"log/slog"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

type HTTPHandler struct {
	service *Service
	logger  *slog.Logger
}

func (h *HTTPHandler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	rawToken, user, err := h.service.Login(ctx, string(request.Body.Email), request.Body.Password)
	if err != nil {
		return api.Login401JSONResponse(errorMsgResponse("Error authenticating")), nil
	}
	cookie := http.Cookie{
		Name:     SessionCookieName,
		Value:    rawToken,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600 * 24 * 30,
	}

	headers := api.Login200ResponseHeaders{SetCookie: cookie.String()}

	return api.Login200JSONResponse{Body: toUserResponse(user), Headers: headers}, nil
}

func (h *HTTPHandler) Logout(ctx context.Context, request api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	//TODO implement me
	panic("implement me")
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

	cookie := http.Cookie{
		Name:     SessionCookieName,
		Value:    rawToken,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600 * 24 * 30,
	}
	headers := api.Signup201ResponseHeaders{SetCookie: cookie.String()}

	return api.Signup201JSONResponse{Body: toUserResponse(user), Headers: headers}, nil

}

func NewHandler(service *Service, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{service: service, logger: logger}
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
