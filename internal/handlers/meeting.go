package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"

	"meetsync/internal/api"
	"meetsync/internal/models"
	"meetsync/pkg/errors"
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
func (h *MeetingHandler) CreateMeeting(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.NewValidationError("Method not allowed", "Only POST method is allowed")
	}

	var req api.CreateMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
	}

	// Validate request
	if req.Title == "" {
		return errors.NewValidationError("Title is required", "")
	}
	if req.EstimatedDuration <= 0 {
		return errors.NewValidationError("Estimated duration must be positive", "")
	}
	if len(req.ProposedSlots) == 0 {
		return errors.NewValidationError("At least one proposed time slot is required", "")
	}

	// Validate organizer exists
	organizer, exists := h.userHandler.users[req.OrganizerID]
	if !exists {
		return errors.NewNotFoundError("Organizer not found")
	}

	// Validate participants exist
	var participants []models.User
	for _, participantID := range req.ParticipantIDs {
		participant, exists := h.userHandler.users[participantID]
		if !exists {
			return errors.NewNotFoundError("Participant not found: " + participantID)
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

	// Save meeting
	h.meetings[meeting.ID] = meeting

	logs.Info("Created meeting: %s by organizer %s", meeting.ID, organizer.Name)

	// Return response
	resp := api.CreateMeetingResponse{
		Meeting: meeting,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// AddAvailability handles adding a participant's availability
func (h *MeetingHandler) AddAvailability(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errors.NewValidationError("Method not allowed", "Only POST method is allowed")
	}

	var req api.AddAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
	}

	// Validate request
	if req.UserID == "" {
		return errors.NewValidationError("User ID is required", "")
	}
	if req.MeetingID == "" {
		return errors.NewValidationError("Meeting ID is required", "")
	}
	if len(req.AvailableSlots) == 0 {
		return errors.NewValidationError("At least one available time slot is required", "")
	}

	// Validate user exists
	user, exists := h.userHandler.users[req.UserID]
	if !exists {
		return errors.NewNotFoundError("User not found")
	}

	// Validate meeting exists
	meeting, exists := h.meetings[req.MeetingID]
	if !exists {
		return errors.NewNotFoundError("Meeting not found")
	}

	// Match available slots with proposed slots and assign correct IDs
	var matchedSlots []models.TimeSlot
	for _, availableSlot := range req.AvailableSlots {
		matched := false
		for _, proposedSlot := range meeting.ProposedSlots {
			if availableSlot.StartTime.Equal(proposedSlot.StartTime) &&
				availableSlot.EndTime.Equal(proposedSlot.EndTime) {
				matchedSlots = append(matchedSlots, proposedSlot)
				matched = true
				break
			}
		}
		if !matched {
			return errors.NewValidationError("Available slot does not match any proposed slot", "")
		}
	}

	// Create availability
	now := time.Now()
	availability := models.Availability{
		ID:             uuid.New().String(),
		ParticipantID:  req.UserID,
		Participant:    &user,
		MeetingID:      req.MeetingID,
		AvailableSlots: matchedSlots,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Save availability
	h.availabilities[availability.ID] = availability

	logs.Info("Added availability for user %s in meeting %s", user.Name, meeting.Title)

	w.WriteHeader(http.StatusCreated)
	return nil
}

// GetRecommendations handles getting recommendations for a meeting
func (h *MeetingHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewValidationError("Method not allowed", "Only GET method is allowed")
	}

	meetingID := r.URL.Query().Get("meetingId")
	if meetingID == "" {
		return errors.NewValidationError("Meeting ID is required", "")
	}

	// Validate meeting exists
	meeting, exists := h.meetings[meetingID]
	if !exists {
		return errors.NewNotFoundError("Meeting not found")
	}

	// Get all availabilities for this meeting
	var meetingAvailabilities []models.Availability
	for _, availability := range h.availabilities {
		if availability.MeetingID == meetingID {
			meetingAvailabilities = append(meetingAvailabilities, availability)
		}
	}

	// Calculate recommendations
	recommendations := h.calculateRecommendations(meeting, meetingAvailabilities)

	resp := api.GetRecommendationsResponse{
		RecommendedSlots: recommendations,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// UpdateMeeting handles updating an existing meeting
func (h *MeetingHandler) UpdateMeeting(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPut {
		return errors.NewValidationError("Method not allowed", "Only PUT method is allowed")
	}

	// Extract meeting ID from URL path
	meetingID := r.URL.Path[len("/api/meetings/"):]
	if meetingID == "" {
		return errors.NewValidationError("Meeting ID is required", "")
	}

	// Check if meeting exists
	meeting, exists := h.meetings[meetingID]
	if !exists {
		return errors.NewNotFoundError("Meeting not found")
	}

	var req api.UpdateMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
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
				return errors.NewNotFoundError("Participant not found: " + participantID)
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
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// DeleteMeeting handles deleting an existing meeting
func (h *MeetingHandler) DeleteMeeting(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodDelete {
		return errors.NewValidationError("Method not allowed", "Only DELETE method is allowed")
	}

	// Extract meeting ID from URL path
	meetingID := r.URL.Path[len("/api/meetings/"):]
	if meetingID == "" {
		return errors.NewValidationError("Meeting ID is required", "")
	}

	// Check if meeting exists
	if _, exists := h.meetings[meetingID]; !exists {
		return errors.NewNotFoundError("Meeting not found")
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
	return nil
}

// UpdateAvailability handles updating a participant's availability
func (h *MeetingHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPut {
		return errors.NewValidationError("Method not allowed", "Only PUT method is allowed")
	}

	// Extract availability ID from URL path
	availabilityID := r.URL.Path[len("/api/availabilities/"):]
	if availabilityID == "" {
		return errors.NewValidationError("Availability ID is required", "")
	}

	// Check if availability exists
	availability, exists := h.availabilities[availabilityID]
	if !exists {
		return errors.NewNotFoundError("Availability not found")
	}

	var req api.UpdateAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
	}

	if len(req.AvailableSlots) == 0 {
		return errors.NewValidationError("At least one available time slot is required", "")
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
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// DeleteAvailability handles deleting a participant's availability
func (h *MeetingHandler) DeleteAvailability(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodDelete {
		return errors.NewValidationError("Method not allowed", "Only DELETE method is allowed")
	}

	// Extract availability ID from URL path
	availabilityID := r.URL.Path[len("/api/availabilities/"):]
	if availabilityID == "" {
		return errors.NewValidationError("Availability ID is required", "")
	}

	// Check if availability exists
	if _, exists := h.availabilities[availabilityID]; !exists {
		return errors.NewNotFoundError("Availability not found")
	}

	// Delete availability
	delete(h.availabilities, availabilityID)

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// GetAvailability handles getting a participant's availability
func (h *MeetingHandler) GetAvailability(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return errors.NewValidationError("Method not allowed", "Only GET method is allowed")
	}

	// Get query parameters
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		return errors.NewValidationError("User ID is required", "")
	}

	meetingID := r.URL.Query().Get("meetingId")
	if meetingID == "" {
		return errors.NewValidationError("Meeting ID is required", "")
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
		return errors.NewNotFoundError("Availability not found")
	}

	resp := api.GetAvailabilityResponse{
		Availability: foundAvailability,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}

// calculateRecommendations calculates recommended time slots based on participant availability
func (h *MeetingHandler) calculateRecommendations(meeting models.Meeting, availabilities []models.Availability) []models.RecommendedSlot {
	// Map to track the number of participants available for each proposed slot
	slotAvailability := make(map[string]int)
	slotMap := make(map[string]models.TimeSlot)
	unavailableParticipants := make(map[string][]models.User)

	// Initialize the maps with proposed slots
	for _, slot := range meeting.ProposedSlots {
		slotAvailability[slot.ID] = 0
		slotMap[slot.ID] = slot
		unavailableParticipants[slot.ID] = []models.User{}
	}

	// Track which participants are available for each slot
	participantAvailability := make(map[string]map[string]bool) // participantID -> slotID -> available
	allParticipants := append([]models.User{*meeting.Organizer}, meeting.Participants...)
	for _, participant := range allParticipants {
		participantAvailability[participant.ID] = make(map[string]bool)
		for _, slot := range meeting.ProposedSlots {
			participantAvailability[participant.ID][slot.ID] = false
		}
	}

	// Process each availability entry
	for _, availability := range availabilities {
		// Skip if not a participant or organizer
		isValid := false
		if availability.ParticipantID == meeting.OrganizerID {
			isValid = true
		} else {
			for _, participant := range meeting.Participants {
				if participant.ID == availability.ParticipantID {
					isValid = true
					break
				}
			}
		}
		if !isValid {
			continue
		}

		// For each available slot in the availability entry
		for _, availableSlot := range availability.AvailableSlots {
			// Match with proposed slots
			for _, proposedSlot := range meeting.ProposedSlots {
				if availableSlot.StartTime.Equal(proposedSlot.StartTime) &&
					availableSlot.EndTime.Equal(proposedSlot.EndTime) {
					slotAvailability[proposedSlot.ID]++
					participantAvailability[availability.ParticipantID][proposedSlot.ID] = true
				}
			}
		}
	}

	// Build unavailable participants list for each slot
	for _, participant := range allParticipants {
		for slotID := range slotMap {
			if !participantAvailability[participant.ID][slotID] {
				unavailableParticipants[slotID] = append(unavailableParticipants[slotID], participant)
			}
		}
	}

	// Convert to recommended slots
	recommendations := make([]models.RecommendedSlot, 0, len(meeting.ProposedSlots))
	totalParticipants := len(allParticipants)

	for slotID, count := range slotAvailability {
		slot := slotMap[slotID]
		recommendations = append(recommendations, models.RecommendedSlot{
			TimeSlot:                slot,
			AvailableCount:          count,
			TotalParticipants:       totalParticipants,
			UnavailableParticipants: unavailableParticipants[slotID],
		})
	}

	// Sort recommendations by available count in descending order
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].AvailableCount > recommendations[j].AvailableCount
	})

	return recommendations
}
