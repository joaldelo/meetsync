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
	"meetsync/internal/interfaces"
	"meetsync/internal/models"
	"meetsync/pkg/errors"
)

// timeSlotMatcher is a custom matcher for TimeSlot arrays that ignores monotonic clock details
type timeSlotMatcher struct {
	expected []models.TimeSlot
}

func (m timeSlotMatcher) Matches(x interface{}) bool {
	slots, ok := x.([]models.TimeSlot)
	if !ok {
		return false
	}
	if len(slots) != len(m.expected) {
		return false
	}
	for i, slot := range slots {
		if slot.ID != m.expected[i].ID {
			return false
		}
		// Compare times using Unix timestamps to ignore monotonic clock
		if slot.StartTime.Unix() != m.expected[i].StartTime.Unix() {
			return false
		}
		if slot.EndTime.Unix() != m.expected[i].EndTime.Unix() {
			return false
		}
	}
	return true
}

func (m timeSlotMatcher) String() string {
	return "matches time slots ignoring monotonic clock"
}

// MockMeetingService is a mock implementation of MeetingService
type MockMeetingService struct {
	mock.Mock
}

var _ interfaces.MeetingService = (*MockMeetingService)(nil) // Verify MockMeetingService implements MeetingService interface

func (m *MockMeetingService) CreateMeeting(title string, organizerID string, estimatedDuration int, proposedSlots []models.TimeSlot, participantIDs []string) (models.Meeting, error) {
	args := m.Called(title, organizerID, estimatedDuration, mock.MatchedBy(func(slots []models.TimeSlot) bool {
		return timeSlotMatcher{proposedSlots}.Matches(slots)
	}), participantIDs)
	return args.Get(0).(models.Meeting), args.Error(1)
}

func (m *MockMeetingService) GetRecommendations(meetingID string) ([]models.RecommendedSlot, error) {
	args := m.Called(meetingID)
	return args.Get(0).([]models.RecommendedSlot), args.Error(1)
}

func (m *MockMeetingService) UpdateMeeting(meetingID string, title string, estimatedDuration int, proposedSlots []models.TimeSlot, participantIDs []string) (models.Meeting, error) {
	args := m.Called(meetingID, title, estimatedDuration, proposedSlots, participantIDs)
	return args.Get(0).(models.Meeting), args.Error(1)
}

func (m *MockMeetingService) DeleteMeeting(meetingID string) error {
	args := m.Called(meetingID)
	return args.Error(0)
}

func (m *MockMeetingService) AddAvailability(userID string, meetingID string, availableSlots []models.TimeSlot) (models.Availability, error) {
	args := m.Called(userID, meetingID, availableSlots)
	return args.Get(0).(models.Availability), args.Error(1)
}

func (m *MockMeetingService) UpdateAvailability(availabilityID string, availableSlots []models.TimeSlot) (models.Availability, error) {
	args := m.Called(availabilityID, availableSlots)
	return args.Get(0).(models.Availability), args.Error(1)
}

func (m *MockMeetingService) DeleteAvailability(availabilityID string) error {
	args := m.Called(availabilityID)
	return args.Error(0)
}

func (m *MockMeetingService) GetAvailability(userID string, meetingID string) (models.Availability, error) {
	args := m.Called(userID, meetingID)
	return args.Get(0).(models.Availability), args.Error(1)
}

func TestCreateMeeting(t *testing.T) {
	// Create test data
	now := time.Now().Truncate(time.Second) // Truncate to remove sub-second precision
	testUser := models.User{
		ID:        uuid.New().String(),
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	testSlot := models.TimeSlot{
		ID:        uuid.New().String(),
		StartTime: now,
		EndTime:   now.Add(time.Hour),
	}

	participantID := uuid.New().String()
	participant := models.User{
		ID:        participantID,
		Name:      "Participant",
		Email:     "participant@example.com",
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name           string
		request        api.CreateMeetingRequest
		setupMock      func(*MockMeetingService)
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
				ParticipantIDs:    []string{participantID},
			},
			setupMock: func(m *MockMeetingService) {
				meeting := models.Meeting{
					ID:                uuid.New().String(),
					Title:             "Test Meeting",
					OrganizerID:       testUser.ID,
					Organizer:         &testUser,
					EstimatedDuration: 60,
					ProposedSlots:     []models.TimeSlot{testSlot},
					Participants:      []models.User{participant},
					CreatedAt:         now,
					UpdatedAt:         now,
				}
				m.On("CreateMeeting",
					"Test Meeting",
					testUser.ID,
					60,
					mock.Anything, // Use mock.Anything for the slots parameter
					[]string{participantID},
				).Return(meeting, nil)
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
				ParticipantIDs:    []string{participantID},
			},
			setupMock: func(m *MockMeetingService) {
				m.On("CreateMeeting",
					"",
					testUser.ID,
					60,
					mock.Anything, // Use mock.Anything for the slots parameter
					[]string{participantID},
				).Return(models.Meeting{}, errors.NewValidationError("Title is required", ""))
			},
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
				ParticipantIDs:    []string{participantID},
			},
			setupMock: func(m *MockMeetingService) {
				m.On("CreateMeeting",
					"Test Meeting",
					"non-existent",
					60,
					mock.Anything, // Use mock.Anything for the slots parameter
					[]string{participantID},
				).Return(models.Meeting{}, errors.NewNotFoundError("Organizer not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMeetingService)
			tt.setupMock(mockService)

			// Create handler with mock service
			handler := &MeetingHandler{service: mockService}

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
				assert.Equal(t, testSlot.ID, resp.Meeting.ProposedSlots[0].ID)
				assert.Equal(t, testSlot.StartTime, resp.Meeting.ProposedSlots[0].StartTime)
				assert.Equal(t, testSlot.EndTime, resp.Meeting.ProposedSlots[0].EndTime)
				assert.NotEmpty(t, resp.Meeting.ID)
				assert.NotNil(t, resp.Meeting.Organizer)
				assert.Equal(t, testUser.ID, resp.Meeting.Organizer.ID)
				assert.Equal(t, testUser.Name, resp.Meeting.Organizer.Name)
				assert.Equal(t, testUser.Email, resp.Meeting.Organizer.Email)
				assert.Len(t, resp.Meeting.Participants, 1)
				assert.Equal(t, participant.ID, resp.Meeting.Participants[0].ID)
				assert.Equal(t, participant.Name, resp.Meeting.Participants[0].Name)
				assert.Equal(t, participant.Email, resp.Meeting.Participants[0].Email)
			}

			// Verify mock expectations
			mockService.AssertExpectations(t)
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
	testSlot := models.TimeSlot{
		ID:        uuid.New().String(),
		StartTime: now,
		EndTime:   now.Add(time.Hour),
	}

	tests := []struct {
		name           string
		meetingID      string
		setupMock      func(*MockMeetingService)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:      "successful recommendations",
			meetingID: meetingID,
			setupMock: func(m *MockMeetingService) {
				recommendations := []models.RecommendedSlot{
					{
						TimeSlot:                testSlot,
						AvailableCount:          1,
						TotalParticipants:       2,
						UnavailableParticipants: []models.User{organizer},
					},
				}
				m.On("GetRecommendations", meetingID).Return(recommendations, nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:      "meeting not found",
			meetingID: "non-existent",
			setupMock: func(m *MockMeetingService) {
				m.On("GetRecommendations", "non-existent").Return([]models.RecommendedSlot{}, errors.NewNotFoundError("Meeting not found"))
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  true,
		},
		{
			name:           "missing meeting ID",
			meetingID:      "",
			setupMock:      func(m *MockMeetingService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMeetingService)
			tt.setupMock(mockService)

			// Create handler with mock service
			handler := &MeetingHandler{service: mockService}

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
				assert.Equal(t, 2, resp.RecommendedSlots[0].TotalParticipants)
			}

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}
