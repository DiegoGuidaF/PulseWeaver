//go:build test

package auth_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/auth"
	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/DiegoGuidaF/PulseWeaver/internal/testutils"
	"github.com/matryer/is"
)

// givenUser inserts a user into the mock repo with the given ID and role.
// No password hash is set; use auth.NewUser directly when password verification is needed.
func givenUser(repo *mockRepository, id auth.UserID, username string, role auth.Role) *auth.User {
	user := &auth.User{
		ID:          id,
		Username:    username,
		DisplayName: username,
		Role:        role,
	}
	repo.users[id] = user
	repo.usersByUsername[username] = user
	repo.userCount++
	if role == auth.AdminRole {
		repo.adminCount++
	}
	return user
}

// principalFor returns a Principal for the given user. The auth.SessionID is a fixed
// dummy value sufficient for tests that do not exercise session-specific logic.
func principalFor(user *auth.User) *auth.Principal {
	return auth.NewPrincipal(user.ID, auth.SessionID(1), user.Role)
}

// givenSessionFor inserts a session into the mock repo for the given user and
// returns a Principal whose auth.SessionID matches. Use this when the test must
// distinguish sessions (e.g. ChangePassword revokes others but keeps current).
func givenSessionFor(repo *mockRepository, user *auth.User) *auth.Principal {
	id := auth.SessionID(len(repo.sessions) + 1)
	tokenHash := user.Username + "-token"
	session := &auth.Session{ID: id, UserID: user.ID, TokenHash: tokenHash}
	swu := &auth.SessionWithUser{Session: *session, UserRole: user.Role}
	repo.sessions[id] = swu
	repo.sessionsByToken[tokenHash] = swu
	return auth.NewPrincipal(user.ID, id, user.Role)
}

func TestService_Login_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	u, err := auth.NewUser("testuser", "Test User", "", "Password123", auth.UserRole, nil)
	is.NoErr(err)
	u.ID = auth.UserID(1)
	user := &u
	mockRepo.users[user.ID] = user
	mockRepo.usersByUsername[user.Username] = user
	mockRepo.userCount = 1

	token, returnedUser, err := service.Login(ctx, "testuser", "Password123")
	is.NoErr(err)
	is.True(token != "")
	is.True(returnedUser != nil)
	is.Equal(returnedUser.Username, "testuser")
}

func newService(mockRepo *mockRepository) *auth.Service {
	return auth.NewService(mockRepo, testutils.NoopTransactor{}, slog.New(slog.DiscardHandler))
}

func TestService_Login_InvalidUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	mockRepo.getUserByUsernameErr = auth.ErrUserNotFound
	service := newService(mockRepo)

	token, user, err := service.Login(ctx, "nonexistent", "Password123")
	is.True(err != nil)
	is.Equal(err, auth.ErrUserNotFound)
	is.Equal(token, "")
	is.True(user == nil)
}

func TestService_Login_InvalidPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	u, err := auth.NewUser("testuser", "Test User", "", "Password123", auth.UserRole, nil)
	is.NoErr(err)
	u.ID = auth.UserID(1)
	user := &u
	mockRepo.users[user.ID] = user
	mockRepo.usersByUsername[user.Username] = user
	mockRepo.userCount = 1

	token, returnedUser, err := service.Login(ctx, "testuser", "WrongPassword")
	is.True(err != nil)
	is.Equal(err, auth.ErrInvalidCredentials)
	is.Equal(token, "")
	is.True(returnedUser == nil)
}

func TestService_Login_RepositoryError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	testErr := errors.New("database error")
	mockRepo.getUserByUsernameErr = testErr
	service := newService(mockRepo)

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
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)
	principal := principalFor(admin)

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
	service := newService(mockRepo)

	principal := principalFor(givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole))
	mockRepo.getUserByIDErr = testErr // set after givenUser to avoid affecting setup

	currentUser, err := service.GetUserFromPrincipal(ctx, principal)
	is.True(err != nil)
	is.Equal(err, testErr)
	is.True(currentUser == nil)
}

func TestService_CreateUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)

	createdUser, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(admin))
	is.NoErr(err)
	is.True(createdUser != nil)
	is.Equal(createdUser.Username, "newuser")
	is.Equal(createdUser.DisplayName, "New User")
	is.Equal(createdUser.Role, auth.UserRole)
}

func TestService_CreateUser_NonAdmin(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	regular := givenUser(mockRepo, auth.UserID(1), "regular", auth.UserRole)

	createdUser, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(regular))
	is.True(err != nil)
	is.Equal(err, auth.ErrAdminCredentialsRequired)
	is.True(createdUser == nil)
}

func TestService_CreateUser_DuplicateUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)

	_, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(admin))
	is.NoErr(err)

	mockRepo.createUserErr = auth.ErrUsernameTaken
	createdUser, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(admin))
	is.True(err != nil)
	is.Equal(err, auth.ErrUsernameTaken)
	is.True(createdUser == nil)
}

func TestService_BootstrapAdmin_CreatesAdminWhenNoUsers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	err := service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "AdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, auth.AdminRole)
	is.Equal(admin.Username, "admin")
}

func TestService_BootstrapAdmin_CreatesAdminWhenNoAdminsExist(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	eu, err := auth.NewUser("existing", "Existing User", "", "Password123", auth.UserRole, nil)
	is.NoErr(err)
	eu.ID = auth.UserID(1)
	existingUser := &eu
	mockRepo.users[existingUser.ID] = existingUser
	mockRepo.usersByUsername[existingUser.Username] = existingUser
	mockRepo.userCount = 1
	mockRepo.adminCount = 0

	err = service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "AdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 2)

	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, auth.AdminRole)
}

func TestService_BootstrapAdmin_SkipsWhenAdminExists(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	givenUser(mockRepo, auth.UserID(1), "existing_admin", auth.AdminRole)

	err := service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "AdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)
	is.Equal(mockRepo.adminCount, 1)
}

func TestService_BootstrapAdmin_UsesProvidedPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	err := service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "CustomAdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, auth.AdminRole)

	token, _, err := service.Login(ctx, "admin", "CustomAdminPass123!")
	is.NoErr(err)
	is.True(token != "")
}

func TestService_ListUsers_ReturnsAllUsers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "alice", auth.AdminRole)
	givenUser(mockRepo, auth.UserID(2), "bob", auth.UserRole)

	users, err := service.ListUsers(ctx, principalFor(admin))
	is.NoErr(err)
	is.Equal(len(users), 2)
}

func TestService_ListUsers_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	regular := givenUser(mockRepo, auth.UserID(1), "regular", auth.UserRole)

	users, err := service.ListUsers(ctx, principalFor(regular))
	is.Equal(err, auth.ErrAdminCredentialsRequired)
	is.True(users == nil)
}

func TestService_UpdateOwnProfile_UpdatesDisplayName(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	givenUser(mockRepo, auth.UserID(1), "alice", auth.UserRole)

	updated, err := service.UpdateOwnProfile(ctx, auth.UserID(1), auth.ProfileUpdates{DisplayName: new("Alice Updated")})
	is.NoErr(err)
	is.Equal(updated.DisplayName, "Alice Updated")
}

func TestService_UpdateOwnProfile_UpdatesUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	givenUser(mockRepo, auth.UserID(1), "alice", auth.UserRole)

	updated, err := service.UpdateOwnProfile(ctx, auth.UserID(1), auth.ProfileUpdates{Username: new("alice2")})
	is.NoErr(err)
	is.Equal(updated.Username, "alice2")
}

func TestService_UpdateOwnProfile_NoFieldsReturnsError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	givenUser(mockRepo, auth.UserID(1), "alice", auth.UserRole)

	updated, err := service.UpdateOwnProfile(ctx, auth.UserID(1), auth.ProfileUpdates{})
	is.Equal(err, auth.ErrNoUpdateFields)
	is.True(updated == nil)
}

func TestService_UpdateOwnProfile_UserNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	updated, err := service.UpdateOwnProfile(ctx, auth.UserID(99), auth.ProfileUpdates{DisplayName: new("Ghost")})
	is.Equal(err, auth.ErrUserNotFound)
	is.True(updated == nil)
}

func TestService_ChangePassword_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	u, err := auth.NewUser("alice", "Alice", "", "OldPass123!", auth.UserRole, nil)
	is.NoErr(err)
	u.ID = auth.UserID(1)
	user := &u
	mockRepo.users[user.ID] = user
	mockRepo.usersByUsername[user.Username] = user
	mockRepo.userCount++
	principal := givenSessionFor(mockRepo, user)

	err = service.ChangePassword(ctx, user.ID, principal.SessionID, "OldPass123!", "NewPass456!")
	is.NoErr(err)
	is.Equal(len(mockRepo.sessions), 1)
}

func TestService_ChangePassword_WrongCurrentPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	u, err := auth.NewUser("alice", "Alice", "", "OldPass123!", auth.UserRole, nil)
	is.NoErr(err)
	u.ID = auth.UserID(1)
	user := &u
	mockRepo.users[user.ID] = user
	mockRepo.userCount++

	err = service.ChangePassword(ctx, user.ID, auth.SessionID(1), "WrongPass!", "NewPass456!")
	is.Equal(err, auth.ErrInvalidCredentials)
}

func TestService_ChangePassword_RevokesOtherSessions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	u, err := auth.NewUser("alice", "Alice", "", "OldPass123!", auth.UserRole, nil)
	is.NoErr(err)
	u.ID = auth.UserID(1)
	user := &u
	mockRepo.users[user.ID] = user
	mockRepo.userCount++

	principalS1 := givenSessionFor(mockRepo, user)
	_ = givenSessionFor(mockRepo, user)

	err = service.ChangePassword(ctx, user.ID, principalS1.SessionID, "OldPass123!", "NewPass456!")
	is.NoErr(err)

	is.Equal(len(mockRepo.sessions), 1)
	_, s1Kept := mockRepo.sessions[principalS1.SessionID]
	is.True(s1Kept)
}

func TestService_PromoteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)
	target := givenUser(mockRepo, auth.UserID(2), "alice", auth.UserRole)

	updated, err := service.PromoteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	is.Equal(updated.Role, auth.AdminRole)
}

func TestService_PromoteUser_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	regular := givenUser(mockRepo, auth.UserID(1), "regular", auth.UserRole)
	target := givenUser(mockRepo, auth.UserID(2), "alice", auth.UserRole)

	updated, err := service.PromoteUser(ctx, principalFor(regular), target.ID)
	is.Equal(err, auth.ErrAdminCredentialsRequired)
	is.True(updated == nil)
}

func TestService_PromoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)

	updated, err := service.PromoteUser(ctx, principalFor(admin), admin.ID)
	is.Equal(err, auth.ErrSelfRoleChangeForbidden)
	is.True(updated == nil)
}

func TestService_DemoteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin1", auth.AdminRole)
	target := givenUser(mockRepo, auth.UserID(2), "admin2", auth.AdminRole)

	updated, err := service.DemoteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	is.Equal(updated.Role, auth.UserRole)
}

func TestService_DemoteUser_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	regular := givenUser(mockRepo, auth.UserID(1), "regular", auth.UserRole)
	target := givenUser(mockRepo, auth.UserID(2), "admin", auth.AdminRole)

	updated, err := service.DemoteUser(ctx, principalFor(regular), target.ID)
	is.Equal(err, auth.ErrAdminCredentialsRequired)
	is.True(updated == nil)
}

func TestService_DemoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)

	updated, err := service.DemoteUser(ctx, principalFor(admin), admin.ID)
	is.Equal(err, auth.ErrSelfRoleChangeForbidden)
	is.True(updated == nil)
}

func TestService_DeleteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)
	target := givenUser(mockRepo, auth.UserID(2), "alice", auth.UserRole)

	err := service.DeleteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	_, exists := mockRepo.users[target.ID]
	is.True(!exists)
}

func TestService_DeleteUser_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	regular := givenUser(mockRepo, auth.UserID(1), "regular", auth.UserRole)
	target := givenUser(mockRepo, auth.UserID(2), "alice", auth.UserRole)

	err := service.DeleteUser(ctx, principalFor(regular), target.ID)
	is.Equal(err, auth.ErrAdminCredentialsRequired)
}

func TestService_DeleteUser_SelfDeleteForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)

	err := service.DeleteUser(ctx, principalFor(admin), admin.ID)
	is.Equal(err, auth.ErrSelfDeleteForbidden)
}

func TestService_DeleteUser_RevokesSessionsOnDelete(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := newService(mockRepo)

	admin := givenUser(mockRepo, auth.UserID(1), "admin", auth.AdminRole)
	target := givenUser(mockRepo, auth.UserID(2), "alice", auth.UserRole)
	givenSessionFor(mockRepo, target)

	err := service.DeleteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	is.Equal(len(mockRepo.sessions), 0)
}

type mockRepository struct {
	users                map[auth.UserID]*auth.User
	usersByUsername      map[string]*auth.User
	sessions             map[auth.SessionID]*auth.SessionWithUser
	sessionsByToken      map[string]*auth.SessionWithUser
	userCount            int
	adminCount           int
	getUserByUsernameErr error
	getUserByIDErr       error
	createUserErr        error
	createSessionErr     error
	getSessionErr        error
	countUsersErr        error
	countAdminUsersErr   error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:           make(map[auth.UserID]*auth.User),
		usersByUsername: make(map[string]*auth.User),
		sessions:        make(map[auth.SessionID]*auth.SessionWithUser),
		sessionsByToken: make(map[string]*auth.SessionWithUser),
	}
}

func (m *mockRepository) GetUserByUsername(_ context.Context, username string) (*auth.User, error) {
	if m.getUserByUsernameErr != nil {
		return nil, m.getUserByUsernameErr
	}
	user, ok := m.usersByUsername[username]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) GetUserByID(_ context.Context, userID auth.UserID) (*auth.User, error) {
	if m.getUserByIDErr != nil {
		return nil, m.getUserByIDErr
	}
	user, ok := m.users[userID]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) CreateUser(_ context.Context, user *auth.User) (*auth.User, error) {
	if m.createUserErr != nil {
		return nil, m.createUserErr
	}
	user.ID = auth.UserID(len(m.users) + 1)
	m.users[user.ID] = user
	m.usersByUsername[user.Username] = user
	m.userCount++
	if user.Role == auth.AdminRole {
		m.adminCount++
	}
	return user, nil
}

func (m *mockRepository) CreateSession(_ context.Context, session *auth.Session) (*auth.Session, error) {
	if m.createSessionErr != nil {
		return nil, m.createSessionErr
	}
	session.ID = auth.SessionID(len(m.sessions) + 1)
	swu := &auth.SessionWithUser{Session: *session, UserRole: m.users[session.UserID].Role}
	m.sessions[session.ID] = swu
	m.sessionsByToken[session.TokenHash] = swu
	return session, nil
}

func (m *mockRepository) GetSessionWithRoleByTokenHash(_ context.Context, tokenHash string) (*auth.SessionWithUser, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	session, ok := m.sessionsByToken[tokenHash]
	if !ok {
		return nil, auth.ErrInvalidCredentials
	}
	return session, nil
}

func (m *mockRepository) CountUsers(_ context.Context) (int, error) {
	if m.countUsersErr != nil {
		return 0, m.countUsersErr
	}
	return m.userCount, nil
}

func (m *mockRepository) CountAdminUsers(_ context.Context) (int, error) {
	if m.countAdminUsersErr != nil {
		return 0, m.countAdminUsersErr
	}
	return m.adminCount, nil
}

func (m *mockRepository) GetAllUsers(_ context.Context) ([]auth.User, error) {
	users := make([]auth.User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, nil
}

func (m *mockRepository) UpdateUser(_ context.Context, user *auth.User) (*auth.User, error) {
	existing, ok := m.users[user.ID]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	if existing.Role == auth.AdminRole && user.Role != auth.AdminRole {
		m.adminCount--
	}
	if existing.Role != auth.AdminRole && user.Role == auth.AdminRole {
		m.adminCount++
	}
	if existing.Username != user.Username {
		delete(m.usersByUsername, existing.Username)
		m.usersByUsername[user.Username] = user
	}
	*existing = *user
	return existing, nil
}

func (m *mockRepository) UpdatePasswordHash(_ context.Context, userID auth.UserID, newHash []byte) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.PasswordHash = newHash
	user.MustChangePassword = false
	return nil
}

func (m *mockRepository) SoftDeleteUser(_ context.Context, userID auth.UserID) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	if user.Role == auth.AdminRole {
		m.adminCount--
	}
	m.userCount--
	delete(m.usersByUsername, user.Username)
	delete(m.users, userID)
	return nil
}

func (m *mockRepository) RevokeSessionByID(_ context.Context, id auth.SessionID) error {
	session, ok := m.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	delete(m.sessionsByToken, session.TokenHash)
	delete(m.sessions, id)
	return nil
}

func (m *mockRepository) RevokeAllUserSessions(_ context.Context, userID auth.UserID) error {
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessionsByToken, session.TokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockRepository) RevokeAllUserSessionsExcept(_ context.Context, userID auth.UserID, exceptSessionID auth.SessionID) error {
	for id, session := range m.sessions {
		if session.UserID == userID && id != exceptSessionID {
			delete(m.sessionsByToken, session.TokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}
