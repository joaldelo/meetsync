package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"meetsync/internal/api"
	"meetsync/internal/models"
	"meetsync/pkg/errors"
	"meetsync/pkg/logs"
)

// UserHandler handles user-related requests
type UserHandler struct {
	// In a real application, you would inject a service or repository here
	users map[string]models.User // In-memory storage for demo purposes
}

// NewUserHandler creates a new UserHandler
func NewUserHandler() *UserHandler {
	return &UserHandler{
		users: make(map[string]models.User),
	}
}

// CreateUser handles the creation of a new user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.NewValidationError("Method not allowed", "Only POST method is allowed")
	}

	var req api.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
	}

	// Validate request
	if req.Name == "" {
		return errors.NewValidationError("Name is required", "")
	}
	if req.Email == "" {
		return errors.NewValidationError("Email is required", "")
	}

	// Check if email is already in use
	for _, user := range h.users {
		if strings.EqualFold(user.Email, req.Email) {
			return errors.NewConflictError("Email is already in use")
		}
	}

	// Create user
	now := time.Now()
	user := models.User{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// In a real application, you would save the user to a database
	h.users[user.ID] = user

	logs.Info("Created user: %s (%s)", user.Name, user.ID)

	// Return response
	resp := api.CreateUserResponse{
		User: user,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// GetUser handles fetching a user by ID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewValidationError("Method not allowed", "Only GET method is allowed")
	}

	// Extract user ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		return errors.NewValidationError("Invalid path", "User ID not provided")
	}
	userID := parts[3]

	// Get user
	user, exists := h.users[userID]
	if !exists {
		return errors.NewNotFoundError("User not found")
	}

	// Return response
	resp := api.GetUserResponse{
		User: user,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// ListUsers handles listing all users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewValidationError("Method not allowed", "Only GET method is allowed")
	}

	// Get all users
	users := make([]models.User, 0, len(h.users))
	for _, user := range h.users {
		users = append(users, user)
	}

	// Return response
	resp := api.ListUsersResponse{
		Users: users,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}
