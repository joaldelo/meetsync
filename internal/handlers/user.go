package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"meetsync/internal/api"
	"meetsync/internal/models"
	"meetsync/internal/repositories"
	"meetsync/pkg/errors"
	"meetsync/pkg/logs"
)

// UserHandler handles user-related requests
type UserHandler struct {
	repository repositories.UserRepository
}

// NewUserHandler creates a new UserHandler
func NewUserHandler() *UserHandler {
	return &UserHandler{
		repository: repositories.NewInMemoryUserRepository(),
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

	// Create user model
	user := models.User{
		Name:  req.Name,
		Email: req.Email,
	}

	// Create user using repository
	createdUser, err := h.repository.Create(user)
	if err != nil {
		return err
	}

	logs.Info("Created user: %s (%s)", createdUser.Name, createdUser.ID)

	// Return response
	resp := api.CreateUserResponse{
		User: createdUser,
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

	// Get user using repository
	user, err := h.repository.GetByID(userID)
	if err != nil {
		return err
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

	// Get all users using repository
	users, err := h.repository.GetAll()
	if err != nil {
		return err
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
