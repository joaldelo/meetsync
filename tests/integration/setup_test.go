package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"meetsync/internal/api"
	"meetsync/internal/models"
)

const baseURL = "http://localhost:8080"

// TestUser represents a simplified user response
type TestUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TestSetup contains common test utilities and state
type TestSetup struct {
	t *testing.T
	// Track created resources for cleanup
	createdUsers    []string
	createdMeetings []string
}

// NewTestSetup creates a new test setup
func NewTestSetup(t *testing.T) *TestSetup {
	return &TestSetup{
		t:               t,
		createdUsers:    make([]string, 0),
		createdMeetings: make([]string, 0),
	}
}

// Helper function to generate unique test emails
func (ts *TestSetup) generateTestEmail(prefix string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s.%d@test.meetsync.local", prefix, timestamp)
}

// Helper function to make HTTP requests
func (ts *TestSetup) makeRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// CreateUser creates a test user
func (ts *TestSetup) CreateUser(name, emailPrefix string) (*TestUser, error) {
	email := ts.generateTestEmail(emailPrefix)
	req := api.CreateUserRequest{
		Name:  name,
		Email: email,
	}

	respBody, err := ts.makeRequest(http.MethodPost, "/api/users", req)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User TestUser `json:"user"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	ts.createdUsers = append(ts.createdUsers, resp.User.ID)
	return &resp.User, nil
}

// CreateMeeting creates a test meeting
func (ts *TestSetup) CreateMeeting(title string, organizerID string, participantIDs []string, slots []models.TimeSlot) (string, error) {
	req := api.CreateMeetingRequest{
		Title:             title,
		OrganizerID:       organizerID,
		EstimatedDuration: 60, // 1 hour
		ProposedSlots:     slots,
		ParticipantIDs:    participantIDs,
	}

	respBody, err := ts.makeRequest(http.MethodPost, "/api/meetings", req)
	if err != nil {
		return "", err
	}

	var resp struct {
		Meeting struct {
			ID string `json:"id"`
		} `json:"meeting"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	ts.createdMeetings = append(ts.createdMeetings, resp.Meeting.ID)
	return resp.Meeting.ID, nil
}

// AddAvailability adds availability for a user
func (ts *TestSetup) AddAvailability(userID, meetingID string, slots []models.TimeSlot) error {
	req := api.AddAvailabilityRequest{
		UserID:         userID,
		MeetingID:      meetingID,
		AvailableSlots: slots,
	}

	_, err := ts.makeRequest(http.MethodPost, "/api/availabilities", req)
	return err
}

// GetRecommendations gets meeting recommendations
func (ts *TestSetup) GetRecommendations(meetingID string) (*api.GetRecommendationsResponse, error) {
	respBody, err := ts.makeRequest(http.MethodGet, fmt.Sprintf("/api/recommendations?meetingId=%s", meetingID), nil)
	if err != nil {
		return nil, err
	}

	var resp api.GetRecommendationsResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &resp, nil
}

// Cleanup removes all resources created during the test
func (ts *TestSetup) Cleanup() {
	// Since we don't have DELETE endpoints implemented yet, just log the resources that would be cleaned up
	if len(ts.createdMeetings) > 0 {
		ts.t.Logf("Would cleanup meetings: %v", ts.createdMeetings)
	}
	if len(ts.createdUsers) > 0 {
		ts.t.Logf("Would cleanup users: %v", ts.createdUsers)
	}

	// Clear the tracking slices
	ts.createdMeetings = nil
	ts.createdUsers = nil
}

func TestMain(m *testing.M) {
	// Setup code before running tests
	// For example, ensure the server is running, set up test database, etc.

	// Run tests
	code := m.Run()

	// Cleanup code after running tests
	os.Exit(code)
}
