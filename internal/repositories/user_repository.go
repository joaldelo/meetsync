package repositories

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"meetsync/internal/models"
	"meetsync/pkg/errors"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(user models.User) (models.User, error)
	GetByID(id string) (models.User, error)
	GetAll() ([]models.User, error)
	GetByEmail(email string) (models.User, bool)
}

// InMemoryUserRepository implements UserRepository using in-memory storage
type InMemoryUserRepository struct {
	users map[string]models.User
	mu    sync.RWMutex
}

// NewInMemoryUserRepository creates a new InMemoryUserRepository
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users: make(map[string]models.User),
	}
}

func (r *InMemoryUserRepository) Create(user models.User) (models.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if email is already in use
	for _, existingUser := range r.users {
		if strings.EqualFold(existingUser.Email, user.Email) {
			return models.User{}, errors.NewConflictError("Email is already in use")
		}
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Generate ID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	r.users[user.ID] = user
	return user, nil
}

func (r *InMemoryUserRepository) GetByID(id string) (models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return models.User{}, errors.NewNotFoundError("User not found")
	}
	return user, nil
}

func (r *InMemoryUserRepository) GetAll() ([]models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]models.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

func (r *InMemoryUserRepository) GetByEmail(email string) (models.User, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if strings.EqualFold(user.Email, email) {
			return user, true
		}
	}
	return models.User{}, false
}
