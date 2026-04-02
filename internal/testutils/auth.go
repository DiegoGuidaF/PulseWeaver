//go:build test

package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/app"
	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
)

// AdminPrincipal returns an admin Principal for the bootstrap admin user, for use in direct service calls.
func AdminPrincipal(t *testing.T, a *app.App) *auth.Principal {
	t.Helper()
	_, user, err := a.AuthService.Login(context.Background(), "admin", TestAdminPassword)
	if err != nil {
		t.Fatalf("AdminPrincipal: login failed: %v", err)
	}
	return auth.NewPrincipal(user.ID, auth.SessionID(0), user.Role)
}

// LoginCookie performs a login request and returns the session cookie.
func LoginCookie(t *testing.T, server http.Handler, username, password string) *http.Cookie {
	t.Helper()

	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed with status %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie from login")
	}

	return cookies[0]
}
