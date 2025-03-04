package router

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

func TestRouterSetup(t *testing.T) {
	// Create a new router
	r := New()
	r.Setup()

	// Create some test data
	now := time.Now()

	// First create a user and get their ID
	createUserReq := api.CreateUserRequest{
		Name:  "Test User",
		Email: "test@example.com",
	}
	createUserBody := mustMarshal(createUserReq)
	req, _ := http.NewRequest(http.MethodPost, "/api/users", bytes.NewBuffer(createUserBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test user: status %d", w.Code)
	}

	var createUserResp api.CreateUserResponse
	if err := json.NewDecoder(w.Body).Decode(&createUserResp); err != nil {
		t.Fatalf("Failed to decode create user response: %v", err)
	}
	validUserID := createUserResp.User.ID

	// Create a meeting with the user as organizer
	createMeetingReq := api.CreateMeetingRequest{
		Title:             "Test Meeting",
		OrganizerID:       validUserID,
		EstimatedDuration: 60,
		ProposedSlots: []models.TimeSlot{
			{
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
		},
	}
	createMeetingBody := mustMarshal(createMeetingReq)
	req, _ = http.NewRequest(http.MethodPost, "/api/meetings", bytes.NewBuffer(createMeetingBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create test meeting: status %d", w.Code)
	}

	var createMeetingResp api.CreateMeetingResponse
	if err := json.NewDecoder(w.Body).Decode(&createMeetingResp); err != nil {
		t.Fatalf("Failed to decode create meeting response: %v", err)
	}
	validMeetingID := createMeetingResp.Meeting.ID

	// Test cases
	tests := []struct {
		name           string
		method         string
		path           string
		body           []byte
		expectedStatus int
	}{
		{
			name:           "GET /api/users - List users",
			method:         http.MethodGet,
			path:           "/api/users",
			body:           nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST /api/users - Create user",
			method: http.MethodPost,
			path:   "/api/users",
			body: mustMarshal(api.CreateUserRequest{
				Name:  "Another User",
				Email: "another@example.com",
			}),
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "GET /api/users/{id} - Get user (not found)",
			method:         http.MethodGet,
			path:           "/api/users/non-existent-id",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "POST /api/meetings - Create meeting (invalid)",
			method: http.MethodPost,
			path:   "/api/meetings",
			body: mustMarshal(api.CreateMeetingRequest{
				Title:             "Test Meeting",
				OrganizerID:       "non-existent-id",
				EstimatedDuration: 60,
				ProposedSlots:     []models.TimeSlot{}, // Empty slots
			}),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST /api/availabilities - Add availability (validation error)",
			method: http.MethodPost,
			path:   "/api/availabilities",
			body: mustMarshal(api.AddAvailabilityRequest{
				UserID:    validUserID,
				MeetingID: validMeetingID,
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: now.Add(time.Hour), // End time before start time
						EndTime:   now,
					},
				},
			}),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST /api/availabilities - Add availability (not found)",
			method: http.MethodPost,
			path:   "/api/availabilities",
			body: mustMarshal(api.AddAvailabilityRequest{
				UserID:    "non-existent-id",
				MeetingID: "non-existent-id",
				AvailableSlots: []models.TimeSlot{
					{
						StartTime: now,
						EndTime:   now.Add(time.Hour),
					},
				},
			}),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "GET /api/recommendations - Get recommendations (invalid)",
			method:         http.MethodGet,
			path:           "/api/recommendations?meetingId=non-existent-id",
			body:           nil,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Method not allowed",
			method:         http.MethodDelete,
			path:           "/api/users",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	// Run test cases
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest(tc.method, tc.path, bytes.NewBuffer(tc.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if tc.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(w, req)

			// Check status code
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

// Helper function to marshal JSON
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
