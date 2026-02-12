package auth

import (
	"context"
	"fmt"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"github.com/matryer/is"
)

func setupAuthTestDB(t *testing.T) UserRepository {
	t.Helper()

	conf := config.ConfDB{
		Dsn:   fmt.Sprintf("file:%s?mode=memory&_loc=auto", t.Name()),
		Debug: false,
	}

	db, err := database.NewSQLite(conf)
	if err != nil {
		t.Fatalf("setup db: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	if err := db.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewRepository(db.DB())
}

func mustNewUser(t *testing.T, username, displayName string, email *string, role Role) *User {
	t.Helper()

	user, err := NewUser(username, displayName, email, "Password123", role, nil)
	if err != nil {
		t.Fatalf("new user: %v", err)
	}

	return user
}

func TestRepository_CreateUser_WithEmail(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t)
	ctx := context.Background()

	email := "john@example.com"
	created, err := repo.CreateUser(ctx, mustNewUser(t, "john_doe", "John Doe", &email, UserRole))
	is.NoErr(err)
	is.Equal(created.Username, "john_doe")
	is.Equal(*created.Email, email)
}

func TestRepository_CreateUser_WithoutEmail(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t)
	ctx := context.Background()

	created, err := repo.CreateUser(ctx, mustNewUser(t, "jane_doe", "Jane Doe", nil, UserRole))
	is.NoErr(err)
	is.True(created.Email == nil)
}

func TestRepository_CreateUser_DuplicateUsernameCaseVariant(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, mustNewUser(t, "john_doe", "John Doe", nil, UserRole))
	is.NoErr(err)

	_, err = repo.CreateUser(ctx, mustNewUser(t, "JOHN_DOE", "Johnny", nil, UserRole))
	is.True(err != nil)
	is.True(err == ErrUsernameTaken)
}

func TestRepository_CreateUser_DuplicateEmail(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t)
	ctx := context.Background()

	email := "duplicate@example.com"
	_, err := repo.CreateUser(ctx, mustNewUser(t, "user_a", "User A", &email, UserRole))
	is.NoErr(err)

	_, err = repo.CreateUser(ctx, mustNewUser(t, "user_b", "User B", &email, UserRole))
	is.True(err != nil)
	is.True(err == ErrEmailTaken)
}

func TestRepository_GetUserByUsername_CaseInsensitiveLookup(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t)
	ctx := context.Background()

	_, err := repo.CreateUser(ctx, mustNewUser(t, "alice_user", "Alice", nil, UserRole))
	is.NoErr(err)

	user, err := repo.GetUserByUsername(ctx, "ALICE_USER")
	is.NoErr(err)
	is.Equal(user.Username, "alice_user")
}

func TestRepository_SessionCreateAndRead_HappyPath(t *testing.T) {
	is := is.New(t)
	repo := setupAuthTestDB(t)
	ctx := context.Background()

	user, err := repo.CreateUser(ctx, mustNewUser(t, "session_user", "Session User", nil, UserRole))
	is.NoErr(err)

	session := NewSession(user.ID, "token-hash-1")
	createdSession, err := repo.CreateSession(ctx, session)
	is.NoErr(err)
	is.Equal(createdSession.UserId, user.ID)

	foundSession, err := repo.GetSessionWithRoleByTokenHash(ctx, "token-hash-1")
	is.NoErr(err)
	is.Equal(foundSession.ID, createdSession.ID)
	is.Equal(foundSession.UserId, user.ID)
	is.Equal(foundSession.UserRole, UserRole)
}
