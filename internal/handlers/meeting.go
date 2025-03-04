package handlers

import (
	"encoding/json"
	"net/http"

	"meetsync/internal/api"
	"meetsync/internal/interfaces"
	"meetsync/internal/services"
	"meetsync/pkg/errors"
	"meetsync/pkg/logs"
)

// MeetingHandler handles meeting-related requests
type MeetingHandler struct {
	service interfaces.MeetingService
}

// NewMeetingHandler creates a new MeetingHandler
func NewMeetingHandler(userHandler *UserHandler) *MeetingHandler {
	return &MeetingHandler{
		service: services.NewMeetingService(userHandler.service),
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

	// Create meeting using service
	createdMeeting, err := h.service.CreateMeeting(
		req.Title,
		req.OrganizerID,
		req.EstimatedDuration,
		req.ProposedSlots,
		req.ParticipantIDs,
	)
	if err != nil {
		return err
	}

	logs.Info("Created meeting: %s by organizer %s", createdMeeting.ID, createdMeeting.Organizer.Name)

	// Return response
	resp := api.CreateMeetingResponse{
		Meeting: createdMeeting,
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

	// Add availability using service
	_, err := h.service.AddAvailability(req.UserID, req.MeetingID, req.AvailableSlots)
	if err != nil {
		return err
	}

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

	// Get recommendations using service
	recommendations, err := h.service.GetRecommendations(meetingID)
	if err != nil {
		return err
	}

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

	var req api.UpdateMeetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
	}

	// Update meeting using service
	updatedMeeting, err := h.service.UpdateMeeting(
		meetingID,
		req.Title,
		req.EstimatedDuration,
		req.ProposedSlots,
		req.ParticipantIDs,
	)
	if err != nil {
		return err
	}

	resp := api.UpdateMeetingResponse{
		Meeting: updatedMeeting,
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

	// Delete meeting using service
	err := h.service.DeleteMeeting(meetingID)
	if err != nil {
		return err
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

	var req api.UpdateAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.NewValidationError("Invalid request body", err.Error())
	}

	// Update availability using service
	updatedAvailability, err := h.service.UpdateAvailability(availabilityID, req.AvailableSlots)
	if err != nil {
		return err
	}

	resp := api.UpdateAvailabilityResponse{
		Availability: updatedAvailability,
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

	// Delete availability using service
	err := h.service.DeleteAvailability(availabilityID)
	if err != nil {
		return err
	}

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

	// Get availability using service
	availability, err := h.service.GetAvailability(userID, meetingID)
	if err != nil {
		return err
	}

	resp := api.GetAvailabilityResponse{
		Availability: availability,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return errors.NewInternalError("Failed to encode response", err)
	}
	return nil
}
