package auth

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"github.com/matryer/is"
)

func TestService_Login_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	// Create a test user
	user, err := NewUser("testuser", "Test User", nil, "Password123", UserRole, nil)
	is.NoErr(err)
	mockRepo.users[user.ID] = user
	mockRepo.usersByUsername[user.Username] = user
	mockRepo.userCount = 1

	token, returnedUser, err := service.Login(ctx, "testuser", "Password123")
	is.NoErr(err)
	is.True(token != "")
	is.True(returnedUser != nil)
	is.Equal(returnedUser.Username, "testuser")
}

func TestService_Login_InvalidUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	mockRepo.getUserByUsernameErr = ErrUsernameNotFound
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	token, user, err := service.Login(ctx, "nonexistent", "Password123")
	is.True(err != nil)
	is.Equal(err, ErrUsernameNotFound)
	is.Equal(token, "")
	is.True(user == nil)
}

func TestService_Login_InvalidPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	// Create a test user
	user, err := NewUser("testuser", "Test User", nil, "Password123", UserRole, nil)
	is.NoErr(err)
	mockRepo.users[user.ID] = user
	mockRepo.usersByUsername[user.Username] = user
	mockRepo.userCount = 1

	token, returnedUser, err := service.Login(ctx, "testuser", "WrongPassword")
	is.True(err != nil)
	is.Equal(err, ErrInvalidCredentials)
	is.Equal(token, "")
	is.True(returnedUser == nil)
}

func TestService_Login_RepositoryError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	testErr := errors.New("database error")
	mockRepo.getUserByUsernameErr = testErr
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	token, user, err := service.Login(ctx, "testuser", "Password123")
	is.True(err != nil)
	is.Equal(err, testErr)
	is.Equal(token, "")
	is.True(user == nil)
}

func TestService_CreateUserByAdmin_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	// Create admin user
	admin, err := NewUser("admin", "Admin", nil, "AdminPass123!", AdminRole, nil)
	is.NoErr(err)
	mockRepo.users[admin.ID] = admin
	mockRepo.usersByUsername[admin.Username] = admin
	mockRepo.userCount = 1

	principal := NewPrincipal(admin.ID, SessionID(1), AdminRole)

	createdUser, err := service.CreateUserByAdmin(ctx, "newuser", "New User", nil, "Password123", principal)
	is.NoErr(err)
	is.True(createdUser != nil)
	is.Equal(createdUser.Username, "newuser")
	is.Equal(createdUser.DisplayName, "New User")
	is.Equal(createdUser.Role, UserRole)
}

func TestService_CreateUserByAdmin_NonAdmin(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	// Create regular user
	regularUser, err := NewUser("regular", "Regular User", nil, "Password123", UserRole, nil)
	is.NoErr(err)
	mockRepo.users[regularUser.ID] = regularUser
	mockRepo.usersByUsername[regularUser.Username] = regularUser

	principal := NewPrincipal(regularUser.ID, SessionID(1), UserRole)

	createdUser, err := service.CreateUserByAdmin(ctx, "newuser", "New User", nil, "Password123", principal)
	is.True(err != nil)
	is.Equal(err, ErrAdminCredentialsRequired)
	is.True(createdUser == nil)
}

func TestService_CreateUserByAdmin_DuplicateUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	// Create admin user
	admin, err := NewUser("admin", "Admin", nil, "AdminPass123!", AdminRole, nil)
	is.NoErr(err)
	mockRepo.users[admin.ID] = admin
	mockRepo.usersByUsername[admin.Username] = admin
	mockRepo.userCount = 1

	principal := NewPrincipal(admin.ID, SessionID(1), AdminRole)

	// First user creation succeeds
	_, err = service.CreateUserByAdmin(ctx, "newuser", "New User", nil, "Password123", principal)
	is.NoErr(err)

	// Second user with same username should fail
	mockRepo.createUserErr = ErrUsernameTaken
	createdUser, err := service.CreateUserByAdmin(ctx, "newuser", "New User", nil, "Password123", principal)
	is.True(err != nil)
	is.Equal(err, ErrUsernameTaken)
	is.True(createdUser == nil)
}

func TestService_BootstrapAdmin_CreatesAdminWhenNoUsers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	mockRepo.userCount = 0
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	conf := config.ConfServer{
		AdminPassword: "AdminPass123!",
	}

	err := service.BootstrapAdmin(ctx, conf)
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

	// Verify admin was created
	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, AdminRole)
	is.Equal(admin.Username, "admin")
}

func TestService_BootstrapAdmin_SkipsWhenUsersExist(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	// Create existing user
	existingUser, err := NewUser("existing", "Existing User", nil, "Password123", UserRole, nil)
	is.NoErr(err)
	mockRepo.users[existingUser.ID] = existingUser
	mockRepo.usersByUsername[existingUser.Username] = existingUser
	mockRepo.userCount = 1

	conf := config.ConfServer{
		AdminPassword: "AdminPass123!",
	}

	err = service.BootstrapAdmin(ctx, conf)
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1) // Should still be 1, not 2

	// Verify admin was NOT created
	_, ok := mockRepo.usersByUsername["admin"]
	is.True(!ok)
}

func TestService_BootstrapAdmin_GeneratesPasswordWhenNotProvided(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	mockRepo.userCount = 0
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	conf := config.ConfServer{
		AdminPassword: "", // Empty password
	}

	err := service.BootstrapAdmin(ctx, conf)
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

	// Verify admin was created
	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, AdminRole)
}

func TestService_BootstrapAdmin_UsesProvidedPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockUserRepository()
	mockRepo.userCount = 0
	logger := newTestLogger()
	service := NewService(mockRepo, logger)

	conf := config.ConfServer{
		AdminPassword: "CustomAdminPass123!",
	}

	err := service.BootstrapAdmin(ctx, conf)
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

	// Verify admin was created with the provided password
	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, AdminRole)

	// Verify password works
	token, _, err := service.Login(ctx, "admin", "CustomAdminPass123!")
	is.NoErr(err)
	is.True(token != "")
}

// mockUserRepository is a hand-rolled mock implementation of UserRepository
type mockUserRepository struct {
	users                map[UserID]*User
	usersByUsername      map[string]*User // lowercase username -> user
	sessions             map[SessionID]*SessionWithUser
	sessionsByToken      map[string]*SessionWithUser // tokenHash -> session
	userCount            int
	getUserByUsernameErr error
	createUserErr        error
	createSessionErr     error
	getSessionErr        error
	countUsersErr        error
	runInTxFn            func(UserRepository) error
}

// Ensure mockUserRepository implements UserRepository interface
var _ UserRepository = (*mockUserRepository)(nil)

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:           make(map[UserID]*User),
		usersByUsername: make(map[string]*User),
		sessions:        make(map[SessionID]*SessionWithUser),
		sessionsByToken: make(map[string]*SessionWithUser),
		userCount:       0,
	}
}

func (m *mockUserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if m.getUserByUsernameErr != nil {
		return nil, m.getUserByUsernameErr
	}
	user, ok := m.usersByUsername[username]
	if !ok {
		return nil, ErrUsernameNotFound
	}
	return user, nil
}

func (m *mockUserRepository) CreateUser(ctx context.Context, user *User) (*User, error) {
	if m.createUserErr != nil {
		return nil, m.createUserErr
	}
	user.ID = UserID(len(m.users) + 1)
	m.users[user.ID] = user
	m.usersByUsername[user.Username] = user
	m.userCount++
	return user, nil
}

func (m *mockUserRepository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	if m.createSessionErr != nil {
		return nil, m.createSessionErr
	}
	session.ID = SessionID(len(m.sessions) + 1)
	sessionWithUser := &SessionWithUser{
		Session:  *session,
		UserRole: m.users[session.UserId].Role,
	}
	m.sessions[session.ID] = sessionWithUser
	m.sessionsByToken[session.TokenHash] = sessionWithUser
	return session, nil
}

func (m *mockUserRepository) GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	session, ok := m.sessionsByToken[tokenHash]
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return session, nil
}

func (m *mockUserRepository) CountUsers(ctx context.Context) (int, error) {
	if m.countUsersErr != nil {
		return 0, m.countUsersErr
	}
	return m.userCount, nil
}

func (m *mockUserRepository) RevokeSessionById(ctx context.Context, id SessionID) error {
	session, ok := m.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	delete(m.sessionsByToken, session.TokenHash)
	delete(m.sessions, id)
	return nil
}

func (m *mockUserRepository) RunInTx(ctx context.Context, fn func(UserRepository) error) error {
	if m.runInTxFn != nil {
		return m.runInTxFn(m)
	}
	return fn(m)
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}
