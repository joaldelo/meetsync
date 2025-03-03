package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meetsync/internal/api"
	"meetsync/internal/models"
)

func TestCreateUser(t *testing.T) {
	// Create a new user handler
	handler := NewUserHandler()

	// Test cases
	tests := []struct {
		name           string
		requestBody    api.CreateUserRequest
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid user creation",
			requestBody: api.CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp api.CreateUserResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.User.Name != "John Doe" {
					t.Errorf("Expected name 'John Doe', got '%s'", resp.User.Name)
				}
				if resp.User.Email != "john@example.com" {
					t.Errorf("Expected email 'john@example.com', got '%s'", resp.User.Email)
				}
				if resp.User.ID == "" {
					t.Error("Expected non-empty user ID")
				}
			},
		},
		{
			name: "Missing name",
			requestBody: api.CreateUserRequest{
				Email: "john@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Name is required\n" {
					t.Errorf("Expected error message 'Name is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Missing email",
			requestBody: api.CreateUserRequest{
				Name: "John Doe",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Email is required\n" {
					t.Errorf("Expected error message 'Email is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Duplicate email",
			requestBody: api.CreateUserRequest{
				Name:  "Jane Doe",
				Email: "john@example.com", // Same as first test case
			},
			expectedStatus: http.StatusConflict,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Email is already in use\n" {
					t.Errorf("Expected error message 'Email is already in use', got '%s'", w.Body.String())
				}
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			reqBody, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			handler.CreateUser(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// Run validation function
			tc.validateFunc(t, w)
		})
	}
}

func TestGetUser(t *testing.T) {
	// Create a new user handler
	handler := NewUserHandler()

	// Create a test user
	user := models.User{
		ID:    "test-user-id",
		Name:  "Test User",
		Email: "test@example.com",
	}
	handler.users["test-user-id"] = user

	// Test cases
	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Valid user ID",
			userID:         "test-user-id",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp api.GetUserResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.User.ID != "test-user-id" {
					t.Errorf("Expected user ID 'test-user-id', got '%s'", resp.User.ID)
				}
				if resp.User.Name != "Test User" {
					t.Errorf("Expected name 'Test User', got '%s'", resp.User.Name)
				}
				if resp.User.Email != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got '%s'", resp.User.Email)
				}
			},
		},
		{
			name:           "Invalid user ID",
			userID:         "non-existent-id",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "User not found\n" {
					t.Errorf("Expected error message 'User not found', got '%s'", w.Body.String())
				}
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest(http.MethodGet, "/api/users/"+tc.userID, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Set up the request path
			req.URL.Path = "/api/users/" + tc.userID

			// Call handler
			handler.GetUser(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// Run validation function
			tc.validateFunc(t, w)
		})
	}
}

func TestListUsers(t *testing.T) {
	// Create a new user handler
	handler := NewUserHandler()

	// Create test users
	handler.users["user1"] = models.User{ID: "user1", Name: "User 1", Email: "user1@example.com"}
	handler.users["user2"] = models.User{ID: "user2", Name: "User 2", Email: "user2@example.com"}

	// Create request
	req, err := http.NewRequest(http.MethodGet, "/api/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	// Call handler
	handler.ListUsers(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Parse response
	var resp api.ListUsersResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Check number of users
	if len(resp.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(resp.Users))
	}

	// Check user IDs
	userIDs := make(map[string]bool)
	for _, user := range resp.Users {
		userIDs[user.ID] = true
	}
	if !userIDs["user1"] || !userIDs["user2"] {
		t.Errorf("Expected users with IDs 'user1' and 'user2', got %v", userIDs)
	}
}
