package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meetsync/internal/api"
	"meetsync/internal/models"
)

func TestRouterSetup(t *testing.T) {
	// Create a new router
	r := New()
	r.Setup()

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
				Name:  "Test User",
				Email: "test@example.com",
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
			name:   "POST /api/availabilities - Add availability (invalid)",
			method: http.MethodPost,
			path:   "/api/availabilities",
			body: mustMarshal(api.AddAvailabilityRequest{
				UserID:         "non-existent-id",
				MeetingID:      "non-existent-id",
				AvailableSlots: []models.TimeSlot{},
			}),
			expectedStatus: http.StatusBadRequest,
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
