//go:build test

package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/ids"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testdb"
	"github.com/matryer/is"
)

func setupAuthTestDB(t *testing.T, shouldBootstrapAdmin bool) *auth.Repository {
	t.Helper()

	db, cleanup := testdb.Setup(t)
	t.Cleanup(cleanup)

	repository := auth.NewRepository(db.DB())
	if shouldBootstrapAdmin {
		bootstrapUser, err := auth.NewBootstrappedAdmin("apassword")
		if err != nil {
			t.Fatal(err)
		}
		_, err = repository.CreateUser(t.Context(), new(bootstrapUser))
		if err != nil {
			return nil
		}
	}
	return repository
}

func mustNewUser(t *testing.T, username, displayName string, email string, role auth.Role) *auth.User {
	t.Helper()

	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}

	var user auth.User
	var err error
	if role == auth.AdminRole {
		user, err = auth.NewAdminUser(username, displayName, "Password123", emailPtr, new(ids.UserID(1)), true)
	} else {
		user, err = auth.NewUserAccount(username, displayName, emailPtr, new(ids.UserID(1)))
	}
	if err != nil {
		t.Fatalf("new user: %v", err)
	}

	return &user
}

func TestRepository_CreateUser_WithEmail(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	email := "john@example.com"
	created, err := repo.CreateUser(ctx, mustNewUser(t, "john_doe", "John Doe", email, auth.UserRole))
	is.NoErr(err)
	is.Equal(created.Username, "john_doe")
	is.Equal(*created.Email, email)
}

func TestRepository_CreateUser_WithoutEmail(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	created, err := repo.CreateUser(ctx, mustNewUser(t, "jane_doe", "Jane Doe", "", auth.UserRole))
	is.NoErr(err)
	is.Equal(created.Email, (*string)(nil))
}

func TestRepository_CreateUser_DuplicateUsernameCaseVariant(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, mustNewUser(t, "john_doe", "John Doe", "", auth.UserRole))
	is.NoErr(err)

	_, err = repo.CreateUser(ctx, mustNewUser(t, "JOHN_DOE", "Johnny", "", auth.UserRole))
	is.True(err != nil)
	is.True(errors.Is(err, auth.ErrUsernameTaken))
}

func TestRepository_CreateUser_DuplicateEmail(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	email := "duplicate@example.com"
	_, err := repo.CreateUser(ctx, mustNewUser(t, "user_a", "User A", email, auth.UserRole))
	is.NoErr(err)

	_, err = repo.CreateUser(ctx, mustNewUser(t, "user_b", "User B", email, auth.UserRole))
	is.True(err != nil)
	is.True(errors.Is(err, auth.ErrEmailTaken))
}

func TestRepository_GetUserByUsername_CaseInsensitiveLookup(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, mustNewUser(t, "alice_user", "Alice", "", auth.UserRole))
	is.NoErr(err)

	user, err := repo.GetUserByUsername(ctx, "ALICE_USER")
	is.NoErr(err)
	is.Equal(user.Username, "alice_user")
}

func TestRepository_SessionCreateAndRead(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, mustNewUser(t, "session_user", "Session User", "", auth.UserRole))
	is.NoErr(err)

	createdSession, err := repo.CreateSession(ctx, new(auth.NewSession(user.ID, "token-hash-1")))
	is.NoErr(err)
	is.Equal(createdSession.UserID, user.ID)

	foundSession, err := repo.GetSessionWithRoleByTokenHash(ctx, "token-hash-1")
	is.NoErr(err)
	is.Equal(foundSession.ID, createdSession.ID)
	is.Equal(foundSession.UserID, user.ID)
	is.Equal(foundSession.UserRole, auth.UserRole)
}

func TestRepository_FindBootstrappedAdmin_Empty(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, false)
	ctx := context.Background()

	admin, err := repo.FindBootstrappedAdmin(ctx)
	is.Equal(err, auth.ErrUserNotFound)
	is.Equal(admin, nil)
}

func TestRepository_FindBootstrappedAdmin_ExcludesUsersWithCreatedBySet(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, false)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, mustNewUser(t, "regular_user", "Regular", "", auth.UserRole))
	is.NoErr(err)
	_, err = repo.CreateUser(ctx, mustNewUser(t, "admin_user", "Admin", "", auth.AdminRole))
	is.NoErr(err)
	_, err = repo.CreateUser(ctx, mustNewUser(t, "superadmin_user", "SuperAdmin", "", auth.SuperAdminRole))
	is.NoErr(err)

	_, err = repo.FindBootstrappedAdmin(ctx)
	is.Equal(err, auth.ErrUserNotFound)
}

func TestRepository_GetAllUsers_ReturnsInsertedUsers(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, mustNewUser(t, "user_alpha", "User Alpha", "", auth.UserRole))
	is.NoErr(err)
	_, err = repo.CreateUser(ctx, mustNewUser(t, "user_beta", "User Beta", "", auth.AdminRole))
	is.NoErr(err)

	users, err := repo.GetAllUsers(ctx)
	is.NoErr(err)
	// We need to count bootstrapped admin too
	is.Equal(len(users), 3)
}

func TestRepository_UpdateUser_UpdatesFields(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	created, err := repo.CreateUser(ctx, mustNewUser(t, "update_me", "Old Name", "", auth.UserRole))
	is.NoErr(err)

	created.DisplayName = "New Name"
	updated, err := repo.UpdateUser(ctx, created)
	is.NoErr(err)
	is.Equal(updated.DisplayName, "New Name")

	fetched, err := repo.GetUserByID(ctx, created.ID)
	is.NoErr(err)
	is.Equal(fetched.DisplayName, "New Name")
}

func TestRepository_UpdatePasswordHash(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, mustNewUser(t, "pw_user", "PW User", "", auth.AdminRole))
	is.NoErr(err)

	newHash := []byte("new-bcrypt-hash")
	err = repo.UpdatePasswordHash(ctx, user.ID, newHash, false)
	is.NoErr(err)

	fetched, err := repo.GetUserByID(ctx, user.ID)
	is.NoErr(err)
	is.Equal(fetched.PasswordHash, newHash)
}

func TestRepository_SoftDeleteUser(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, mustNewUser(t, "delete_me", "Delete Me", "", auth.UserRole))
	is.NoErr(err)

	err = repo.SoftDeleteUser(ctx, user.ID)
	is.NoErr(err)

	_, err = repo.GetUserByID(ctx, user.ID)
	is.True(errors.Is(err, auth.ErrUserNotFound))

	_, err = repo.GetUserByUsername(ctx, user.Username)
	is.True(errors.Is(err, auth.ErrUserNotFound))
}

func TestRepository_RevokeAllUserSessions(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, mustNewUser(t, "multi_session", "Multi Session", "", auth.UserRole))
	is.NoErr(err)

	_, err = repo.CreateSession(ctx, new(auth.NewSession(user.ID, "hash-a")))
	is.NoErr(err)
	_, err = repo.CreateSession(ctx, new(auth.NewSession(user.ID, "hash-b")))
	is.NoErr(err)

	err = repo.RevokeAllUserSessions(ctx, user.ID)
	is.NoErr(err)

	_, err = repo.GetSessionWithRoleByTokenHash(ctx, "hash-a")
	is.True(err != nil)
	_, err = repo.GetSessionWithRoleByTokenHash(ctx, "hash-b")
	is.True(err != nil)
}

func TestRepository_RevokeAllUserSessionsExcept(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t, true)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, mustNewUser(t, "except_user", "Except User", "", auth.UserRole))
	is.NoErr(err)

	kept, err := repo.CreateSession(ctx, new(auth.NewSession(user.ID, "hash-keep")))
	is.NoErr(err)
	_, err = repo.CreateSession(ctx, new(auth.NewSession(user.ID, "hash-revoke")))
	is.NoErr(err)

	err = repo.RevokeAllUserSessionsExcept(ctx, user.ID, kept.ID)
	is.NoErr(err)

	_, err = repo.GetSessionWithRoleByTokenHash(ctx, "hash-keep")
	is.NoErr(err)

	_, err = repo.GetSessionWithRoleByTokenHash(ctx, "hash-revoke")
	is.True(err != nil)
}
