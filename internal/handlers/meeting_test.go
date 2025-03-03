package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meetsync/internal/api"
	"meetsync/internal/models"
)

func TestCreateMeeting(t *testing.T) {
	// Create handlers
	userHandler := NewUserHandler()
	meetingHandler := NewMeetingHandler(userHandler)

	// Create a test user (organizer)
	organizer := models.User{
		ID:    "organizer-id",
		Name:  "Organizer",
		Email: "organizer@example.com",
	}
	userHandler.users[organizer.ID] = organizer

	// Create test participants
	participant1 := models.User{
		ID:    "participant1-id",
		Name:  "Participant 1",
		Email: "participant1@example.com",
	}
	userHandler.users[participant1.ID] = participant1

	participant2 := models.User{
		ID:    "participant2-id",
		Name:  "Participant 2",
		Email: "participant2@example.com",
	}
	userHandler.users[participant2.ID] = participant2

	// Test cases
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	dayAfterTomorrow := now.Add(48 * time.Hour)

	tests := []struct {
		name           string
		requestBody    api.CreateMeetingRequest
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid meeting creation",
			requestBody: api.CreateMeetingRequest{
				Title:             "Team Meeting",
				OrganizerID:       organizer.ID,
				EstimatedDuration: 60,
				ProposedSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
					{
						StartTime: dayAfterTomorrow,
						EndTime:   dayAfterTomorrow.Add(time.Hour),
					},
				},
				ParticipantIDs: []string{participant1.ID, participant2.ID},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp api.CreateMeetingResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.Meeting.Title != "Team Meeting" {
					t.Errorf("Expected title 'Team Meeting', got '%s'", resp.Meeting.Title)
				}
				if resp.Meeting.OrganizerID != organizer.ID {
					t.Errorf("Expected organizer ID '%s', got '%s'", organizer.ID, resp.Meeting.OrganizerID)
				}
				if resp.Meeting.EstimatedDuration != 60 {
					t.Errorf("Expected duration 60, got %d", resp.Meeting.EstimatedDuration)
				}
				if len(resp.Meeting.ProposedSlots) != 2 {
					t.Errorf("Expected 2 proposed slots, got %d", len(resp.Meeting.ProposedSlots))
				}
				if len(resp.Meeting.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(resp.Meeting.Participants))
				}
			},
		},
		{
			name: "Missing title",
			requestBody: api.CreateMeetingRequest{
				OrganizerID:       organizer.ID,
				EstimatedDuration: 60,
				ProposedSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Title is required\n" {
					t.Errorf("Expected error message 'Title is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Invalid duration",
			requestBody: api.CreateMeetingRequest{
				Title:             "Team Meeting",
				OrganizerID:       organizer.ID,
				EstimatedDuration: 0,
				ProposedSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Estimated duration must be positive\n" {
					t.Errorf("Expected error message 'Estimated duration must be positive', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "No proposed slots",
			requestBody: api.CreateMeetingRequest{
				Title:             "Team Meeting",
				OrganizerID:       organizer.ID,
				EstimatedDuration: 60,
				ProposedSlots:     []models.TimeSlot{},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "At least one proposed time slot is required\n" {
					t.Errorf("Expected error message 'At least one proposed time slot is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Invalid organizer",
			requestBody: api.CreateMeetingRequest{
				Title:             "Team Meeting",
				OrganizerID:       "non-existent-id",
				EstimatedDuration: 60,
				ProposedSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Organizer not found\n" {
					t.Errorf("Expected error message 'Organizer not found', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Invalid participant",
			requestBody: api.CreateMeetingRequest{
				Title:             "Team Meeting",
				OrganizerID:       organizer.ID,
				EstimatedDuration: 60,
				ProposedSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
				ParticipantIDs: []string{participant1.ID, "non-existent-id"},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Participant not found: non-existent-id\n" {
					t.Errorf("Expected error message 'Participant not found: non-existent-id', got '%s'", w.Body.String())
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
			req, err := http.NewRequest(http.MethodPost, "/api/meetings", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			meetingHandler.CreateMeeting(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// Run validation function
			tc.validateFunc(t, w)
		})
	}
}

func TestAddAvailability(t *testing.T) {
	// Create handlers
	userHandler := NewUserHandler()
	meetingHandler := NewMeetingHandler(userHandler)

	// Create a test user
	user := models.User{
		ID:    "user-id",
		Name:  "Test User",
		Email: "user@example.com",
	}
	userHandler.users[user.ID] = user

	// Create a test meeting
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	dayAfterTomorrow := now.Add(48 * time.Hour)

	meeting := models.Meeting{
		ID:                "meeting-id",
		Title:             "Test Meeting",
		OrganizerID:       user.ID,
		Organizer:         &user,
		EstimatedDuration: 60,
		ProposedSlots: []models.TimeSlot{
			{
				ID:        "slot1",
				StartTime: tomorrow,
				EndTime:   tomorrow.Add(time.Hour),
			},
			{
				ID:        "slot2",
				StartTime: dayAfterTomorrow,
				EndTime:   dayAfterTomorrow.Add(time.Hour),
			},
		},
		Participants: []models.User{user},
	}
	meetingHandler.meetings[meeting.ID] = meeting

	// Test cases
	tests := []struct {
		name           string
		requestBody    api.AddAvailabilityRequest
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid availability",
			requestBody: api.AddAvailabilityRequest{
				UserID:    user.ID,
				MeetingID: meeting.ID,
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// No response body to validate
			},
		},
		{
			name: "Missing user ID",
			requestBody: api.AddAvailabilityRequest{
				MeetingID: meeting.ID,
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "User ID is required\n" {
					t.Errorf("Expected error message 'User ID is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Missing meeting ID",
			requestBody: api.AddAvailabilityRequest{
				UserID: user.ID,
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Meeting ID is required\n" {
					t.Errorf("Expected error message 'Meeting ID is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "No available slots",
			requestBody: api.AddAvailabilityRequest{
				UserID:         user.ID,
				MeetingID:      meeting.ID,
				AvailableSlots: []models.TimeSlot{},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "At least one available time slot is required\n" {
					t.Errorf("Expected error message 'At least one available time slot is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Invalid user",
			requestBody: api.AddAvailabilityRequest{
				UserID:    "non-existent-id",
				MeetingID: meeting.ID,
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "User not found\n" {
					t.Errorf("Expected error message 'User not found', got '%s'", w.Body.String())
				}
			},
		},
		{
			name: "Invalid meeting",
			requestBody: api.AddAvailabilityRequest{
				UserID:    user.ID,
				MeetingID: "non-existent-id",
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Meeting not found\n" {
					t.Errorf("Expected error message 'Meeting not found', got '%s'", w.Body.String())
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
			req, err := http.NewRequest(http.MethodPost, "/api/availabilities", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			meetingHandler.AddAvailability(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// Run validation function
			tc.validateFunc(t, w)
		})
	}
}

func TestGetRecommendations(t *testing.T) {
	// Create handlers
	userHandler := NewUserHandler()
	meetingHandler := NewMeetingHandler(userHandler)

	// Create a test user
	user := models.User{
		ID:    "user-id",
		Name:  "Test User",
		Email: "user@example.com",
	}
	userHandler.users[user.ID] = user

	// Create a test meeting
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	dayAfterTomorrow := now.Add(48 * time.Hour)

	meeting := models.Meeting{
		ID:                "meeting-id",
		Title:             "Test Meeting",
		OrganizerID:       user.ID,
		Organizer:         &user,
		EstimatedDuration: 60,
		ProposedSlots: []models.TimeSlot{
			{
				ID:        "slot1",
				StartTime: tomorrow,
				EndTime:   tomorrow.Add(time.Hour),
			},
			{
				ID:        "slot2",
				StartTime: dayAfterTomorrow,
				EndTime:   dayAfterTomorrow.Add(time.Hour),
			},
		},
		Participants: []models.User{user},
	}
	meetingHandler.meetings[meeting.ID] = meeting

	// Test cases
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Valid meeting ID",
			meetingID:      meeting.ID,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp api.GetRecommendationsResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(resp.RecommendedSlots) == 0 {
					t.Error("Expected at least one recommended slot")
				}
			},
		},
		{
			name:           "Missing meeting ID",
			meetingID:      "",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Meeting ID is required\n" {
					t.Errorf("Expected error message 'Meeting ID is required', got '%s'", w.Body.String())
				}
			},
		},
		{
			name:           "Invalid meeting ID",
			meetingID:      "non-existent-id",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Body.String() != "Meeting not found\n" {
					t.Errorf("Expected error message 'Meeting not found', got '%s'", w.Body.String())
				}
			},
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest(http.MethodGet, "/api/recommendations?meetingId="+tc.meetingID, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			meetingHandler.GetRecommendations(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			// Run validation function
			tc.validateFunc(t, w)
		})
	}
}
