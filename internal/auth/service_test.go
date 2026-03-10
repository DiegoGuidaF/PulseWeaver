//go:build test

package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/WallyDex/internal/config"
	"github.com/matryer/is"
)

func TestService_Login_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create a test user
	user, err := NewUser("testuser", "Test User", "", "Password123", UserRole, nil)
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

	mockRepo := newMockRepository()
	mockRepo.getUserByUsernameErr = ErrUserNotFound
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	token, user, err := service.Login(ctx, "nonexistent", "Password123")
	is.True(err != nil)
	is.Equal(err, ErrUserNotFound)
	is.Equal(token, "")
	is.True(user == nil)
}

func TestService_Login_InvalidPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create a test user
	user, err := NewUser("testuser", "Test User", "", "Password123", UserRole, nil)
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

	mockRepo := newMockRepository()
	testErr := errors.New("database error")
	mockRepo.getUserByUsernameErr = testErr
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	token, user, err := service.Login(ctx, "testuser", "Password123")
	is.True(err != nil)
	is.Equal(err, testErr)
	is.Equal(token, "")
	is.True(user == nil)
}

func TestService_GetUserById_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create admin user
	admin, err := NewUser("admin", "Admin", "", "AdminPass123!", AdminRole, nil)
	is.NoErr(err)
	mockRepo.users[admin.ID] = admin
	mockRepo.usersByUsername[admin.Username] = admin
	mockRepo.userCount = 1

	principal := NewPrincipal(admin.ID, SessionID(1), AdminRole)

	currentUser, err := service.GetUserFromPrincipal(ctx, principal)
	is.NoErr(err)
	is.Equal(currentUser, admin)
}

func TestService_GetUserById_RepositoryError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	testErr := errors.New("database error")
	mockRepo.getUserByIDErr = testErr
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	principal := NewPrincipal(UserID(1), SessionID(1), AdminRole)

	currentUser, err := service.GetUserFromPrincipal(ctx, principal)
	is.True(err != nil)
	is.Equal(err, testErr)
	is.True(currentUser == nil)
}

func TestService_CreateUserByAdmin_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create admin user
	admin, err := NewUser("admin", "Admin", "", "AdminPass123!", AdminRole, nil)
	is.NoErr(err)
	mockRepo.users[admin.ID] = admin
	mockRepo.usersByUsername[admin.Username] = admin
	mockRepo.userCount = 1

	principal := NewPrincipal(admin.ID, SessionID(1), AdminRole)

	createdUser, err := service.CreateUserByAdmin(ctx, "newuser", "New User", "", "Password123", principal)
	is.NoErr(err)
	is.True(createdUser != nil)
	is.Equal(createdUser.Username, "newuser")
	is.Equal(createdUser.DisplayName, "New User")
	is.Equal(createdUser.Role, UserRole)
}

func TestService_CreateUserByAdmin_NonAdmin(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create regular user
	regularUser, err := NewUser("regular", "Regular User", "", "Password123", UserRole, nil)
	is.NoErr(err)
	mockRepo.users[regularUser.ID] = regularUser
	mockRepo.usersByUsername[regularUser.Username] = regularUser

	principal := NewPrincipal(regularUser.ID, SessionID(1), UserRole)

	createdUser, err := service.CreateUserByAdmin(ctx, "newuser", "New User", "", "Password123", principal)
	is.True(err != nil)
	is.Equal(err, ErrAdminCredentialsRequired)
	is.True(createdUser == nil)
}

func TestService_CreateUserByAdmin_DuplicateUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create admin user
	admin, err := NewUser("admin", "Admin", "", "AdminPass123!", AdminRole, nil)
	is.NoErr(err)
	mockRepo.users[admin.ID] = admin
	mockRepo.usersByUsername[admin.Username] = admin
	mockRepo.userCount = 1

	principal := NewPrincipal(admin.ID, SessionID(1), AdminRole)

	// First user creation succeeds
	_, err = service.CreateUserByAdmin(ctx, "newuser", "New User", "", "Password123", principal)
	is.NoErr(err)

	// Second user with same username should fail
	mockRepo.createUserErr = ErrUsernameTaken
	createdUser, err := service.CreateUserByAdmin(ctx, "newuser", "New User", "", "Password123", principal)
	is.True(err != nil)
	is.Equal(err, ErrUsernameTaken)
	is.True(createdUser == nil)
}

func TestService_BootstrapAdmin_CreatesAdminWhenNoUsers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.userCount = 0
	mockRepo.adminCount = 0
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

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

func TestService_BootstrapAdmin_CreatesAdminWhenNoAdminsExist(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	// Create existing user
	existingUser, err := NewUser("existing", "Existing User", "", "Password123", UserRole, nil)
	is.NoErr(err)
	mockRepo.users[existingUser.ID] = existingUser
	mockRepo.usersByUsername[existingUser.Username] = existingUser
	mockRepo.userCount = 1
	mockRepo.adminCount = 0

	conf := config.ConfServer{
		AdminPassword: "AdminPass123!",
	}

	err = service.BootstrapAdmin(ctx, conf)
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 2)

	// Verify admin was created
	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, AdminRole)
}

func TestService_BootstrapAdmin_SkipsWhenAdminExists(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	existingAdmin, err := NewUser("existing_admin", "Existing Admin", "", "Password123", AdminRole, nil)
	is.NoErr(err)
	mockRepo.users[UserID(1)] = existingAdmin
	mockRepo.usersByUsername[existingAdmin.Username] = existingAdmin
	mockRepo.userCount = 1
	mockRepo.adminCount = 1

	conf := config.ConfServer{
		AdminPassword: "AdminPass123!",
	}

	err = service.BootstrapAdmin(ctx, conf)
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)
	is.Equal(mockRepo.adminCount, 1)
}

func TestService_BootstrapAdmin_UsesProvidedPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.userCount = 0
	mockRepo.adminCount = 0
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

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

// mockRepository is a hand-rolled mock implementation of UserRepository
type mockRepository struct {
	users                map[UserID]*User
	usersByUsername      map[string]*User // lowercase username -> user
	sessions             map[SessionID]*SessionWithUser
	sessionsByToken      map[string]*SessionWithUser // tokenHash -> session
	userCount            int
	adminCount           int
	getUserByUsernameErr error
	getUserByIDErr       error
	createUserErr        error
	createSessionErr     error
	getSessionErr        error
	countUsersErr        error
	countAdminUsersErr   error
	runInTxFn            func(repository) error
}

// Ensure mockRepository implements UserRepository interface
var _ repository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:           make(map[UserID]*User),
		usersByUsername: make(map[string]*User),
		sessions:        make(map[SessionID]*SessionWithUser),
		sessionsByToken: make(map[string]*SessionWithUser),
		userCount:       0,
	}
}

func (m *mockRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	if m.getUserByUsernameErr != nil {
		return nil, m.getUserByUsernameErr
	}
	user, ok := m.usersByUsername[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) GetUserByID(ctx context.Context, userID UserID) (*User, error) {
	if m.getUserByIDErr != nil {
		return nil, m.getUserByIDErr
	}
	user, ok := m.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) CreateUser(ctx context.Context, user *User) (*User, error) {
	if m.createUserErr != nil {
		return nil, m.createUserErr
	}
	user.ID = UserID(len(m.users) + 1)
	m.users[user.ID] = user
	m.usersByUsername[user.Username] = user
	m.userCount++
	if user.Role == AdminRole {
		m.adminCount++
	}
	return user, nil
}

func (m *mockRepository) CreateSession(ctx context.Context, session *Session) (*Session, error) {
	if m.createSessionErr != nil {
		return nil, m.createSessionErr
	}
	session.ID = SessionID(len(m.sessions) + 1)
	sessionWithUser := &SessionWithUser{
		Session:  *session,
		UserRole: m.users[session.UserID].Role,
	}
	m.sessions[session.ID] = sessionWithUser
	m.sessionsByToken[session.TokenHash] = sessionWithUser
	return session, nil
}

func (m *mockRepository) GetSessionWithRoleByTokenHash(ctx context.Context, tokenHash string) (*SessionWithUser, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	session, ok := m.sessionsByToken[tokenHash]
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return session, nil
}

func (m *mockRepository) CountUsers(ctx context.Context) (int, error) {
	if m.countUsersErr != nil {
		return 0, m.countUsersErr
	}
	return m.userCount, nil
}

func (m *mockRepository) CountAdminUsers(ctx context.Context) (int, error) {
	if m.countAdminUsersErr != nil {
		return 0, m.countAdminUsersErr
	}
	return m.adminCount, nil
}

func (m *mockRepository) GetAllUsers(ctx context.Context) ([]User, error) {
	users := make([]User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, nil
}

func (m *mockRepository) UpdateUser(ctx context.Context, user *User) (*User, error) {
	existing, ok := m.users[user.ID]
	if !ok {
		return nil, ErrUserNotFound
	}

	if existing.Role == AdminRole && user.Role != AdminRole {
		m.adminCount--
	}
	if existing.Role != AdminRole && user.Role == AdminRole {
		m.adminCount++
	}

	if existing.Username != user.Username {
		delete(m.usersByUsername, existing.Username)
		m.usersByUsername[user.Username] = user
	}

	*existing = *user
	return existing, nil
}

func (m *mockRepository) UpdatePasswordHash(ctx context.Context, userID UserID, newHash []byte) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.PasswordHash = newHash
	user.MustChangePassword = false
	return nil
}

func (m *mockRepository) SoftDeleteUser(ctx context.Context, userID UserID) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}

	if user.Role == AdminRole {
		m.adminCount--
	}
	m.userCount--
	delete(m.usersByUsername, user.Username)
	delete(m.users, userID)
	return nil
}

func (m *mockRepository) RevokeSessionByID(ctx context.Context, id SessionID) error {
	session, ok := m.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	delete(m.sessionsByToken, session.TokenHash)
	delete(m.sessions, id)
	return nil
}

func (m *mockRepository) RevokeAllUserSessions(ctx context.Context, userID UserID) error {
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessionsByToken, session.TokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockRepository) RevokeAllUserSessionsExcept(ctx context.Context, userID UserID, exceptSessionID SessionID) error {
	for id, session := range m.sessions {
		if session.UserID == userID && id != exceptSessionID {
			delete(m.sessionsByToken, session.TokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockRepository) RunInTx(ctx context.Context, fn func(repository) error) error {
	if m.runInTxFn != nil {
		return m.runInTxFn(m)
	}
	return fn(m)
}
