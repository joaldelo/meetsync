package repositories

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"meetsync/internal/models"
)

func createTestMeeting() models.Meeting {
	now := time.Now()
	return models.Meeting{
		ID:                uuid.New().String(),
		Title:             "Test Meeting",
		OrganizerID:       uuid.New().String(),
		EstimatedDuration: 60,
		ProposedSlots: []models.TimeSlot{
			{
				ID:        uuid.New().String(),
				StartTime: now,
				EndTime:   now.Add(time.Hour),
			},
		},
		Participants: []models.User{
			{
				ID:    uuid.New().String(),
				Name:  "Test Participant",
				Email: "participant@example.com",
			},
		},
	}
}

func TestInMemoryMeetingRepository_CreateMeeting(t *testing.T) {
	repo := NewInMemoryMeetingRepository()
	meeting := createTestMeeting()

	// Test successful creation
	created, err := repo.CreateMeeting(meeting)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, meeting.Title, created.Title)
	assert.Equal(t, meeting.OrganizerID, created.OrganizerID)
	assert.Equal(t, meeting.EstimatedDuration, created.EstimatedDuration)
	assert.Len(t, created.ProposedSlots, len(meeting.ProposedSlots))
	assert.NotEmpty(t, created.ProposedSlots[0].ID)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())
}

func TestInMemoryMeetingRepository_GetMeetingByID(t *testing.T) {
	repo := NewInMemoryMeetingRepository()
	meeting := createTestMeeting()

	// Add test meeting
	created, err := repo.CreateMeeting(meeting)
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing meeting",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "non-existent meeting",
			id:      "non-existent-id",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meeting, err := repo.GetMeetingByID(tt.id)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "Meeting not found")
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, created.ID, meeting.ID)
			assert.Equal(t, created.Title, meeting.Title)
		})
	}
}

func TestInMemoryMeetingRepository_UpdateMeeting(t *testing.T) {
	repo := NewInMemoryMeetingRepository()
	meeting := createTestMeeting()

	// Add test meeting
	created, err := repo.CreateMeeting(meeting)
	require.NoError(t, err)

	// Update meeting
	created.Title = "Updated Title"
	updated, err := repo.UpdateMeeting(created)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)

	// Try to update non-existent meeting
	nonExistent := createTestMeeting()
	_, err = repo.UpdateMeeting(nonExistent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Meeting not found")
}

func TestInMemoryMeetingRepository_DeleteMeeting(t *testing.T) {
	repo := NewInMemoryMeetingRepository()
	meeting := createTestMeeting()

	// Add test meeting
	created, err := repo.CreateMeeting(meeting)
	require.NoError(t, err)

	// Create availability for the meeting
	availability := models.Availability{
		ID:             uuid.New().String(),
		ParticipantID:  created.Participants[0].ID,
		MeetingID:      created.ID,
		AvailableSlots: []models.TimeSlot{created.ProposedSlots[0]},
	}
	_, err = repo.CreateAvailability(availability)
	require.NoError(t, err)

	// Test deletion
	err = repo.DeleteMeeting(created.ID)
	assert.NoError(t, err)

	// Verify meeting is deleted
	_, err = repo.GetMeetingByID(created.ID)
	assert.Error(t, err)

	// Verify associated availabilities are deleted
	availabilities := repo.GetAllAvailabilities()
	assert.Empty(t, availabilities)

	// Try to delete non-existent meeting
	err = repo.DeleteMeeting("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Meeting not found")
}

func TestInMemoryMeetingRepository_Availability(t *testing.T) {
	repo := NewInMemoryMeetingRepository()
	meeting := createTestMeeting()

	// Create test meeting
	created, err := repo.CreateMeeting(meeting)
	require.NoError(t, err)

	// Test creating availability
	availability := models.Availability{
		ParticipantID:  created.Participants[0].ID,
		MeetingID:      created.ID,
		AvailableSlots: []models.TimeSlot{created.ProposedSlots[0]},
	}
	createdAvail, err := repo.CreateAvailability(availability)
	assert.NoError(t, err)
	assert.NotEmpty(t, createdAvail.ID)
	assert.False(t, createdAvail.CreatedAt.IsZero())
	assert.False(t, createdAvail.UpdatedAt.IsZero())

	// Test getting availability
	foundAvail, err := repo.GetAvailability(availability.ParticipantID, availability.MeetingID)
	assert.NoError(t, err)
	assert.Equal(t, createdAvail.ID, foundAvail.ID)

	// Test updating availability
	createdAvail.AvailableSlots = []models.TimeSlot{}
	updatedAvail, err := repo.UpdateAvailability(createdAvail)
	assert.NoError(t, err)
	assert.Empty(t, updatedAvail.AvailableSlots)

	// Test getting meeting availabilities
	meetingAvails, err := repo.GetMeetingAvailabilities(created.ID)
	assert.NoError(t, err)
	assert.Len(t, meetingAvails, 1)

	// Test deleting availability
	err = repo.DeleteAvailability(createdAvail.ID)
	assert.NoError(t, err)

	// Verify availability is deleted
	meetingAvails, err = repo.GetMeetingAvailabilities(created.ID)
	assert.NoError(t, err)
	assert.Empty(t, meetingAvails)
}

func TestInMemoryMeetingRepository_ConcurrentOperations(t *testing.T) {
	repo := NewInMemoryMeetingRepository()
	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent meeting creations
	wg.Add(numGoroutines)
	meetings := make([]models.Meeting, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			meeting := createTestMeeting()
			created, err := repo.CreateMeeting(meeting)
			assert.NoError(t, err)
			meetings[i] = created
		}(i)
	}
	wg.Wait()

	// Test concurrent availability creations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			defer wg.Done()
			availability := models.Availability{
				ParticipantID:  meetings[i].Participants[0].ID,
				MeetingID:      meetings[i].ID,
				AvailableSlots: []models.TimeSlot{meetings[i].ProposedSlots[0]},
			}
			_, err := repo.CreateAvailability(availability)
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Verify all meetings and availabilities were created
	allAvails := repo.GetAllAvailabilities()
	assert.Len(t, allAvails, numGoroutines)
}
