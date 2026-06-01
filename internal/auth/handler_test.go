//go:build test

package auth_test

import (
	"context"
	"net/http"
	"testing"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/httpapi"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

func TestHandler_Login(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	// Use unauthenticated client: login produces its own session cookie.
	client := testutils.NewAPIClient(t, srv)

	resp, err := client.LoginWithResponse(ctx, httpapi.LoginJSONRequestBody{
		Username: "admin",
		Password: "AdminPass123!",
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(len(resp.HTTPResponse.Cookies()) > 0)
	is.Equal(resp.JSON200.Username, "admin")
}

func TestHandler_ListUsers_AdminCanList(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ListUsersWithResponse(ctx)
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusOK)
	is.True(len(*resp.JSON200) >= 1)
}

func TestHandler_ListUsers_RequiresAuth(t *testing.T) {
	// After PW-40 non-admin users can't obtain sessions; any unauthenticated request returns 401.
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)

	resp, err := testutils.NewAPIClient(t, srv).ListUsersWithResponse(ctx)
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusUnauthorized)
}

func TestHandler_UpdateMe_UpdatesDisplayName(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	displayName := httpapi.DisplayName("Updated Admin")
	resp, err := client.UpdateMeWithResponse(ctx, httpapi.UpdateMeJSONRequestBody{
		DisplayName: &displayName,
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.DisplayName, "Updated Admin")
	is.Equal(resp.JSON200.Username, auth.BootstrapAdminUsername)
	is.Equal(string(resp.JSON200.Email), auth.BootstrapAdminEmail)
}

func TestHandler_UpdateMe_ConflictOnDuplicateUsername(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	// Setup: create a user whose username we will attempt to steal.
	email := openapi_types.Email("taken_user@example.com")
	createResp, err := client.CreateUserWithResponse(ctx, httpapi.CreateUserJSONRequestBody{
		Username:    "taken_user",
		DisplayName: "Taken",
		Email:       &email,
	})
	is.NoErr(err)
	is.Equal(createResp.StatusCode(), http.StatusCreated)

	// Attempt to rename admin to the already-taken username.
	username := httpapi.Username("taken_user")
	resp, err := client.UpdateMeWithResponse(ctx, httpapi.UpdateMeJSONRequestBody{
		Username: &username,
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusConflict)
}

func TestHandler_ChangePassword_Success(t *testing.T) {
	// After PW-40 only admins can log in; test admin changing their own password.
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ChangePasswordWithResponse(ctx, httpapi.ChangePasswordJSONRequestBody{
		CurrentPassword: "AdminPass123!",
		Password:        "NewAdminPass456!",
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_ChangePassword_WrongCurrentPassword(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.ChangePasswordWithResponse(ctx, httpapi.ChangePasswordJSONRequestBody{
		CurrentPassword: "WrongPassword!",
		Password:        "NewPass456!",
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusBadRequest)
}

func TestHandler_PromoteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	userID := createUserAndGetID(t, ctx, client, "promote_target", "promote@example.com")

	resp, err := client.PromoteUserWithResponse(ctx, userID, httpapi.PromoteUserJSONRequestBody{
		Password: "NewAdminPass123!",
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.Role, httpapi.UserRoleAdmin)
}

func TestHandler_PromoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	adminID := testutils.AdminPrincipal(t, srv).UserID.Int64()

	resp, err := client.PromoteUserWithResponse(ctx, adminID, httpapi.PromoteUserJSONRequestBody{
		Password: "NewAdminPass123!",
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusForbidden)
}

func TestHandler_DemoteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	userID := createUserAndGetID(t, ctx, client, "demote_target", "demote@example.com")

	// Setup: promote first so there is something to demote.
	promoteResp, err := client.PromoteUserWithResponse(ctx, userID, httpapi.PromoteUserJSONRequestBody{
		Password: "NewAdminPass123!",
	})
	is.NoErr(err)
	is.Equal(promoteResp.StatusCode(), http.StatusOK)

	resp, err := client.DemoteUserWithResponse(ctx, userID)
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusOK)
	is.Equal(resp.JSON200.Role, httpapi.UserRoleUser)
}

func TestHandler_DemoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	adminID := testutils.AdminPrincipal(t, srv).UserID.Int64()

	resp, err := client.DemoteUserWithResponse(ctx, adminID)
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusForbidden)
}

func TestHandler_DeleteUser_AdminCanDelete(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	userID := createUserAndGetID(t, ctx, client, "delete_target", "delete_target@example.com")

	resp, err := client.DeleteUserWithResponse(ctx, userID)
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusNoContent)
}

func TestHandler_DeleteUser_SelfDeleteForbidden(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	adminID := testutils.AdminPrincipal(t, srv).UserID.Int64()

	resp, err := client.DeleteUserWithResponse(ctx, adminID)
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusForbidden)
}

func TestHandler_CreateUser_WithEmail(t *testing.T) {
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	email := openapi_types.Email("new_user@example.com")
	resp, err := client.CreateUserWithResponse(ctx, httpapi.CreateUserJSONRequestBody{
		Username:    "new_user_with_email",
		DisplayName: "New User",
		Email:       &email,
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusCreated)
	is.Equal(resp.JSON201.Username, "new_user_with_email")
	is.Equal(string(resp.JSON201.Email), "new_user@example.com")
}

func TestHandler_CreateUser_WithoutEmail_Succeeds(t *testing.T) {
	// Email is optional; omitting it must be accepted.
	is := is.New(t)
	ctx := t.Context()
	srv := testutils.SetupIntegrationServer(t)
	client := testutils.NewAdminAPIClient(t, srv)

	resp, err := client.CreateUserWithResponse(ctx, httpapi.CreateUserJSONRequestBody{
		Username:    "new_user_without_email",
		DisplayName: "No Email User",
	})
	is.NoErr(err)

	is.Equal(resp.StatusCode(), http.StatusCreated)
}

// createUserAndGetID creates a user-role account via the admin API and returns the user's numeric ID.
func createUserAndGetID(t *testing.T, ctx context.Context, client *httpapi.ClientWithResponses, username, email string) httpapi.ID {
	t.Helper()
	e := openapi_types.Email(email)
	resp, err := client.CreateUserWithResponse(ctx, httpapi.CreateUserJSONRequestBody{
		Username:    username,
		DisplayName: username,
		Email:       &e,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		t.Fatalf("create user: unexpected status %d", resp.StatusCode())
	}
	return resp.JSON201.Id
}
