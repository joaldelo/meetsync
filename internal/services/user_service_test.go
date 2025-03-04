package services

import (
	"testing"

	"meetsync/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name         string
		inputName    string
		inputEmail   string
		expectError  bool
		errorType    string
		errorMessage string
	}{
		{
			name:       "Valid user creation",
			inputName:  "John Doe",
			inputEmail: "john@example.com",
		},
		{
			name:         "Empty name",
			inputName:    "",
			inputEmail:   "john@example.com",
			expectError:  true,
			errorType:    "ValidationError",
			errorMessage: "Name is required",
		},
		{
			name:         "Empty email",
			inputName:    "John Doe",
			inputEmail:   "",
			expectError:  true,
			errorType:    "ValidationError",
			errorMessage: "Email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewUserService()
			user, err := service.CreateUser(tt.inputName, tt.inputEmail)

			if tt.expectError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.errorMessage, appErr.Message)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, user.ID)
				assert.Equal(t, tt.inputName, user.Name)
				assert.Equal(t, tt.inputEmail, user.Email)
			}
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	service := NewUserService()

	// Create a test user first
	testUser, err := service.CreateUser("Test User", "test@example.com")
	assert.NoError(t, err)

	tests := []struct {
		name        string
		userID      string
		expectError bool
	}{
		{
			name:   "Existing user",
			userID: testUser.ID,
		},
		{
			name:        "Non-existing user",
			userID:      "non-existing-id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.GetUserByID(tt.userID)

			if tt.expectError {
				assert.Error(t, err)
				appErr, ok := err.(*errors.AppError)
				assert.True(t, ok)
				assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testUser.ID, user.ID)
				assert.Equal(t, testUser.Name, user.Name)
				assert.Equal(t, testUser.Email, user.Email)
			}
		})
	}
}

func TestUserService_ListUsers(t *testing.T) {
	service := NewUserService()

	// Create some test users
	testUsers := []struct {
		name  string
		email string
	}{
		{"User 1", "user1@example.com"},
		{"User 2", "user2@example.com"},
		{"User 3", "user3@example.com"},
	}

	for _, u := range testUsers {
		_, err := service.CreateUser(u.name, u.email)
		assert.NoError(t, err)
	}

	// Test listing users
	users, err := service.ListUsers()
	assert.NoError(t, err)
	assert.Len(t, users, len(testUsers))

	// Verify each user has required fields
	for _, user := range users {
		assert.NotEmpty(t, user.ID)
		assert.NotEmpty(t, user.Name)
		assert.NotEmpty(t, user.Email)
	}
}
