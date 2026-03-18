//go:build test

package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DiegoGuidaF/PulseWeaver/internal/config"
	"github.com/matryer/is"
)

// givenUser inserts a user into the mock repo with the given ID and role.
// No password hash is set; use NewUser directly when password verification is needed.
func givenUser(repo *mockRepository, id UserID, username string, role Role) *User {
	user := &User{
		ID:          id,
		Username:    username,
		DisplayName: username,
		Role:        role,
	}
	repo.users[id] = user
	repo.usersByUsername[username] = user
	repo.userCount++
	if role == AdminRole {
		repo.adminCount++
	}
	return user
}

// principalFor returns a Principal for the given user. The SessionID is a fixed
// dummy value sufficient for tests that do not exercise session-specific logic.
func principalFor(user *User) *Principal {
	return NewPrincipal(user.ID, SessionID(1), user.Role)
}

// givenSessionFor inserts a session into the mock repo for the given user and
// returns a Principal whose SessionID matches. Use this when the test must
// distinguish sessions (e.g. ChangePassword revokes others but keeps current).
func givenSessionFor(repo *mockRepository, user *User) *Principal {
	id := SessionID(len(repo.sessions) + 1)
	tokenHash := user.Username + "-token"
	session := &Session{ID: id, UserID: user.ID, TokenHash: tokenHash}
	swu := &SessionWithUser{Session: *session, UserRole: user.Role}
	repo.sessions[id] = swu
	repo.sessionsByToken[tokenHash] = swu
	return NewPrincipal(user.ID, id, user.Role)
}

func TestService_Login_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	u, err := NewUser("testuser", "Test User", "", "Password123", UserRole, nil)
	is.NoErr(err)
	u.ID = UserID(1)
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

	u, err := NewUser("testuser", "Test User", "", "Password123", UserRole, nil)
	is.NoErr(err)
	u.ID = UserID(1)
	user := &u
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

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)
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
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	principal := principalFor(givenUser(mockRepo, UserID(1), "admin", AdminRole))
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
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)

	createdUser, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(admin))
	is.NoErr(err)
	is.True(createdUser != nil)
	is.Equal(createdUser.Username, "newuser")
	is.Equal(createdUser.DisplayName, "New User")
	is.Equal(createdUser.Role, UserRole)
}

func TestService_CreateUser_NonAdmin(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	regular := givenUser(mockRepo, UserID(1), "regular", UserRole)

	createdUser, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(regular))
	is.True(err != nil)
	is.Equal(err, ErrAdminCredentialsRequired)
	is.True(createdUser == nil)
}

func TestService_CreateUser_DuplicateUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)

	_, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(admin))
	is.NoErr(err)

	mockRepo.createUserErr = ErrUsernameTaken
	createdUser, err := service.CreateUser(ctx, "newuser", "New User", "", "Password123", principalFor(admin))
	is.True(err != nil)
	is.Equal(err, ErrUsernameTaken)
	is.True(createdUser == nil)
}

func TestService_BootstrapAdmin_CreatesAdminWhenNoUsers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	err := service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "AdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

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

	eu, err := NewUser("existing", "Existing User", "", "Password123", UserRole, nil)
	is.NoErr(err)
	eu.ID = UserID(1)
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
	is.Equal(admin.Role, AdminRole)
}

func TestService_BootstrapAdmin_SkipsWhenAdminExists(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	givenUser(mockRepo, UserID(1), "existing_admin", AdminRole)

	err := service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "AdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)
	is.Equal(mockRepo.adminCount, 1)
}

func TestService_BootstrapAdmin_UsesProvidedPassword(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	err := service.BootstrapAdmin(ctx, config.ConfServer{AdminPassword: "CustomAdminPass123!"})
	is.NoErr(err)
	is.Equal(mockRepo.userCount, 1)

	admin, ok := mockRepo.usersByUsername["admin"]
	is.True(ok)
	is.Equal(admin.Role, AdminRole)

	token, _, err := service.Login(ctx, "admin", "CustomAdminPass123!")
	is.NoErr(err)
	is.True(token != "")
}

func TestService_ListUsers_ReturnsAllUsers(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "alice", AdminRole)
	givenUser(mockRepo, UserID(2), "bob", UserRole)

	users, err := service.ListUsers(ctx, principalFor(admin))
	is.NoErr(err)
	is.Equal(len(users), 2)
}

func TestService_ListUsers_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	regular := givenUser(mockRepo, UserID(1), "regular", UserRole)

	users, err := service.ListUsers(ctx, principalFor(regular))
	is.Equal(err, ErrAdminCredentialsRequired)
	is.True(users == nil)
}

func TestService_UpdateOwnProfile_UpdatesDisplayName(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	givenUser(mockRepo, UserID(1), "alice", UserRole)

	newName := "Alice Updated"
	updated, err := service.UpdateOwnProfile(ctx, UserID(1), ProfileUpdates{DisplayName: &newName})
	is.NoErr(err)
	is.Equal(updated.DisplayName, "Alice Updated")
}

func TestService_UpdateOwnProfile_UpdatesUsername(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	givenUser(mockRepo, UserID(1), "alice", UserRole)

	newUsername := "alice2"
	updated, err := service.UpdateOwnProfile(ctx, UserID(1), ProfileUpdates{Username: &newUsername})
	is.NoErr(err)
	is.Equal(updated.Username, "alice2")
}

func TestService_UpdateOwnProfile_NoFieldsReturnsError(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	givenUser(mockRepo, UserID(1), "alice", UserRole)

	updated, err := service.UpdateOwnProfile(ctx, UserID(1), ProfileUpdates{})
	is.Equal(err, ErrNoUpdateFields)
	is.True(updated == nil)
}

func TestService_UpdateOwnProfile_UserNotFound(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	newName := "Ghost"
	updated, err := service.UpdateOwnProfile(ctx, UserID(99), ProfileUpdates{DisplayName: &newName})
	is.Equal(err, ErrUserNotFound)
	is.True(updated == nil)
}

func TestService_ChangePassword_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	u, err := NewUser("alice", "Alice", "", "OldPass123!", UserRole, nil)
	is.NoErr(err)
	u.ID = UserID(1)
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
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	u, err := NewUser("alice", "Alice", "", "OldPass123!", UserRole, nil)
	is.NoErr(err)
	u.ID = UserID(1)
	user := &u
	mockRepo.users[user.ID] = user
	mockRepo.userCount++

	err = service.ChangePassword(ctx, user.ID, SessionID(1), "WrongPass!", "NewPass456!")
	is.Equal(err, ErrInvalidCredentials)
}

func TestService_ChangePassword_RevokesOtherSessions(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	u, err := NewUser("alice", "Alice", "", "OldPass123!", UserRole, nil)
	is.NoErr(err)
	u.ID = UserID(1)
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
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)
	target := givenUser(mockRepo, UserID(2), "alice", UserRole)

	updated, err := service.PromoteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	is.Equal(updated.Role, AdminRole)
}

func TestService_PromoteUser_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	regular := givenUser(mockRepo, UserID(1), "regular", UserRole)
	target := givenUser(mockRepo, UserID(2), "alice", UserRole)

	updated, err := service.PromoteUser(ctx, principalFor(regular), target.ID)
	is.Equal(err, ErrAdminCredentialsRequired)
	is.True(updated == nil)
}

func TestService_PromoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)

	updated, err := service.PromoteUser(ctx, principalFor(admin), admin.ID)
	is.Equal(err, ErrSelfRoleChangeForbidden)
	is.True(updated == nil)
}

func TestService_DemoteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin1", AdminRole)
	target := givenUser(mockRepo, UserID(2), "admin2", AdminRole)

	updated, err := service.DemoteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	is.Equal(updated.Role, UserRole)
}

func TestService_DemoteUser_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	regular := givenUser(mockRepo, UserID(1), "regular", UserRole)
	target := givenUser(mockRepo, UserID(2), "admin", AdminRole)

	updated, err := service.DemoteUser(ctx, principalFor(regular), target.ID)
	is.Equal(err, ErrAdminCredentialsRequired)
	is.True(updated == nil)
}

func TestService_DemoteUser_SelfForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)

	updated, err := service.DemoteUser(ctx, principalFor(admin), admin.ID)
	is.Equal(err, ErrSelfRoleChangeForbidden)
	is.True(updated == nil)
}

func TestService_DeleteUser_Success(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)
	target := givenUser(mockRepo, UserID(2), "alice", UserRole)

	err := service.DeleteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	_, exists := mockRepo.users[target.ID]
	is.True(!exists)
}

func TestService_DeleteUser_NonAdminForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	regular := givenUser(mockRepo, UserID(1), "regular", UserRole)
	target := givenUser(mockRepo, UserID(2), "alice", UserRole)

	err := service.DeleteUser(ctx, principalFor(regular), target.ID)
	is.Equal(err, ErrAdminCredentialsRequired)
}

func TestService_DeleteUser_SelfDeleteForbidden(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)

	err := service.DeleteUser(ctx, principalFor(admin), admin.ID)
	is.Equal(err, ErrSelfDeleteForbidden)
}

func TestService_DeleteUser_RevokesSessionsOnDelete(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	mockRepo := newMockRepository()
	service := NewService(mockRepo, slog.New(slog.DiscardHandler))

	admin := givenUser(mockRepo, UserID(1), "admin", AdminRole)
	target := givenUser(mockRepo, UserID(2), "alice", UserRole)
	givenSessionFor(mockRepo, target)

	err := service.DeleteUser(ctx, principalFor(admin), target.ID)
	is.NoErr(err)
	is.Equal(len(mockRepo.sessions), 0)
}

type mockRepository struct {
	users                map[UserID]*User
	usersByUsername      map[string]*User
	sessions             map[SessionID]*SessionWithUser
	sessionsByToken      map[string]*SessionWithUser
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

var _ repository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:           make(map[UserID]*User),
		usersByUsername: make(map[string]*User),
		sessions:        make(map[SessionID]*SessionWithUser),
		sessionsByToken: make(map[string]*SessionWithUser),
	}
}

func (m *mockRepository) GetUserByUsername(_ context.Context, username string) (*User, error) {
	if m.getUserByUsernameErr != nil {
		return nil, m.getUserByUsernameErr
	}
	user, ok := m.usersByUsername[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) GetUserByID(_ context.Context, userID UserID) (*User, error) {
	if m.getUserByIDErr != nil {
		return nil, m.getUserByIDErr
	}
	user, ok := m.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) CreateUser(_ context.Context, user *User) (*User, error) {
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

func (m *mockRepository) CreateSession(_ context.Context, session *Session) (*Session, error) {
	if m.createSessionErr != nil {
		return nil, m.createSessionErr
	}
	session.ID = SessionID(len(m.sessions) + 1)
	swu := &SessionWithUser{Session: *session, UserRole: m.users[session.UserID].Role}
	m.sessions[session.ID] = swu
	m.sessionsByToken[session.TokenHash] = swu
	return session, nil
}

func (m *mockRepository) GetSessionWithRoleByTokenHash(_ context.Context, tokenHash string) (*SessionWithUser, error) {
	if m.getSessionErr != nil {
		return nil, m.getSessionErr
	}
	session, ok := m.sessionsByToken[tokenHash]
	if !ok {
		return nil, ErrInvalidCredentials
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

func (m *mockRepository) GetAllUsers(_ context.Context) ([]User, error) {
	users := make([]User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, *user)
	}
	return users, nil
}

func (m *mockRepository) UpdateUser(_ context.Context, user *User) (*User, error) {
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

func (m *mockRepository) UpdatePasswordHash(_ context.Context, userID UserID, newHash []byte) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.PasswordHash = newHash
	user.MustChangePassword = false
	return nil
}

func (m *mockRepository) SoftDeleteUser(_ context.Context, userID UserID) error {
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

func (m *mockRepository) RevokeSessionByID(_ context.Context, id SessionID) error {
	session, ok := m.sessions[id]
	if !ok {
		return errors.New("session not found")
	}
	delete(m.sessionsByToken, session.TokenHash)
	delete(m.sessions, id)
	return nil
}

func (m *mockRepository) RevokeAllUserSessions(_ context.Context, userID UserID) error {
	for id, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessionsByToken, session.TokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockRepository) RevokeAllUserSessionsExcept(_ context.Context, userID UserID, exceptSessionID SessionID) error {
	for id, session := range m.sessions {
		if session.UserID == userID && id != exceptSessionID {
			delete(m.sessionsByToken, session.TokenHash)
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockRepository) RunInTx(_ context.Context, fn func(repository) error) error {
	if m.runInTxFn != nil {
		return m.runInTxFn(m)
	}
	return fn(m)
}
