package services

import (
	"testing"
	"time"

	"meetsync/internal/models"
	"meetsync/pkg/errors"

	"github.com/stretchr/testify/assert"
)

func setupTestMeetingService(t *testing.T) (*MeetingServiceImpl, models.User, []models.User) {
	userService := NewUserService()

	// Create organizer
	organizer, err := userService.CreateUser("Organizer", "organizer@example.com")
	assert.NoError(t, err)

	// Create participants
	participants := make([]models.User, 0)
	participantData := []struct {
		name  string
		email string
	}{
		{"Participant 1", "participant1@example.com"},
		{"Participant 2", "participant2@example.com"},
	}

	for _, p := range participantData {
		participant, err := userService.CreateUser(p.name, p.email)
		assert.NoError(t, err)
		participants = append(participants, participant)
	}

	return NewMeetingService(userService).(*MeetingServiceImpl), organizer, participants
}

func createTestTimeSlots() []models.TimeSlot {
	now := time.Now()
	return []models.TimeSlot{
		{
			StartTime: now.Add(24 * time.Hour),
			EndTime:   now.Add(25 * time.Hour),
		},
		{
			StartTime: now.Add(48 * time.Hour),
			EndTime:   now.Add(49 * time.Hour),
		},
	}
}

func TestMeetingService_CreateMeeting(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	tests := []struct {
		name              string
		title             string
		organizerID       string
		estimatedDuration int
		proposedSlots     []models.TimeSlot
		participantIDs    []string
		expectError       bool
		errorMessage      string
	}{
		{
			name:              "Valid meeting creation",
			title:             "Team Meeting",
			organizerID:       organizer.ID,
			estimatedDuration: 60,
			proposedSlots:     timeSlots,
			participantIDs:    participantIDs,
		},
		{
			name:              "Empty title",
			organizerID:       organizer.ID,
			estimatedDuration: 60,
			proposedSlots:     timeSlots,
			participantIDs:    participantIDs,
			expectError:       true,
			errorMessage:      "Title is required",
		},
		{
			name:              "Invalid duration",
			title:             "Team Meeting",
			organizerID:       organizer.ID,
			estimatedDuration: 0,
			proposedSlots:     timeSlots,
			participantIDs:    participantIDs,
			expectError:       true,
			errorMessage:      "Estimated duration must be positive",
		},
		{
			name:              "No time slots",
			title:             "Team Meeting",
			organizerID:       organizer.ID,
			estimatedDuration: 60,
			proposedSlots:     []models.TimeSlot{},
			participantIDs:    participantIDs,
			expectError:       true,
			errorMessage:      "At least one proposed time slot is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meeting, err := service.CreateMeeting(
				tt.title,
				tt.organizerID,
				tt.estimatedDuration,
				tt.proposedSlots,
				tt.participantIDs,
			)

			if tt.expectError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.errorMessage, appErr.Message)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, meeting.ID)
				assert.Equal(t, tt.title, meeting.Title)
				assert.Equal(t, tt.organizerID, meeting.OrganizerID)
				assert.Equal(t, tt.estimatedDuration, meeting.EstimatedDuration)
				assert.Len(t, meeting.ProposedSlots, len(tt.proposedSlots))
				assert.Len(t, meeting.Participants, len(tt.participantIDs))
			}
		})
	}
}

func TestMeetingService_AddAvailability(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	// Create a test meeting first
	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	meeting, err := service.CreateMeeting(
		"Test Meeting",
		organizer.ID,
		60,
		timeSlots,
		participantIDs,
	)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		userID         string
		meetingID      string
		availableSlots []models.TimeSlot
		expectError    bool
	}{
		{
			name:           "Valid availability",
			userID:         participants[0].ID,
			meetingID:      meeting.ID,
			availableSlots: timeSlots,
		},
		{
			name:           "Invalid user",
			userID:         "non-existing-user",
			meetingID:      meeting.ID,
			availableSlots: timeSlots,
			expectError:    true,
		},
		{
			name:           "Invalid meeting",
			userID:         participants[0].ID,
			meetingID:      "non-existing-meeting",
			availableSlots: timeSlots,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			availability, err := service.AddAvailability(
				tt.userID,
				tt.meetingID,
				tt.availableSlots,
			)

			if tt.expectError {
				assert.Error(t, err)
				appErr, ok := err.(*errors.AppError)
				assert.True(t, ok)
				if tt.userID == "non-existing-user" {
					assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, availability.ID)
				assert.Equal(t, tt.userID, availability.ParticipantID)
				assert.Equal(t, tt.meetingID, availability.MeetingID)
				assert.Len(t, availability.AvailableSlots, len(tt.availableSlots))
			}
		})
	}
}

func TestMeetingService_GetRecommendations(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	// Create a test meeting
	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	meeting, err := service.CreateMeeting(
		"Test Meeting",
		organizer.ID,
		60,
		timeSlots,
		participantIDs,
	)
	assert.NoError(t, err)

	// Add availabilities for participants
	for _, participant := range participants {
		_, err := service.AddAvailability(
			participant.ID,
			meeting.ID,
			timeSlots,
		)
		assert.NoError(t, err)
	}

	// Test getting recommendations
	recommendations, err := service.GetRecommendations(meeting.ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, recommendations)

	// Verify recommendations are sorted by available count
	for i := 1; i < len(recommendations); i++ {
		assert.GreaterOrEqual(t,
			recommendations[i-1].AvailableCount,
			recommendations[i].AvailableCount,
			"Recommendations should be sorted by available count in descending order",
		)
	}
}

func TestMeetingService_UpdateMeeting(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	// Create a test meeting first
	meeting, err := service.CreateMeeting(
		"Original Meeting",
		organizer.ID,
		60,
		timeSlots,
		participantIDs,
	)
	assert.NoError(t, err)

	// Create new time slots for update
	newTimeSlots := []models.TimeSlot{
		{
			StartTime: time.Now().Add(72 * time.Hour),
			EndTime:   time.Now().Add(73 * time.Hour),
		},
	}

	tests := []struct {
		name              string
		meetingID         string
		title             string
		estimatedDuration int
		proposedSlots     []models.TimeSlot
		participantIDs    []string
		expectError       bool
		errorMessage      string
	}{
		{
			name:              "Valid update",
			meetingID:         meeting.ID,
			title:             "Updated Meeting",
			estimatedDuration: 90,
			proposedSlots:     newTimeSlots,
			participantIDs:    participantIDs,
		},
		{
			name:              "Non-existing meeting",
			meetingID:         "non-existing-id",
			title:             "Updated Meeting",
			estimatedDuration: 90,
			proposedSlots:     newTimeSlots,
			participantIDs:    participantIDs,
			expectError:       true,
			errorMessage:      "Meeting not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedMeeting, err := service.UpdateMeeting(
				tt.meetingID,
				tt.title,
				tt.estimatedDuration,
				tt.proposedSlots,
				tt.participantIDs,
			)

			if tt.expectError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.errorMessage, appErr.Message)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.title, updatedMeeting.Title)
				assert.Equal(t, tt.estimatedDuration, updatedMeeting.EstimatedDuration)
				assert.Len(t, updatedMeeting.ProposedSlots, len(tt.proposedSlots))
				assert.Len(t, updatedMeeting.Participants, len(tt.participantIDs))
			}
		})
	}
}

func TestMeetingService_DeleteMeeting(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	// Create a test meeting first
	meeting, err := service.CreateMeeting(
		"Test Meeting",
		organizer.ID,
		60,
		timeSlots,
		participantIDs,
	)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		meetingID   string
		expectError bool
	}{
		{
			name:      "Valid deletion",
			meetingID: meeting.ID,
		},
		{
			name:        "Non-existing meeting",
			meetingID:   "non-existing-id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteMeeting(tt.meetingID)

			if tt.expectError {
				assert.Error(t, err)
				appErr, ok := err.(*errors.AppError)
				assert.True(t, ok)
				assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
			} else {
				assert.NoError(t, err)

				// Verify meeting is deleted
				_, err := service.GetRecommendations(tt.meetingID)
				assert.Error(t, err)
				appErr, ok := err.(*errors.AppError)
				assert.True(t, ok)
				assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
			}
		})
	}
}

func TestMeetingService_UpdateAvailability(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	// Create a test meeting and availability first
	meeting, err := service.CreateMeeting(
		"Test Meeting",
		organizer.ID,
		60,
		timeSlots,
		[]string{participants[0].ID},
	)
	assert.NoError(t, err)

	availability, err := service.AddAvailability(
		participants[0].ID,
		meeting.ID,
		timeSlots[:1], // Only add first time slot initially
	)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		availabilityID string
		availableSlots []models.TimeSlot
		expectError    bool
		errorMessage   string
	}{
		{
			name:           "Valid update",
			availabilityID: availability.ID,
			availableSlots: timeSlots, // Update to include all time slots
		},
		{
			name:           "Non-existing availability",
			availabilityID: "non-existing-id",
			availableSlots: timeSlots,
			expectError:    true,
			errorMessage:   "Availability not found",
		},
		{
			name:           "Empty time slots",
			availabilityID: availability.ID,
			availableSlots: []models.TimeSlot{},
			expectError:    true,
			errorMessage:   "At least one available time slot is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedAvailability, err := service.UpdateAvailability(
				tt.availabilityID,
				tt.availableSlots,
			)

			if tt.expectError {
				assert.Error(t, err)
				if appErr, ok := err.(*errors.AppError); ok {
					assert.Equal(t, tt.errorMessage, appErr.Message)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.availabilityID, updatedAvailability.ID)
				assert.Len(t, updatedAvailability.AvailableSlots, len(tt.availableSlots))
			}
		})
	}
}

func TestMeetingService_DeleteAvailability(t *testing.T) {
	service, organizer, participants := setupTestMeetingService(t)
	timeSlots := createTestTimeSlots()

	// Create a test meeting and availability first
	meeting, err := service.CreateMeeting(
		"Test Meeting",
		organizer.ID,
		60,
		timeSlots,
		[]string{participants[0].ID},
	)
	assert.NoError(t, err)

	availability, err := service.AddAvailability(
		participants[0].ID,
		meeting.ID,
		timeSlots,
	)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		availabilityID string
		expectError    bool
	}{
		{
			name:           "Valid deletion",
			availabilityID: availability.ID,
		},
		{
			name:           "Non-existing availability",
			availabilityID: "non-existing-id",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteAvailability(tt.availabilityID)

			if tt.expectError {
				assert.Error(t, err)
				appErr, ok := err.(*errors.AppError)
				assert.True(t, ok)
				assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
			} else {
				assert.NoError(t, err)

				// Verify availability is deleted
				_, err := service.GetAvailability(participants[0].ID, meeting.ID)
				assert.Error(t, err)
				appErr, ok := err.(*errors.AppError)
				assert.True(t, ok)
				assert.Equal(t, errors.ErrorTypeNotFound, appErr.Type)
			}
		})
	}
}
