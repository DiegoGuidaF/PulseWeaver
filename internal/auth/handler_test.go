//go:build test

package auth_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_Login(t *testing.T) {
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

	var user httpapi.User
	err := json.NewDecoder(res.Body).Decode(&user)
	is.NoErr(err)
	is.Equal(user.Username, "admin")
}

func TestHandler_ListUsers_AdminCanList(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var users []map[string]any
	err := json.NewDecoder(res.Body).Decode(&users)
	is.NoErr(err)
	is.True(len(users) >= 1)
}

func TestHandler_ListUsers_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	// Create a regular user
	createBody, _ := json.Marshal(map[string]string{
		"username":     "regular_lister",
		"display_name": "Regular",
		"email":        "regular_lister@example.com",
		"password":     "Password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	is.Equal(res.Code, http.StatusCreated)

	regularCookie := testutils.LoginCookie(t, server, "regular_lister", "Password123")

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req2.AddCookie(regularCookie)
	res2 := httptest.NewRecorder()
	server.ServeHTTP(res2, req2)

	is.Equal(res2.Code, http.StatusForbidden)
}

func TestHandler_UpdateMe_UpdatesDisplayName(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, auth.BootstrapAdminUsername, "AdminPass123!")

	body, _ := json.Marshal(map[string]string{
		"display_name": "Updated Admin",
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)

	var user httpapi.User
	err := json.NewDecoder(res.Body).Decode(&user)
	is.NoErr(err)
	is.Equal(user.DisplayName, "Updated Admin")
	is.Equal(user.Username, auth.BootstrapAdminUsername)
	is.Equal(string(user.Email), auth.BootstrapAdminEmail)
}

func TestHandler_UpdateMe_ConflictOnDuplicateUsername(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	createBody, _ := json.Marshal(map[string]string{
		"username":     "taken_user",
		"display_name": "Taken",
		"email":        "taken_user@example.com",
		"password":     "Password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	is.Equal(res.Code, http.StatusCreated)

	body, _ := json.Marshal(map[string]string{"username": "taken_user"})
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/users/me", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(adminCookie)
	res2 := httptest.NewRecorder()
	server.ServeHTTP(res2, req2)

	is.Equal(res2.Code, http.StatusConflict)
}

func TestHandler_ChangePassword_Success(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	// Create a regular user
	createBody, _ := json.Marshal(map[string]string{
		"username":     "pw_change_user",
		"display_name": "PW Change",
		"email":        "pw_change@example.com",
		"password":     "OldPass123!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	is.Equal(res.Code, http.StatusCreated)

	userCookie := testutils.LoginCookie(t, server, "pw_change_user", "OldPass123!")

	body, _ := json.Marshal(map[string]string{
		"current_password": "OldPass123!",
		"password":         "NewPass456!",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.AddCookie(userCookie)
	res2 := httptest.NewRecorder()
	server.ServeHTTP(res2, req2)

	is.Equal(res2.Code, http.StatusNoContent)
}

func TestHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	body, _ := json.Marshal(map[string]string{
		"current_password": "WrongPassword!",
		"password":         "NewPass456!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusBadRequest)
}

func TestHandler_PromoteUser_Success(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	userID := createUserAndGetID(t, server, adminCookie, "promote_target", "promote@example.com")

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%.0f/promote", userID), nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	var user map[string]any
	err := json.NewDecoder(res.Body).Decode(&user)
	is.NoErr(err)
	is.Equal(user["role"], "admin")
}

func TestHandler_PromoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	adminID := getSelfID(t, server, adminCookie)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%.0f/promote", adminID), nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusForbidden)
}

func TestHandler_DemoteUser_Success(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	userID := createUserAndGetID(t, server, adminCookie, "demote_target", "demote@example.com")
	promoteReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%.0f/promote", userID), nil)
	promoteReq.AddCookie(adminCookie)
	promoteRes := httptest.NewRecorder()
	server.ServeHTTP(promoteRes, promoteReq)
	is.Equal(promoteRes.Code, http.StatusOK)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%.0f/demote", userID), nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusOK)
	var user map[string]any
	err := json.NewDecoder(res.Body).Decode(&user)
	is.NoErr(err)
	is.Equal(user["role"], "user")
}

func TestHandler_DemoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	adminID := getSelfID(t, server, adminCookie)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/admin/users/%.0f/demote", adminID), nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusForbidden)
}

func TestHandler_DeleteUser_AdminCanDelete(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	userID := createUserAndGetID(t, server, adminCookie, "delete_target", "delete_target@example.com")

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%.0f", userID), nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusNoContent)
}

func TestHandler_DeleteUser_SelfDeleteForbidden(t *testing.T) {
	is := is.New(t)
	testServer := testutils.SetupIntegrationServer(t)
	server := testServer.HTTPServer
	adminCookie := testutils.LoginCookie(t, server, "admin", "AdminPass123!")

	adminID := getSelfID(t, server, adminCookie)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/admin/users/%.0f", adminID), nil)
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)

	is.Equal(res.Code, http.StatusForbidden)
}

func TestHandler_CreateUser(t *testing.T) {
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

	t.Run("without_email_returns_400", func(t *testing.T) {
		// Email is now required; omitting it must be rejected by the generated request validator.
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

		is.Equal(res.Code, http.StatusBadRequest)
	})
}

// getSelfID returns the numeric ID of the currently authenticated user.
func getSelfID(t *testing.T, server http.Handler, cookie *http.Cookie) float64 {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("get self: unexpected status %d", res.Code)
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode get self response: %v", err)
	}
	return raw["id"].(float64)
}

// createUserAndGetID creates a regular user via the admin API and returns the user's numeric ID.
func createUserAndGetID(t *testing.T, server http.Handler, adminCookie *http.Cookie, username, email string) float64 {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username":     username,
		"display_name": username,
		"email":        email,
		"password":     "Password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create user: unexpected status %d", res.Code)
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode create user response: %v", err)
	}
	return raw["id"].(float64)
}
