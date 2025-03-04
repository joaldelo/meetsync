package api

import (
	"meetsync/internal/models"
)

// CreateMeetingRequest represents the request to create a meeting
type CreateMeetingRequest struct {
	Title             string            `json:"title"`
	OrganizerID       string            `json:"organizerId"`
	EstimatedDuration int               `json:"estimatedDuration"` // in minutes
	ProposedSlots     []models.TimeSlot `json:"proposedSlots"`
	ParticipantIDs    []string          `json:"participantIds,omitempty"`
}

// CreateMeetingResponse represents the response after creating a meeting
type CreateMeetingResponse struct {
	Meeting models.Meeting `json:"meeting"`
}

// AddParticipantRequest represents the request to add a participant to a meeting
type AddParticipantRequest struct {
	UserID    string `json:"userId"`
	MeetingID string `json:"meetingId"`
}

// AddAvailabilityRequest represents the request to add availability
type AddAvailabilityRequest struct {
	UserID         string            `json:"userId"`
	MeetingID      string            `json:"meetingId"`
	AvailableSlots []models.TimeSlot `json:"availableSlots"`
}

// GetRecommendationsRequest represents the request to get recommendations
type GetRecommendationsRequest struct {
	MeetingID string `json:"meetingId"`
}

// GetRecommendationsResponse represents the response with recommendations
type GetRecommendationsResponse struct {
	RecommendedSlots []models.RecommendedSlot `json:"recommendedSlots"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserResponse represents the response after creating a user
type CreateUserResponse struct {
	User models.User `json:"user"`
}

// GetUserResponse represents the response when fetching a user
type GetUserResponse struct {
	User models.User `json:"user"`
}

// ListUsersResponse represents the response when listing users
type ListUsersResponse struct {
	Users []models.User `json:"users"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// UpdateMeetingRequest represents the request to update a meeting
type UpdateMeetingRequest struct {
	Title             string            `json:"title,omitempty"`
	EstimatedDuration int               `json:"estimatedDuration,omitempty"`
	ProposedSlots     []models.TimeSlot `json:"proposedSlots,omitempty"`
	ParticipantIDs    []string          `json:"participantIds,omitempty"`
}

// UpdateMeetingResponse represents the response after updating a meeting
type UpdateMeetingResponse struct {
	Meeting models.Meeting `json:"meeting"`
}

// UpdateAvailabilityRequest represents the request to update availability
type UpdateAvailabilityRequest struct {
	AvailableSlots []models.TimeSlot `json:"availableSlots"`
}

// UpdateAvailabilityResponse represents the response after updating availability
type UpdateAvailabilityResponse struct {
	Availability models.Availability `json:"availability"`
}

// GetAvailabilityResponse represents the response when getting availability
type GetAvailabilityResponse struct {
	Availability models.Availability `json:"availability"`
}
