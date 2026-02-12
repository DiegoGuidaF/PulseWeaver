package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/api"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/auth"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
	"github.com/matryer/is"
)

func setupAuthIntegrationServer(t *testing.T) http.Handler {
	t.Helper()

	conf := config.Conf{
		Server: config.ConfServer{
			Port:          2000,
			AdminPassword: "AdminPass123!",
		},
		DB: config.ConfDB{
			Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
			Debug: false,
		},
	}

	db, err := database.NewSQLite(conf.DB)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authRepo := auth.NewRepository(db.DB())
	authService := auth.NewService(authRepo, logger)
	if err := authService.BootstrapAdmin(context.Background(), conf.Server); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	authHandler := auth.NewHandler(authService, logger)

	deviceRepo := device.NewRepository(db.DB())
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewOpenApiHandler(deviceService, logger)

	return httpserver.NewServer(deviceHandler, authHandler, logger)
}

func loginAndGetCookie(t *testing.T, server http.Handler) *http.Cookie {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "AdminPass123!",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("login failed with status %d", res.Code)
	}

	cookies := res.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("missing session cookie")
	}

	return cookies[0]
}

func TestHandler_Login_HappyPath(t *testing.T) {
	is := is.New(t)
	server := setupAuthIntegrationServer(t)

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
	server := setupAuthIntegrationServer(t)
	adminCookie := loginAndGetCookie(t, server)

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
