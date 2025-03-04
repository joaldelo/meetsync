package services

import (
	"sort"
	"time"

	"meetsync/internal/interfaces"
	"meetsync/internal/models"
	"meetsync/internal/repositories"
	"meetsync/pkg/errors"

	"github.com/google/uuid"
)

// MeetingServiceImpl implements the MeetingService interface
type MeetingServiceImpl struct {
	repository  repositories.MeetingRepository
	userService interfaces.UserService
}

var _ interfaces.MeetingService = (*MeetingServiceImpl)(nil) // Verify MeetingServiceImpl implements MeetingService interface

// NewMeetingService creates a new MeetingService
func NewMeetingService(userService interfaces.UserService) interfaces.MeetingService {
	return &MeetingServiceImpl{
		repository:  repositories.NewInMemoryMeetingRepository(),
		userService: userService,
	}
}

// CreateMeeting creates a new meeting
func (s *MeetingServiceImpl) CreateMeeting(title string, organizerID string, estimatedDuration int, proposedSlots []models.TimeSlot, participantIDs []string) (models.Meeting, error) {
	// Validate input
	if title == "" {
		return models.Meeting{}, errors.NewValidationError("Title is required", "")
	}
	if estimatedDuration <= 0 {
		return models.Meeting{}, errors.NewValidationError("Estimated duration must be positive", "")
	}
	if len(proposedSlots) == 0 {
		return models.Meeting{}, errors.NewValidationError("At least one proposed time slot is required", "")
	}

	// Validate organizer exists
	organizer, err := s.userService.GetUserByID(organizerID)
	if err != nil {
		return models.Meeting{}, errors.NewNotFoundError("Organizer not found")
	}

	// Validate participants exist
	var participants []models.User
	for _, participantID := range participantIDs {
		participant, err := s.userService.GetUserByID(participantID)
		if err != nil {
			return models.Meeting{}, errors.NewNotFoundError("Participant not found: " + participantID)
		}
		participants = append(participants, participant)
	}

	// Create meeting
	meeting := models.Meeting{
		Title:             title,
		OrganizerID:       organizerID,
		Organizer:         &organizer,
		EstimatedDuration: estimatedDuration,
		ProposedSlots:     proposedSlots,
		Participants:      participants,
	}

	return s.repository.CreateMeeting(meeting)
}

// GetRecommendations gets meeting time recommendations based on participant availability
func (s *MeetingServiceImpl) GetRecommendations(meetingID string) ([]models.RecommendedSlot, error) {
	// Get meeting
	meeting, err := s.repository.GetMeetingByID(meetingID)
	if err != nil {
		return nil, err
	}

	// Get availabilities
	availabilities, err := s.repository.GetMeetingAvailabilities(meetingID)
	if err != nil {
		return nil, err
	}

	return s.calculateRecommendations(meeting, availabilities), nil
}

// UpdateMeeting updates an existing meeting
func (s *MeetingServiceImpl) UpdateMeeting(meetingID string, title string, estimatedDuration int, proposedSlots []models.TimeSlot, participantIDs []string) (models.Meeting, error) {
	// Get existing meeting
	meeting, err := s.repository.GetMeetingByID(meetingID)
	if err != nil {
		return models.Meeting{}, err
	}

	// Update fields if provided
	if title != "" {
		meeting.Title = title
	}
	if estimatedDuration > 0 {
		meeting.EstimatedDuration = estimatedDuration
	}
	if len(proposedSlots) > 0 {
		// Assign IDs to new time slots
		for i := range proposedSlots {
			if proposedSlots[i].ID == "" {
				proposedSlots[i].ID = uuid.New().String()
			}
		}
		meeting.ProposedSlots = proposedSlots
	}
	if len(participantIDs) > 0 {
		// Validate and update participants
		var participants []models.User
		for _, participantID := range participantIDs {
			participant, err := s.userService.GetUserByID(participantID)
			if err != nil {
				return models.Meeting{}, errors.NewNotFoundError("Participant not found: " + participantID)
			}
			participants = append(participants, participant)
		}
		meeting.Participants = participants
	}

	return s.repository.UpdateMeeting(meeting)
}

// DeleteMeeting deletes a meeting
func (s *MeetingServiceImpl) DeleteMeeting(meetingID string) error {
	return s.repository.DeleteMeeting(meetingID)
}

// AddAvailability adds a participant's availability for a meeting
func (s *MeetingServiceImpl) AddAvailability(userID string, meetingID string, availableSlots []models.TimeSlot) (models.Availability, error) {
	// Validate user exists
	user, err := s.userService.GetUserByID(userID)
	if err != nil {
		return models.Availability{}, err
	}

	// Validate meeting exists
	meeting, err := s.repository.GetMeetingByID(meetingID)
	if err != nil {
		return models.Availability{}, err
	}

	// Match available slots with proposed slots
	var matchedSlots []models.TimeSlot
	for _, availableSlot := range availableSlots {
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
			return models.Availability{}, errors.NewValidationError("Available slot does not match any proposed slot", "")
		}
	}

	// Create availability
	availability := models.Availability{
		ParticipantID:  userID,
		Participant:    &user,
		MeetingID:      meetingID,
		AvailableSlots: matchedSlots,
	}

	return s.repository.CreateAvailability(availability)
}

// UpdateAvailability updates a participant's availability
func (s *MeetingServiceImpl) UpdateAvailability(availabilityID string, availableSlots []models.TimeSlot) (models.Availability, error) {
	if len(availableSlots) == 0 {
		return models.Availability{}, errors.NewValidationError("At least one available time slot is required", "")
	}

	// Find the availability
	availabilities := s.repository.GetAllAvailabilities()
	var availability models.Availability
	var found bool
	for _, a := range availabilities {
		if a.ID == availabilityID {
			availability = a
			found = true
			break
		}
	}
	if !found {
		return models.Availability{}, errors.NewNotFoundError("Availability not found")
	}

	// Update availability
	availability.AvailableSlots = availableSlots
	availability.UpdatedAt = time.Now()

	return s.repository.UpdateAvailability(availability)
}

// DeleteAvailability deletes a participant's availability
func (s *MeetingServiceImpl) DeleteAvailability(availabilityID string) error {
	return s.repository.DeleteAvailability(availabilityID)
}

// GetAvailability gets a participant's availability for a meeting
func (s *MeetingServiceImpl) GetAvailability(userID string, meetingID string) (models.Availability, error) {
	return s.repository.GetAvailability(userID, meetingID)
}

// calculateRecommendations calculates recommended time slots based on participant availability
func (s *MeetingServiceImpl) calculateRecommendations(meeting models.Meeting, availabilities []models.Availability) []models.RecommendedSlot {
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
				if availableSlot.ID == proposedSlot.ID {
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
