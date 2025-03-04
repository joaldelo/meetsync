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
