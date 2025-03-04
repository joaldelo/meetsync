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
	"meetsync/pkg/errors"
)

// Helper function to validate error response
func validateMeetingErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedMessage string, expectedStatus int) {
	var errResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Message != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, errResp.Error.Message)
	}
	if w.Code != expectedStatus {
		t.Errorf("Expected status code %d, got %d", expectedStatus, w.Code)
	}
}

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
				validateMeetingErrorResponse(t, w, "Title is required", http.StatusBadRequest)
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
				validateMeetingErrorResponse(t, w, "Estimated duration must be positive", http.StatusBadRequest)
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
				validateMeetingErrorResponse(t, w, "At least one proposed time slot is required", http.StatusBadRequest)
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
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Organizer not found", http.StatusNotFound)
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
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Participant not found: non-existent-id", http.StatusNotFound)
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
			err = meetingHandler.CreateMeeting(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
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
				validateMeetingErrorResponse(t, w, "User ID is required", http.StatusBadRequest)
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
				validateMeetingErrorResponse(t, w, "Meeting ID is required", http.StatusBadRequest)
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
				validateMeetingErrorResponse(t, w, "At least one available time slot is required", http.StatusBadRequest)
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
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "User not found", http.StatusNotFound)
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
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Meeting not found", http.StatusNotFound)
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
			err = meetingHandler.AddAvailability(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
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

	// Create additional participants
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
		Participants: []models.User{participant1, participant2},
	}
	meetingHandler.meetings[meeting.ID] = meeting

	// Add availability for participants
	// User (organizer) is available for both slots
	availability1 := models.Availability{
		ID:            "availability1-id",
		ParticipantID: user.ID,
		Participant:   &user,
		MeetingID:     meeting.ID,
		AvailableSlots: []models.TimeSlot{
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
	}
	meetingHandler.availabilities[availability1.ID] = availability1

	// Participant 1 is available for first slot only
	availability2 := models.Availability{
		ID:            "availability2-id",
		ParticipantID: participant1.ID,
		Participant:   &participant1,
		MeetingID:     meeting.ID,
		AvailableSlots: []models.TimeSlot{
			{
				ID:        "slot1",
				StartTime: tomorrow,
				EndTime:   tomorrow.Add(time.Hour),
			},
		},
	}
	meetingHandler.availabilities[availability2.ID] = availability2

	// Participant 2 is available for second slot only
	availability3 := models.Availability{
		ID:            "availability3-id",
		ParticipantID: participant2.ID,
		Participant:   &participant2,
		MeetingID:     meeting.ID,
		AvailableSlots: []models.TimeSlot{
			{
				ID:        "slot2",
				StartTime: dayAfterTomorrow,
				EndTime:   dayAfterTomorrow.Add(time.Hour),
			},
		},
	}
	meetingHandler.availabilities[availability3.ID] = availability3

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

				// Verify the recommendations
				if len(resp.RecommendedSlots) != 2 {
					t.Errorf("Expected 2 recommended slots, got %d", len(resp.RecommendedSlots))
				}

				// First slot should have 2 available participants (organizer and participant1)
				firstSlot := resp.RecommendedSlots[0]
				if firstSlot.AvailableCount != 2 {
					t.Errorf("Expected 2 available participants for first slot, got %d", firstSlot.AvailableCount)
				}
				if firstSlot.TotalParticipants != 3 {
					t.Errorf("Expected 3 total participants (2 participants + organizer), got %d", firstSlot.TotalParticipants)
				}
				if len(firstSlot.UnavailableParticipants) != 1 {
					t.Errorf("Expected 1 unavailable participant for first slot, got %d", len(firstSlot.UnavailableParticipants))
				}

				// Second slot should have 2 available participants (organizer and participant2)
				secondSlot := resp.RecommendedSlots[1]
				if secondSlot.AvailableCount != 2 {
					t.Errorf("Expected 2 available participants for second slot, got %d", secondSlot.AvailableCount)
				}
				if secondSlot.TotalParticipants != 3 {
					t.Errorf("Expected 3 total participants (2 participants + organizer), got %d", secondSlot.TotalParticipants)
				}
				if len(secondSlot.UnavailableParticipants) != 1 {
					t.Errorf("Expected 1 unavailable participant for second slot, got %d", len(secondSlot.UnavailableParticipants))
				}
			},
		},
		{
			name:           "Missing meeting ID",
			meetingID:      "",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Meeting ID is required", http.StatusBadRequest)
			},
		},
		{
			name:           "Invalid meeting ID",
			meetingID:      "non-existent-id",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Meeting not found", http.StatusNotFound)
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
			err = meetingHandler.GetRecommendations(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
			}

			// Run validation function
			tc.validateFunc(t, w)
		})
	}
}

func TestUpdateMeeting(t *testing.T) {
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

	// Create a test meeting
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	meeting := models.Meeting{
		ID:                "meeting-id",
		Title:             "Original Meeting",
		OrganizerID:       organizer.ID,
		Organizer:         &organizer,
		EstimatedDuration: 60,
		ProposedSlots: []models.TimeSlot{
			{
				ID:        "slot1",
				StartTime: tomorrow,
				EndTime:   tomorrow.Add(time.Hour),
			},
		},
		Participants: []models.User{participant1},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	meetingHandler.meetings[meeting.ID] = meeting

	tests := []struct {
		name           string
		meetingID      string
		requestBody    api.UpdateMeetingRequest
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:      "Valid update",
			meetingID: meeting.ID,
			requestBody: api.UpdateMeetingRequest{
				Title:             "Updated Meeting",
				EstimatedDuration: 90,
				ParticipantIDs:    []string{participant1.ID, participant2.ID},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp api.UpdateMeetingResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if resp.Meeting.Title != "Updated Meeting" {
					t.Errorf("Expected title 'Updated Meeting', got '%s'", resp.Meeting.Title)
				}
				if resp.Meeting.EstimatedDuration != 90 {
					t.Errorf("Expected duration 90, got %d", resp.Meeting.EstimatedDuration)
				}
				if len(resp.Meeting.Participants) != 2 {
					t.Errorf("Expected 2 participants, got %d", len(resp.Meeting.Participants))
				}
			},
		},
		{
			name:      "Meeting not found",
			meetingID: "non-existent-id",
			requestBody: api.UpdateMeetingRequest{
				Title: "Updated Meeting",
			},
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Meeting not found", http.StatusNotFound)
			},
		},
		{
			name:      "Invalid participant",
			meetingID: meeting.ID,
			requestBody: api.UpdateMeetingRequest{
				ParticipantIDs: []string{"non-existent-id"},
			},
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Participant not found: non-existent-id", http.StatusNotFound)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, err := http.NewRequest(http.MethodPut, "/api/meetings/"+tc.meetingID, bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			err = meetingHandler.UpdateMeeting(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
			}

			tc.validateFunc(t, w)
		})
	}
}

func TestDeleteMeeting(t *testing.T) {
	tests := []struct {
		name           string
		meetingID      string
		expectedStatus int
		validateFunc   func(*testing.T, *MeetingHandler)
	}{
		{
			name:           "Valid deletion",
			meetingID:      "meeting-id",
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, h *MeetingHandler) {
				// Verify meeting is deleted
				if _, exists := h.meetings["meeting-id"]; exists {
					t.Error("Meeting should be deleted")
				}
				// Verify associated availability is deleted
				if _, exists := h.availabilities["availability-id"]; exists {
					t.Error("Associated availability should be deleted")
				}
			},
		},
		{
			name:           "Meeting not found",
			meetingID:      "non-existent-id",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, h *MeetingHandler) {
				// No changes should be made
				if _, exists := h.meetings["meeting-id"]; !exists {
					t.Error("Existing meeting should not be affected")
				}
				if _, exists := h.availabilities["availability-id"]; !exists {
					t.Error("Existing availability should not be affected")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh handlers for each test case
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
			meeting := models.Meeting{
				ID:                "meeting-id",
				Title:             "Test Meeting",
				OrganizerID:       user.ID,
				Organizer:         &user,
				EstimatedDuration: 60,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			meetingHandler.meetings[meeting.ID] = meeting

			// Create test availability
			availability := models.Availability{
				ID:            "availability-id",
				ParticipantID: user.ID,
				Participant:   &user,
				MeetingID:     meeting.ID,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			meetingHandler.availabilities[availability.ID] = availability

			req, err := http.NewRequest(http.MethodDelete, "/api/meetings/"+tc.meetingID, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			err = meetingHandler.DeleteMeeting(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			tc.validateFunc(t, meetingHandler)
		})
	}
}

func TestUpdateAvailability(t *testing.T) {
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
	meeting := models.Meeting{
		ID:                "meeting-id",
		Title:             "Test Meeting",
		OrganizerID:       user.ID,
		Organizer:         &user,
		EstimatedDuration: 60,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	meetingHandler.meetings[meeting.ID] = meeting

	// Create test availability
	availability := models.Availability{
		ID:            "availability-id",
		ParticipantID: user.ID,
		Participant:   &user,
		MeetingID:     meeting.ID,
		AvailableSlots: []models.TimeSlot{
			{
				ID:        "slot1",
				StartTime: tomorrow,
				EndTime:   tomorrow.Add(time.Hour),
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	meetingHandler.availabilities[availability.ID] = availability

	tests := []struct {
		name           string
		availabilityID string
		requestBody    api.UpdateAvailabilityRequest
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Valid update",
			availabilityID: availability.ID,
			requestBody: api.UpdateAvailabilityRequest{
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow.Add(2 * time.Hour),
						EndTime:   tomorrow.Add(3 * time.Hour),
					},
				},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp api.UpdateAvailabilityResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				if err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}
				if len(resp.Availability.AvailableSlots) != 1 {
					t.Errorf("Expected 1 available slot, got %d", len(resp.Availability.AvailableSlots))
				}
			},
		},
		{
			name:           "Availability not found",
			availabilityID: "non-existent-id",
			requestBody: api.UpdateAvailabilityRequest{
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: tomorrow,
						EndTime:   tomorrow.Add(time.Hour),
					},
				},
			},
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "Availability not found", http.StatusNotFound)
			},
		},
		{
			name:           "No available slots",
			availabilityID: availability.ID,
			requestBody: api.UpdateAvailabilityRequest{
				AvailableSlots: []models.TimeSlot{},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				validateMeetingErrorResponse(t, w, "At least one available time slot is required", http.StatusBadRequest)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			req, err := http.NewRequest(http.MethodPut, "/api/availabilities/"+tc.availabilityID, bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			err = meetingHandler.UpdateAvailability(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
			}

			tc.validateFunc(t, w)
		})
	}
}

func TestDeleteAvailability(t *testing.T) {
	tests := []struct {
		name           string
		availabilityID string
		expectedStatus int
		validateFunc   func(*testing.T, *MeetingHandler)
	}{
		{
			name:           "Valid deletion",
			availabilityID: "availability-id",
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, h *MeetingHandler) {
				if _, exists := h.availabilities["availability-id"]; exists {
					t.Error("Availability should be deleted")
				}
			},
		},
		{
			name:           "Availability not found",
			availabilityID: "non-existent-id",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, h *MeetingHandler) {
				if _, exists := h.availabilities["availability-id"]; !exists {
					t.Error("Existing availability should not be affected")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh handlers for each test case
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
			meeting := models.Meeting{
				ID:                "meeting-id",
				Title:             "Test Meeting",
				OrganizerID:       user.ID,
				Organizer:         &user,
				EstimatedDuration: 60,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			meetingHandler.meetings[meeting.ID] = meeting

			// Create test availability
			availability := models.Availability{
				ID:            "availability-id",
				ParticipantID: user.ID,
				Participant:   &user,
				MeetingID:     meeting.ID,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			meetingHandler.availabilities[availability.ID] = availability

			req, err := http.NewRequest(http.MethodDelete, "/api/availabilities/"+tc.availabilityID, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			err = meetingHandler.DeleteAvailability(w, req)
			if err != nil {
				// Error should be handled by middleware in production
				if appErr, ok := err.(*errors.AppError); ok {
					w.WriteHeader(appErr.HTTPStatusCode())
					errors.WriteError(w, appErr)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					errors.WriteError(w, errors.NewInternalError("Internal server error", err))
				}
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}

			tc.validateFunc(t, meetingHandler)
		})
	}
}
