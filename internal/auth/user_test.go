package auth

import (
	"errors"
	"testing"

	"github.com/matryer/is"
)

func TestNewUser_ValidInputs(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName string
		email       *string
		password    string
		role        Role
	}{
		{
			name:        "valid user with email",
			username:    "john_doe",
			displayName: "John Doe",
			email:       stringPtr("john@example.com"),
			password:    "Password123",
			role:        UserRole,
		},
		{
			name:        "valid user without email",
			username:    "jane_doe",
			displayName: "Jane Doe",
			email:       nil,
			password:    "Password123",
			role:        UserRole,
		},
		{
			name:        "valid admin user",
			username:    "admin_user",
			displayName: "Admin User",
			email:       nil,
			password:    "AdminPass123!",
			role:        AdminRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			user, err := NewUser(tt.username, tt.displayName, tt.email, tt.password, tt.role, nil)
			is.NoErr(err)
			is.True(user != nil)
			is.Equal(user.Username, tt.username)
			is.Equal(user.DisplayName, tt.displayName)
			is.Equal(user.Email, tt.email)
			is.Equal(user.Role, tt.role)
			is.True(len(user.PasswordHash) > 0)
		})
	}
}

func TestNewUser_InvalidUsername(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName string
		password    string
		wantErr     error
	}{
		{
			name:        "username too short",
			username:    "ab",
			displayName: "Test User",
			password:    "Password123",
			wantErr:     ErrInvalidUsername,
		},
		{
			name:        "username too long",
			username:    "this_is_a_very_long_username_that_exceeds_limit",
			displayName: "Test User",
			password:    "Password123",
			wantErr:     ErrInvalidUsername,
		},
		{
			name:        "username with invalid characters",
			username:    "john.doe",
			displayName: "Test User",
			password:    "Password123",
			wantErr:     ErrInvalidUsername,
		},
		{
			name:        "username with uppercase normalized to lowercase",
			username:    "JOHN_DOE",
			displayName: "Test User",
			password:    "Password123",
			wantErr:     nil,
		},
		{
			name:        "username with whitespace trimmed",
			username:    "  john_doe  ",
			displayName: "Test User",
			password:    "Password123",
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			user, err := NewUser(tt.username, tt.displayName, nil, tt.password, UserRole, nil)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
				is.True(user == nil)
			} else {
				is.NoErr(err)
				is.True(user != nil)
				// Verify normalization
				expectedUsername := "john_doe"
				is.Equal(user.Username, expectedUsername)
			}
		})
	}
}

func TestNewUser_InvalidDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName string
		password    string
		wantErr     error
	}{
		{
			name:        "display name too short",
			username:    "testuser",
			displayName: "",
			password:    "Password123",
			wantErr:     ErrInvalidDisplayName,
		},
		{
			name:        "display name too long",
			username:    "testuser",
			displayName: "This is a very long display name that exceeds the maximum allowed length of fifty characters",
			password:    "Password123",
			wantErr:     ErrInvalidDisplayName,
		},
		{
			name:        "display name with whitespace trimmed",
			username:    "testuser",
			displayName: "  John Doe  ",
			password:    "Password123",
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			user, err := NewUser(tt.username, tt.displayName, nil, tt.password, UserRole, nil)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
				is.True(user == nil)
			} else {
				is.NoErr(err)
				is.True(user != nil)
				// Verify trimming
				if tt.displayName == "  John Doe  " {
					is.Equal(user.DisplayName, "John Doe")
				}
			}
		})
	}
}

func TestNewUser_InvalidPassword(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		displayName string
		password    string
		wantErr     error
	}{
		{
			name:        "password too short",
			username:    "testuser",
			displayName: "Test User",
			password:    "Pass1",
			wantErr:     ErrInvalidPassword,
		},
		{
			name:        "password too long",
			username:    "testuser",
			displayName: "Test User",
			password:    "ThisIsAVeryLongPasswordThatExceedsTheMaximumAllowedLength",
			wantErr:     ErrInvalidPassword,
		},
		{
			name:        "valid password minimum length",
			username:    "testuser",
			displayName: "Test User",
			password:    "Password",
			wantErr:     nil,
		},
		{
			name:        "valid password maximum length",
			username:    "testuser",
			displayName: "Test User",
			password:    "ThisIsAVeryLongPassword123",
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			user, err := NewUser(tt.username, tt.displayName, nil, tt.password, UserRole, nil)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
				is.True(user == nil)
			} else {
				is.NoErr(err)
				is.True(user != nil)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     string
		wantErr  error
	}{
		{
			name:     "valid username",
			username: "john_doe",
			want:     "john_doe",
			wantErr:  nil,
		},
		{
			name:     "valid username with numbers",
			username: "user123",
			want:     "user123",
			wantErr:  nil,
		},
		{
			name:     "valid username with dash",
			username: "user-name",
			want:     "user-name",
			wantErr:  nil,
		},
		{
			name:     "uppercase normalized",
			username: "JOHN_DOE",
			want:     "john_doe",
			wantErr:  nil,
		},
		{
			name:     "whitespace trimmed",
			username: "  john_doe  ",
			want:     "john_doe",
			wantErr:  nil,
		},
		{
			name:     "too short",
			username: "ab",
			want:     "",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "too long",
			username: "this_is_a_very_long_username_that_exceeds_limit",
			want:     "",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "invalid characters dot",
			username: "john.doe",
			want:     "",
			wantErr:  ErrInvalidUsername,
		},
		{
			name:     "invalid characters space",
			username: "john doe",
			want:     "",
			wantErr:  ErrInvalidUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			got, err := validateUsername(tt.username)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
			} else {
				is.NoErr(err)
				is.Equal(got, tt.want)
			}
		})
	}
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		want        string
		wantErr     error
	}{
		{
			name:        "valid display name",
			displayName: "John Doe",
			want:        "John Doe",
			wantErr:     nil,
		},
		{
			name:        "single character",
			displayName: "J",
			want:        "J",
			wantErr:     nil,
		},
		{
			name:        "maximum length",
			displayName: "This is exactly fifty characters long display name",
			want:        "This is exactly fifty characters long display name",
			wantErr:     nil,
		},
		{
			name:        "whitespace trimmed",
			displayName: "  John Doe  ",
			want:        "John Doe",
			wantErr:     nil,
		},
		{
			name:        "empty after trim",
			displayName: "   ",
			want:        "",
			wantErr:     ErrInvalidDisplayName,
		},
		{
			name:        "too long",
			displayName: "This is a very long display name that exceeds the maximum allowed length of fifty characters",
			want:        "",
			wantErr:     ErrInvalidDisplayName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			got, err := validateDisplayName(tt.displayName)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
			} else {
				is.NoErr(err)
				is.Equal(got, tt.want)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password minimum length",
			password: "Password",
			wantErr:  nil,
		},
		{
			name:     "valid password maximum length",
			password: "ThisIsAVeryLongPassword123",
			wantErr:  nil,
		},
		{
			name:     "too short",
			password: "Pass1",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "too long",
			password: "ThisIsAVeryLongPasswordThatExceedsTheMaximumAllowedLength",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "exactly 8 characters",
			password: "12345678",
			wantErr:  nil,
		},
		{
			name:     "exactly 32 characters",
			password: "12345678901234567890123456789012",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			err := validatePassword(tt.password)
			if tt.wantErr != nil {
				is.True(err != nil)
				is.True(errors.Is(err, tt.wantErr))
			} else {
				is.NoErr(err)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
