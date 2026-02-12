package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_Login_HappyPath(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "AdminPass123!",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	is.True(len(res.Result().Cookies()) > 0)

	var user api.User
	err := json.NewDecoder(res.Body).Decode(&user)
	is.NoErr(err)
	is.Equal(user.Username, "admin")
}

func TestHandler_CreateUser_HappyPaths(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	t.Run("with_email", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"username":     "new_user_with_email",
			"display_name": "New User",
			"email":        "new_user@example.com",
			"password":     "Password123",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)

		is.Equal(res.Code, http.StatusCreated)

		var raw map[string]any
		err := json.NewDecoder(res.Body).Decode(&raw)
		is.NoErr(err)
		is.Equal(raw["username"], "new_user_with_email")
		is.Equal(raw["email"], "new_user@example.com")
	})

	t.Run("without_email", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"username":     "new_user_without_email",
			"display_name": "No Email User",
			"password":     "Password123",
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)

		is.Equal(res.Code, http.StatusCreated)

		var raw map[string]any
		err := json.NewDecoder(res.Body).Decode(&raw)
		is.NoErr(err)
		is.Equal(raw["username"], "new_user_without_email")

		_, hasEmail := raw["email"]
		is.True(!hasEmail)
	})
}
