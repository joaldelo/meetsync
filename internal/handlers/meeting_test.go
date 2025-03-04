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

// MockMeetingRepository is a mock implementation of MeetingRepository
type MockMeetingRepository struct {
	mock.Mock
}

func (m *MockMeetingRepository) CreateMeeting(meeting models.Meeting) (models.Meeting, error) {
	args := m.Called(meeting)
	return args.Get(0).(models.Meeting), args.Error(1)
}

func (m *MockMeetingRepository) GetMeetingByID(id string) (models.Meeting, error) {
	args := m.Called(id)
	return args.Get(0).(models.Meeting), args.Error(1)
}

func (m *MockMeetingRepository) UpdateMeeting(meeting models.Meeting) (models.Meeting, error) {
	args := m.Called(meeting)
	return args.Get(0).(models.Meeting), args.Error(1)
}

func (m *MockMeetingRepository) DeleteMeeting(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockMeetingRepository) CreateAvailability(availability models.Availability) (models.Availability, error) {
	args := m.Called(availability)
	return args.Get(0).(models.Availability), args.Error(1)
}

func (m *MockMeetingRepository) GetAvailability(userID, meetingID string) (models.Availability, error) {
	args := m.Called(userID, meetingID)
	return args.Get(0).(models.Availability), args.Error(1)
}

func (m *MockMeetingRepository) UpdateAvailability(availability models.Availability) (models.Availability, error) {
	args := m.Called(availability)
	return args.Get(0).(models.Availability), args.Error(1)
}

func (m *MockMeetingRepository) DeleteAvailability(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockMeetingRepository) GetMeetingAvailabilities(meetingID string) ([]models.Availability, error) {
	args := m.Called(meetingID)
	return args.Get(0).([]models.Availability), args.Error(1)
}

func (m *MockMeetingRepository) GetAllAvailabilities() []models.Availability {
	args := m.Called()
	return args.Get(0).([]models.Availability)
}

func TestCreateMeeting(t *testing.T) {
	// Create test data
	now := time.Now()
	testUser := models.User{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	testSlot := models.TimeSlot{
		StartTime: now,
		EndTime:   now.Add(time.Hour),
	}

	tests := []struct {
		name           string
		request        api.CreateMeetingRequest
		setupMocks     func(*MockUserRepository, *MockMeetingRepository)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful creation",
			request: api.CreateMeetingRequest{
				Title:             "Test Meeting",
				OrganizerID:       testUser.ID,
				EstimatedDuration: 60,
				ProposedSlots:     []models.TimeSlot{testSlot},
				ParticipantIDs:    []string{uuid.New().String()},
			},
			setupMocks: func(ur *MockUserRepository, mr *MockMeetingRepository) {
				// Mock organizer lookup
				ur.On("GetByID", testUser.ID).Return(testUser, nil)

				// Mock participant lookup
				participantID := uuid.New().String()
				participant := models.User{
					ID:        participantID,
					Name:      "Participant",
					Email:     "participant@example.com",
					CreatedAt: now,
					UpdatedAt: now,
				}
				ur.On("GetByID", mock.AnythingOfType("string")).Return(participant, nil)

				// Mock meeting creation with flexible matching
				mr.On("CreateMeeting", mock.MatchedBy(func(m models.Meeting) bool {
					return m.Title == "Test Meeting" &&
						m.OrganizerID == testUser.ID &&
						m.EstimatedDuration == 60 &&
						len(m.ProposedSlots) == 1 &&
						m.ProposedSlots[0].StartTime.Equal(testSlot.StartTime) &&
						m.ProposedSlots[0].EndTime.Equal(testSlot.EndTime) &&
						len(m.Participants) == 1
				})).Run(func(args mock.Arguments) {
					m := args.Get(0).(models.Meeting)
					m.ID = uuid.New().String()
					for i := range m.ProposedSlots {
						m.ProposedSlots[i].ID = uuid.New().String()
					}
					m.CreatedAt = now
					m.UpdatedAt = now
				}).Return(models.Meeting{
					ID:                uuid.New().String(),
					Title:             "Test Meeting",
					OrganizerID:       testUser.ID,
					Organizer:         &testUser,
					EstimatedDuration: 60,
					ProposedSlots: []models.TimeSlot{{
						ID:        uuid.New().String(),
						StartTime: testSlot.StartTime,
						EndTime:   testSlot.EndTime,
					}},
					Participants: []models.User{participant},
					CreatedAt:    now,
					UpdatedAt:    now,
				}, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "missing title",
			request: api.CreateMeetingRequest{
				OrganizerID:       testUser.ID,
				EstimatedDuration: 60,
				ProposedSlots:     []models.TimeSlot{testSlot},
				ParticipantIDs:    []string{uuid.New().String()},
			},
			setupMocks:     func(ur *MockUserRepository, mr *MockMeetingRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "organizer not found",
			request: api.CreateMeetingRequest{
				Title:             "Test Meeting",
				OrganizerID:       "non-existent",
				EstimatedDuration: 60,
				ProposedSlots:     []models.TimeSlot{testSlot},
				ParticipantIDs:    []string{uuid.New().String()},
			},
			setupMocks: func(ur *MockUserRepository, mr *MockMeetingRepository) {
				ur.On("GetByID", "non-existent").Return(models.User{}, errors.NewNotFoundError("User not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repositories
			mockUserRepo := new(MockUserRepository)
			mockMeetingRepo := new(MockMeetingRepository)
			tt.setupMocks(mockUserRepo, mockMeetingRepo)

			// Create handlers
			userHandler := &UserHandler{repository: mockUserRepo}
			handler := &MeetingHandler{
				repository:  mockMeetingRepo,
				userHandler: userHandler,
			}

			// Create request
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/meetings", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			// Handle request
			err := handler.CreateMeeting(w, req)

			// Check response
			if tt.expectedError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.expectedStatus, appErr.HTTPStatusCode())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, w.Code)

				var resp api.CreateMeetingResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.request.Title, resp.Meeting.Title)
				assert.Equal(t, tt.request.OrganizerID, resp.Meeting.OrganizerID)
				assert.Equal(t, tt.request.EstimatedDuration, resp.Meeting.EstimatedDuration)
				assert.Len(t, resp.Meeting.ProposedSlots, len(tt.request.ProposedSlots))
				assert.NotEmpty(t, resp.Meeting.ID)
				assert.NotEmpty(t, resp.Meeting.ProposedSlots[0].ID)
			}

			// Verify mock expectations
			mockUserRepo.AssertExpectations(t)
			mockMeetingRepo.AssertExpectations(t)
		})
	}
}

func TestGetRecommendations(t *testing.T) {
	// Create test data
	now := time.Now()
	meetingID := uuid.New().String()
	organizerID := uuid.New().String()
	organizer := models.User{
		ID:        organizerID,
		Name:      "Test Organizer",
		Email:     "organizer@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}
	participant := models.User{
		ID:        uuid.New().String(),
		Name:      "Test Participant",
		Email:     "participant@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}
	testSlot := models.TimeSlot{
		ID:        uuid.New().String(),
		StartTime: now,
		EndTime:   now.Add(time.Hour),
	}

	tests := []struct {
		name           string
		meetingID      string
		setupMocks     func(*MockMeetingRepository)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:      "successful recommendations",
			meetingID: meetingID,
			setupMocks: func(mr *MockMeetingRepository) {
				meeting := models.Meeting{
					ID:                meetingID,
					Title:             "Test Meeting",
					OrganizerID:       organizerID,
					Organizer:         &organizer,
					EstimatedDuration: 60,
					ProposedSlots:     []models.TimeSlot{testSlot},
					Participants:      []models.User{participant},
					CreatedAt:         now,
					UpdatedAt:         now,
				}
				mr.On("GetMeetingByID", meetingID).Return(meeting, nil)

				availabilities := []models.Availability{
					{
						ID:             uuid.New().String(),
						ParticipantID:  participant.ID,
						Participant:    &participant,
						MeetingID:      meetingID,
						AvailableSlots: []models.TimeSlot{testSlot},
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				}
				mr.On("GetMeetingAvailabilities", meetingID).Return(availabilities, nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:      "meeting not found",
			meetingID: "non-existent",
			setupMocks: func(mr *MockMeetingRepository) {
				mr.On("GetMeetingByID", "non-existent").Return(models.Meeting{}, errors.NewNotFoundError("Meeting not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "missing meeting ID",
			meetingID:      "",
			setupMocks:     func(mr *MockMeetingRepository) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := new(MockMeetingRepository)
			tt.setupMocks(mockRepo)

			// Create handler
			handler := &MeetingHandler{
				repository: mockRepo,
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/recommendations?meetingId="+tt.meetingID, nil)
			w := httptest.NewRecorder()

			// Handle request
			err := handler.GetRecommendations(w, req)

			// Check response
			if tt.expectedError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.expectedStatus, appErr.HTTPStatusCode())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, w.Code)

				var resp api.GetRecommendationsResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				assert.NoError(t, err)
				assert.NotNil(t, resp.RecommendedSlots)
				assert.Len(t, resp.RecommendedSlots, 1)
				assert.Equal(t, 1, resp.RecommendedSlots[0].AvailableCount)
				assert.Equal(t, 2, resp.RecommendedSlots[0].TotalParticipants) // organizer + 1 participant
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}
