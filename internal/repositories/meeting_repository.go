package repositories

import (
	"time"

	"github.com/google/uuid"

	"meetsync/internal/models"
	"meetsync/pkg/errors"
)

// MeetingRepository defines the interface for meeting data access
type MeetingRepository interface {
	CreateMeeting(meeting models.Meeting) (models.Meeting, error)
	GetMeetingByID(id string) (models.Meeting, error)
	UpdateMeeting(meeting models.Meeting) (models.Meeting, error)
	DeleteMeeting(id string) error
	CreateAvailability(availability models.Availability) (models.Availability, error)
	GetAvailability(userID, meetingID string) (models.Availability, error)
	UpdateAvailability(availability models.Availability) (models.Availability, error)
	DeleteAvailability(id string) error
	GetMeetingAvailabilities(meetingID string) ([]models.Availability, error)
	GetAllAvailabilities() []models.Availability
}

// InMemoryMeetingRepository implements MeetingRepository using in-memory storage
type InMemoryMeetingRepository struct {
	meetings       map[string]models.Meeting
	availabilities map[string]models.Availability
}

// NewInMemoryMeetingRepository creates a new InMemoryMeetingRepository
func NewInMemoryMeetingRepository() *InMemoryMeetingRepository {
	return &InMemoryMeetingRepository{
		meetings:       make(map[string]models.Meeting),
		availabilities: make(map[string]models.Availability),
	}
}

func (r *InMemoryMeetingRepository) CreateMeeting(meeting models.Meeting) (models.Meeting, error) {
	// Set timestamps and ID
	now := time.Now()
	meeting.CreatedAt = now
	meeting.UpdatedAt = now
	if meeting.ID == "" {
		meeting.ID = uuid.New().String()
	}

	// Assign IDs to time slots if not already assigned
	for i := range meeting.ProposedSlots {
		if meeting.ProposedSlots[i].ID == "" {
			meeting.ProposedSlots[i].ID = uuid.New().String()
		}
	}

	r.meetings[meeting.ID] = meeting
	return meeting, nil
}

func (r *InMemoryMeetingRepository) GetMeetingByID(id string) (models.Meeting, error) {
	meeting, exists := r.meetings[id]
	if !exists {
		return models.Meeting{}, errors.NewNotFoundError("Meeting not found")
	}
	return meeting, nil
}

func (r *InMemoryMeetingRepository) UpdateMeeting(meeting models.Meeting) (models.Meeting, error) {
	if _, exists := r.meetings[meeting.ID]; !exists {
		return models.Meeting{}, errors.NewNotFoundError("Meeting not found")
	}

	meeting.UpdatedAt = time.Now()
	r.meetings[meeting.ID] = meeting
	return meeting, nil
}

func (r *InMemoryMeetingRepository) DeleteMeeting(id string) error {
	if _, exists := r.meetings[id]; !exists {
		return errors.NewNotFoundError("Meeting not found")
	}

	delete(r.meetings, id)

	// Delete associated availabilities
	for availID, avail := range r.availabilities {
		if avail.MeetingID == id {
			delete(r.availabilities, availID)
		}
	}

	return nil
}

func (r *InMemoryMeetingRepository) CreateAvailability(availability models.Availability) (models.Availability, error) {
	// Set timestamps and ID
	now := time.Now()
	availability.CreatedAt = now
	availability.UpdatedAt = now
	if availability.ID == "" {
		availability.ID = uuid.New().String()
	}

	r.availabilities[availability.ID] = availability
	return availability, nil
}

func (r *InMemoryMeetingRepository) GetAvailability(userID, meetingID string) (models.Availability, error) {
	for _, availability := range r.availabilities {
		if availability.ParticipantID == userID && availability.MeetingID == meetingID {
			return availability, nil
		}
	}
	return models.Availability{}, errors.NewNotFoundError("Availability not found")
}

func (r *InMemoryMeetingRepository) UpdateAvailability(availability models.Availability) (models.Availability, error) {
	if _, exists := r.availabilities[availability.ID]; !exists {
		return models.Availability{}, errors.NewNotFoundError("Availability not found")
	}

	availability.UpdatedAt = time.Now()
	r.availabilities[availability.ID] = availability
	return availability, nil
}

func (r *InMemoryMeetingRepository) DeleteAvailability(id string) error {
	if _, exists := r.availabilities[id]; !exists {
		return errors.NewNotFoundError("Availability not found")
	}

	delete(r.availabilities, id)
	return nil
}

func (r *InMemoryMeetingRepository) GetMeetingAvailabilities(meetingID string) ([]models.Availability, error) {
	var availabilities []models.Availability
	for _, availability := range r.availabilities {
		if availability.MeetingID == meetingID {
			availabilities = append(availabilities, availability)
		}
	}
	return availabilities, nil
}

func (r *InMemoryMeetingRepository) GetAllAvailabilities() []models.Availability {
	availabilities := make([]models.Availability, 0, len(r.availabilities))
	for _, a := range r.availabilities {
		availabilities = append(availabilities, a)
	}
	return availabilities
}
