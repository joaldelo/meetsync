package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"meetsync/internal/api"
	"meetsync/internal/models"
	"meetsync/pkg/logs"
)

// MeetingHandler handles meeting-related requests
type MeetingHandler struct {
	// In a real application, you would inject services or repositories here
	meetings       map[string]models.Meeting      // In-memory storage for demo purposes
	availabilities map[string]models.Availability // In-memory storage for demo purposes
	userHandler    *UserHandler                   // Reference to user handler for user lookups
}

// NewMeetingHandler creates a new MeetingHandler
func NewMeetingHandler(userHandler *UserHandler) *MeetingHandler {
	return &MeetingHandler{
		meetings:       make(map[string]models.Meeting),
		availabilities: make(map[string]models.Availability),
		userHandler:    userHandler,
	}
}

// CreateMeeting handles the creation of a new meeting
func (h *MeetingHandler) CreateMeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.CreateMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logs.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if req.EstimatedDuration <= 0 {
		http.Error(w, "Estimated duration must be positive", http.StatusBadRequest)
		return
	}
	if len(req.ProposedSlots) == 0 {
		http.Error(w, "At least one proposed time slot is required", http.StatusBadRequest)
		return
	}

	// Validate organizer exists
	organizer, exists := h.userHandler.users[req.OrganizerID]
	if !exists {
		http.Error(w, "Organizer not found", http.StatusBadRequest)
		return
	}

	// Validate participants exist
	var participants []models.User
	for _, participantID := range req.ParticipantIDs {
		participant, exists := h.userHandler.users[participantID]
		if !exists {
			http.Error(w, "Participant not found: "+participantID, http.StatusBadRequest)
			return
		}
		participants = append(participants, participant)
	}

	// Assign IDs to time slots
	for i := range req.ProposedSlots {
		req.ProposedSlots[i].ID = uuid.New().String()
	}

	// Create meeting
	now := time.Now()
	meeting := models.Meeting{
		ID:                uuid.New().String(),
		Title:             req.Title,
		OrganizerID:       req.OrganizerID,
		Organizer:         &organizer,
		EstimatedDuration: req.EstimatedDuration,
		ProposedSlots:     req.ProposedSlots,
		Participants:      participants,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// In a real application, you would save the meeting to a database
	h.meetings[meeting.ID] = meeting

	logs.Info("Created meeting: %s by organizer %s", meeting.ID, organizer.Name)

	// Return response
	resp := api.CreateMeetingResponse{
		Meeting: meeting,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logs.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// AddAvailability handles adding a participant's availability
func (h *MeetingHandler) AddAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req api.AddAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logs.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.UserID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}
	if req.MeetingID == "" {
		http.Error(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}
	if len(req.AvailableSlots) == 0 {
		http.Error(w, "At least one available time slot is required", http.StatusBadRequest)
		return
	}

	// Validate user exists
	user, exists := h.userHandler.users[req.UserID]
	if !exists {
		http.Error(w, "User not found", http.StatusBadRequest)
		return
	}

	// Validate meeting exists
	meeting, exists := h.meetings[req.MeetingID]
	if !exists {
		http.Error(w, "Meeting not found", http.StatusBadRequest)
		return
	}

	// Assign IDs to time slots
	for i := range req.AvailableSlots {
		req.AvailableSlots[i].ID = uuid.New().String()
	}

	// Create availability
	now := time.Now()
	availability := models.Availability{
		ID:             uuid.New().String(),
		ParticipantID:  req.UserID,
		Participant:    &user,
		MeetingID:      req.MeetingID,
		AvailableSlots: req.AvailableSlots,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// In a real application, you would save the availability to a database
	h.availabilities[availability.ID] = availability

	logs.Info("Added availability for user %s in meeting %s", user.Name, meeting.Title)

	w.WriteHeader(http.StatusCreated)
}

// GetRecommendations handles getting recommendations for a meeting
func (h *MeetingHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	meetingID := r.URL.Query().Get("meetingId")
	if meetingID == "" {
		http.Error(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}

	// Validate meeting exists
	meeting, exists := h.meetings[meetingID]
	if !exists {
		http.Error(w, "Meeting not found", http.StatusNotFound)
		return
	}

	// Get all availabilities for this meeting
	var meetingAvailabilities []models.Availability
	for _, availability := range h.availabilities {
		if availability.MeetingID == meetingID {
			meetingAvailabilities = append(meetingAvailabilities, availability)
		}
	}

	// In a real application, you would calculate recommendations based on actual availabilities
	// For now, we'll return a mock response

	// Get all participants
	participantMap := make(map[string]models.User)
	for _, participant := range meeting.Participants {
		participantMap[participant.ID] = participant
	}

	// Add organizer to participants if not already included
	if _, exists := participantMap[meeting.OrganizerID]; !exists {
		participantMap[meeting.OrganizerID] = *meeting.Organizer
	}

	// Mock recommendations
	mockRecommendations := []models.RecommendedSlot{
		{
			TimeSlot:          meeting.ProposedSlots[0],
			AvailableCount:    len(participantMap) - 1,
			TotalParticipants: len(participantMap),
			UnavailableParticipants: []models.User{
				meeting.Participants[0],
			},
		},
	}

	// If there's more than one proposed slot, add a perfect match
	if len(meeting.ProposedSlots) > 1 {
		mockRecommendations = append(mockRecommendations, models.RecommendedSlot{
			TimeSlot:                meeting.ProposedSlots[1],
			AvailableCount:          len(participantMap),
			TotalParticipants:       len(participantMap),
			UnavailableParticipants: []models.User{},
		})
	}

	resp := api.GetRecommendationsResponse{
		RecommendedSlots: mockRecommendations,
	}

	logs.Info("Generated recommendations for meeting %s (%s)", meeting.Title, meetingID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logs.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
