package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"meetsync/internal/api"
	"meetsync/internal/models"
	"meetsync/pkg/errors"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user models.User) (models.User, error) {
	args := m.Called(user)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockUserRepository) GetByID(id string) (models.User, error) {
	args := m.Called(id)
	return args.Get(0).(models.User), args.Error(1)
}

func (m *MockUserRepository) GetAll() ([]models.User, error) {
	args := m.Called()
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (models.User, bool) {
	args := m.Called(email)
	return args.Get(0).(models.User), args.Bool(1)
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name           string
		request        api.CreateUserRequest
		setupMock      func(*MockUserRepository)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful creation",
			request: api.CreateUserRequest{
				Name:  "Test User",
				Email: "test@example.com",
			},
			setupMock: func(m *MockUserRepository) {
				user := models.User{
					ID:        uuid.New().String(),
					Name:      "Test User",
					Email:     "test@example.com",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("Create", mock.MatchedBy(func(u models.User) bool {
					return u.Name == "Test User" && u.Email == "test@example.com"
				})).Return(user, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "duplicate email",
			request: api.CreateUserRequest{
				Name:  "Test User",
				Email: "existing@example.com",
			},
			setupMock: func(m *MockUserRepository) {
				m.On("Create", mock.Anything).Return(models.User{}, errors.NewConflictError("Email is already in use"))
			},
			expectedStatus: http.StatusConflict,
			expectedError:  true,
		},
		{
			name: "missing name",
			request: api.CreateUserRequest{
				Email: "test@example.com",
			},
			setupMock:      func(m *MockUserRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "missing email",
			request: api.CreateUserRequest{
				Name: "Test User",
			},
			setupMock:      func(m *MockUserRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := new(MockUserRepository)
			tt.setupMock(mockRepo)

			// Create handler with mock repository
			handler := &UserHandler{repository: mockRepo}

			// Create request
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Handle request
			err := handler.CreateUser(w, req)

			// Check response
			if tt.expectedError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.expectedStatus, appErr.HTTPStatusCode())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, w.Code)

				var resp api.CreateUserResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.request.Name, resp.User.Name)
				assert.Equal(t, tt.request.Email, resp.User.Email)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(*MockUserRepository)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:   "successful get",
			userID: "test-id",
			setupMock: func(m *MockUserRepository) {
				user := models.User{
					ID:        "test-id",
					Name:      "Test User",
					Email:     "test@example.com",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("GetByID", "test-id").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:   "user not found",
			userID: "non-existent-id",
			setupMock: func(m *MockUserRepository) {
				m.On("GetByID", "non-existent-id").Return(models.User{}, errors.NewNotFoundError("User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := new(MockUserRepository)
			tt.setupMock(mockRepo)

			// Create handler with mock repository
			handler := &UserHandler{repository: mockRepo}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/users/"+tt.userID, nil)
			w := httptest.NewRecorder()

			// Handle request
			err := handler.GetUser(w, req)

			// Check response
			if tt.expectedError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.expectedStatus, appErr.HTTPStatusCode())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, w.Code)

				var resp api.GetUserResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, resp.User.ID)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestListUsers(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockUserRepository)
		expectedStatus int
		expectedError  bool
		expectedCount  int
	}{
		{
			name: "successful list",
			setupMock: func(m *MockUserRepository) {
				users := []models.User{
					{
						ID:        "user-1",
						Name:      "User 1",
						Email:     "user1@example.com",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					{
						ID:        "user-2",
						Name:      "User 2",
						Email:     "user2@example.com",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				}
				m.On("GetAll").Return(users, nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedCount:  2,
		},
		{
			name: "empty list",
			setupMock: func(m *MockUserRepository) {
				m.On("GetAll").Return([]models.User{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := new(MockUserRepository)
			tt.setupMock(mockRepo)

			// Create handler with mock repository
			handler := &UserHandler{repository: mockRepo}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
			w := httptest.NewRecorder()

			// Handle request
			err := handler.ListUsers(w, req)

			// Check response
			if tt.expectedError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.expectedStatus, appErr.HTTPStatusCode())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, w.Code)

				var resp api.ListUsersResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(resp.Users))
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}
