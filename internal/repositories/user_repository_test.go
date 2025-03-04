package repositories

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"meetsync/internal/models"
)

func TestInMemoryUserRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		existingUsers []models.User
		newUser       models.User
		wantErr       bool
		errType       string
	}{
		{
			name: "successful creation",
			newUser: models.User{
				Name:  "Test User",
				Email: "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "duplicate email",
			existingUsers: []models.User{
				{
					ID:    "existing-id",
					Name:  "Existing User",
					Email: "test@example.com",
				},
			},
			newUser: models.User{
				Name:  "Test User",
				Email: "test@example.com",
			},
			wantErr: true,
			errType: "conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewInMemoryUserRepository()

			// Add existing users if any
			for _, user := range tt.existingUsers {
				_, err := repo.Create(user)
				require.NoError(t, err)
			}

			// Test creating new user
			user, err := repo.Create(tt.newUser)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType == "conflict" {
					assert.Contains(t, err.Error(), "Email is already in use")
				}
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, user.ID)
			assert.Equal(t, tt.newUser.Name, user.Name)
			assert.Equal(t, tt.newUser.Email, user.Email)
			assert.False(t, user.CreatedAt.IsZero())
			assert.False(t, user.UpdatedAt.IsZero())
		})
	}
}

func TestInMemoryUserRepository_GetByID(t *testing.T) {
	repo := NewInMemoryUserRepository()
	existingUser := models.User{
		ID:    "test-id",
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Add test user
	_, err := repo.Create(existingUser)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      "test-id",
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "User not found")
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.id, user.ID)
			assert.Equal(t, existingUser.Name, user.Name)
			assert.Equal(t, existingUser.Email, user.Email)
		})
	}
}

func TestInMemoryUserRepository_GetByEmail(t *testing.T) {
	repo := NewInMemoryUserRepository()
	existingUser := models.User{
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Add test user
	created, err := repo.Create(existingUser)
	require.NoError(t, err)

	tests := []struct {
		name      string
		email     string
		wantFound bool
	}{
		{
			name:      "existing email",
			email:     "test@example.com",
			wantFound: true,
		},
		{
			name:      "non-existent email",
			email:     "nonexistent@example.com",
			wantFound: false,
		},
		{
			name:      "case insensitive email match",
			email:     "TEST@example.com",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, found := repo.GetByEmail(tt.email)

			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, created.ID, user.ID)
				assert.Equal(t, created.Email, user.Email)
			} else {
				assert.Empty(t, user)
			}
		})
	}
}

func TestInMemoryUserRepository_GetAll(t *testing.T) {
	repo := NewInMemoryUserRepository()

	// Test empty repository
	users, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Empty(t, users)

	// Add test users
	testUsers := []models.User{
		{
			Name:  "User 1",
			Email: "user1@example.com",
		},
		{
			Name:  "User 2",
			Email: "user2@example.com",
		},
	}

	for _, user := range testUsers {
		_, err := repo.Create(user)
		require.NoError(t, err)
	}

	// Test non-empty repository
	users, err = repo.GetAll()
	assert.NoError(t, err)
	assert.Len(t, users, len(testUsers))
}

func TestInMemoryUserRepository_ConcurrentOperations(t *testing.T) {
	repo := NewInMemoryUserRepository()
	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent creations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			user := models.User{
				Name:  fmt.Sprintf("User %d", i),
				Email: fmt.Sprintf("user%d@example.com", i),
			}
			_, err := repo.Create(user)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Verify all users were created
	users, err := repo.GetAll()
	assert.NoError(t, err)
	assert.Len(t, users, numGoroutines)

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := repo.GetAll()
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}
