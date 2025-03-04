package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
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

	// Get all participants who have submitted availability
	participantMap := make(map[string]models.User)
	for _, availability := range meetingAvailabilities {
		if user, exists := h.userHandler.users[availability.ParticipantID]; exists {
			participantMap[user.ID] = user
		}
	}

	// Calculate recommendations based on actual availabilities
	var recommendations []models.RecommendedSlot
	for _, slot := range meeting.ProposedSlots {
		availableParticipants := make(map[string]bool)
		unavailableParticipants := []models.User{}

		// Count available participants for this slot
		for _, availability := range meetingAvailabilities {
			for _, availableSlot := range availability.AvailableSlots {
				if availableSlot.StartTime.Equal(slot.StartTime) && availableSlot.EndTime.Equal(slot.EndTime) {
					availableParticipants[availability.ParticipantID] = true
					logs.Info("Found available participant %s for slot %v-%v",
						availability.ParticipantID,
						availableSlot.StartTime.Format("15:04"),
						availableSlot.EndTime.Format("15:04"))
					break
				}
			}
		}

		// Identify unavailable participants
		for participantID, participant := range participantMap {
			if !availableParticipants[participantID] {
				unavailableParticipants = append(unavailableParticipants, participant)
			}
		}

		rec := models.RecommendedSlot{
			TimeSlot:                slot,
			AvailableCount:          len(availableParticipants),
			TotalParticipants:       len(participantMap),
			UnavailableParticipants: unavailableParticipants,
		}
		logs.Info("Adding recommendation for slot %v-%v: available=%d total=%d",
			slot.StartTime.Format("15:04"),
			slot.EndTime.Format("15:04"),
			rec.AvailableCount,
			rec.TotalParticipants)
		recommendations = append(recommendations, rec)
	}

	// Sort recommendations by available count (highest to lowest)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].AvailableCount > recommendations[j].AvailableCount
	})

	for i, rec := range recommendations {
		logs.Info("After sorting - Slot[%d]: %v-%v available=%d",
			i,
			rec.TimeSlot.StartTime.Format("15:04"),
			rec.TimeSlot.EndTime.Format("15:04"),
			rec.AvailableCount)
	}

	resp := api.GetRecommendationsResponse{
		RecommendedSlots: recommendations,
	}

	logs.Info("Generated recommendations for meeting %s (%s)", meeting.Title, meetingID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logs.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// UpdateMeeting handles updating an existing meeting
func (h *MeetingHandler) UpdateMeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract meeting ID from URL path
	meetingID := r.URL.Path[len("/api/meetings/"):]
	if meetingID == "" {
		http.Error(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}

	// Check if meeting exists
	meeting, exists := h.meetings[meetingID]
	if !exists {
		http.Error(w, "Meeting not found", http.StatusNotFound)
		return
	}

	var req api.UpdateMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logs.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update fields if provided
	if req.Title != "" {
		meeting.Title = req.Title
	}
	if req.EstimatedDuration > 0 {
		meeting.EstimatedDuration = req.EstimatedDuration
	}
	if len(req.ProposedSlots) > 0 {
		// Assign IDs to new time slots
		for i := range req.ProposedSlots {
			req.ProposedSlots[i].ID = uuid.New().String()
		}
		meeting.ProposedSlots = req.ProposedSlots
	}
	if len(req.ParticipantIDs) > 0 {
		// Validate and update participants
		var participants []models.User
		for _, participantID := range req.ParticipantIDs {
			participant, exists := h.userHandler.users[participantID]
			if !exists {
				http.Error(w, "Participant not found: "+participantID, http.StatusBadRequest)
				return
			}
			participants = append(participants, participant)
		}
		meeting.Participants = participants
	}

	meeting.UpdatedAt = time.Now()
	h.meetings[meetingID] = meeting

	resp := api.UpdateMeetingResponse{
		Meeting: meeting,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logs.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// DeleteMeeting handles deleting an existing meeting
func (h *MeetingHandler) DeleteMeeting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract meeting ID from URL path
	meetingID := r.URL.Path[len("/api/meetings/"):]
	if meetingID == "" {
		http.Error(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}

	// Check if meeting exists
	if _, exists := h.meetings[meetingID]; !exists {
		http.Error(w, "Meeting not found", http.StatusNotFound)
		return
	}

	// Delete meeting and associated availabilities
	delete(h.meetings, meetingID)

	// Delete all availabilities for this meeting
	for id, availability := range h.availabilities {
		if availability.MeetingID == meetingID {
			delete(h.availabilities, id)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateAvailability handles updating a participant's availability
func (h *MeetingHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract availability ID from URL path
	availabilityID := r.URL.Path[len("/api/availabilities/"):]
	if availabilityID == "" {
		http.Error(w, "Availability ID is required", http.StatusBadRequest)
		return
	}

	// Check if availability exists
	availability, exists := h.availabilities[availabilityID]
	if !exists {
		http.Error(w, "Availability not found", http.StatusNotFound)
		return
	}

	var req api.UpdateAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logs.Error("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.AvailableSlots) == 0 {
		http.Error(w, "At least one available time slot is required", http.StatusBadRequest)
		return
	}

	// Assign IDs to new time slots
	for i := range req.AvailableSlots {
		req.AvailableSlots[i].ID = uuid.New().String()
	}

	// Update availability
	availability.AvailableSlots = req.AvailableSlots
	availability.UpdatedAt = time.Now()
	h.availabilities[availabilityID] = availability

	resp := api.UpdateAvailabilityResponse{
		Availability: availability,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logs.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// DeleteAvailability handles deleting a participant's availability
func (h *MeetingHandler) DeleteAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract availability ID from URL path
	availabilityID := r.URL.Path[len("/api/availabilities/"):]
	if availabilityID == "" {
		http.Error(w, "Availability ID is required", http.StatusBadRequest)
		return
	}

	// Check if availability exists
	if _, exists := h.availabilities[availabilityID]; !exists {
		http.Error(w, "Availability not found", http.StatusNotFound)
		return
	}

	// Delete availability
	delete(h.availabilities, availabilityID)

	w.WriteHeader(http.StatusNoContent)
}

// GetAvailability handles getting a participant's availability
func (h *MeetingHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get query parameters
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	meetingID := r.URL.Query().Get("meetingId")
	if meetingID == "" {
		http.Error(w, "Meeting ID is required", http.StatusBadRequest)
		return
	}

	// Find availability for the user and meeting
	var foundAvailability models.Availability
	found := false
	for _, availability := range h.availabilities {
		if availability.ParticipantID == userID && availability.MeetingID == meetingID {
			foundAvailability = availability
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Availability not found", http.StatusNotFound)
		return
	}

	resp := api.GetAvailabilityResponse{
		Availability: foundAvailability,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logs.Error("Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
