package models

import (
	"time"
)

// TimeSlot represents a time slot for a meeting
type TimeSlot struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// Meeting represents a meeting with multiple time slots
type Meeting struct {
	ID                string     `json:"id"`
	Title             string     `json:"title"`
	OrganizerID       string     `json:"organizerId"`
	Organizer         *User      `json:"organizer,omitempty"`
	EstimatedDuration int        `json:"estimatedDuration"` // in minutes
	ProposedSlots     []TimeSlot `json:"proposedSlots"`
	Participants      []User     `json:"participants,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

// Participant represents a participant in a meeting
type Participant struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	User      *User     `json:"user,omitempty"`
	MeetingID string    `json:"meetingId"`
	JoinedAt  time.Time `json:"joinedAt"`
}

// Availability represents a participant's availability for a meeting
type Availability struct {
	ID             string     `json:"id"`
	ParticipantID  string     `json:"participantId"`
	Participant    *User      `json:"participant,omitempty"`
	MeetingID      string     `json:"meetingId"`
	AvailableSlots []TimeSlot `json:"availableSlots"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// RecommendedSlot represents a recommended time slot for a meeting
type RecommendedSlot struct {
	TimeSlot                TimeSlot `json:"timeSlot"`
	AvailableCount          int      `json:"availableCount"`
	TotalParticipants       int      `json:"totalParticipants"`
	UnavailableParticipants []User   `json:"unavailableParticipants,omitempty"`
}

// CreateMeetingRequest represents the request to create a meeting
type CreateMeetingRequest struct {
	Title             string     `json:"title"`
	OrganizerID       string     `json:"organizerId"`
	EstimatedDuration int        `json:"estimatedDuration"` // in minutes
	ProposedSlots     []TimeSlot `json:"proposedSlots"`
	ParticipantIDs    []string   `json:"participantIds,omitempty"`
}

// CreateMeetingResponse represents the response after creating a meeting
type CreateMeetingResponse struct {
	Meeting Meeting `json:"meeting"`
}

// AddParticipantRequest represents the request to add a participant to a meeting
type AddParticipantRequest struct {
	UserID    string `json:"userId"`
	MeetingID string `json:"meetingId"`
}

// AddAvailabilityRequest represents the request to add availability
type AddAvailabilityRequest struct {
	UserID         string     `json:"userId"`
	MeetingID      string     `json:"meetingId"`
	AvailableSlots []TimeSlot `json:"availableSlots"`
}

// GetRecommendationsRequest represents the request to get recommendations
type GetRecommendationsRequest struct {
	MeetingID string `json:"meetingId"`
}

// GetRecommendationsResponse represents the response with recommendations
type GetRecommendationsResponse struct {
	RecommendedSlots []RecommendedSlot `json:"recommendedSlots"`
}
