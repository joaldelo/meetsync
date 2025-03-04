package services

import (
	"meetsync/internal/interfaces"
	"meetsync/internal/models"
	"meetsync/internal/repositories"
	"meetsync/pkg/errors"
)

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	repository repositories.UserRepository
}

var _ interfaces.UserService = (*UserServiceImpl)(nil) // Verify UserServiceImpl implements UserService interface

// NewUserService creates a new UserService
func NewUserService() interfaces.UserService {
	return &UserServiceImpl{
		repository: repositories.NewInMemoryUserRepository(),
	}
}

// CreateUser creates a new user
func (s *UserServiceImpl) CreateUser(name, email string) (models.User, error) {
	// Validate input
	if name == "" {
		return models.User{}, errors.NewValidationError("Name is required", "")
	}
	if email == "" {
		return models.User{}, errors.NewValidationError("Email is required", "")
	}

	// Create user model
	user := models.User{
		Name:  name,
		Email: email,
	}

	// Create user using repository
	return s.repository.Create(user)
}

// GetUserByID retrieves a user by their ID
func (s *UserServiceImpl) GetUserByID(userID string) (models.User, error) {
	return s.repository.GetByID(userID)
}

// ListUsers retrieves all users
func (s *UserServiceImpl) ListUsers() ([]models.User, error) {
	return s.repository.GetAll()
}
