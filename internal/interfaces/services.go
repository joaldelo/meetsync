package interfaces

import "meetsync/internal/models"

// UserService defines the interface for user-related business logic
type UserService interface {
	CreateUser(name, email string) (models.User, error)
	GetUserByID(userID string) (models.User, error)
	ListUsers() ([]models.User, error)
}

// MeetingService defines the interface for meeting-related business logic
type MeetingService interface {
	CreateMeeting(title string, organizerID string, estimatedDuration int, proposedSlots []models.TimeSlot, participantIDs []string) (models.Meeting, error)
	GetRecommendations(meetingID string) ([]models.RecommendedSlot, error)
	UpdateMeeting(meetingID string, title string, estimatedDuration int, proposedSlots []models.TimeSlot, participantIDs []string) (models.Meeting, error)
	DeleteMeeting(meetingID string) error
	AddAvailability(userID string, meetingID string, availableSlots []models.TimeSlot) (models.Availability, error)
	UpdateAvailability(availabilityID string, availableSlots []models.TimeSlot) (models.Availability, error)
	DeleteAvailability(availabilityID string) error
	GetAvailability(userID string, meetingID string) (models.Availability, error)
}
