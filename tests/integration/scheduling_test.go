package integration

import (
	"fmt"
	"testing"
	"time"

	"meetsync/internal/models"
)

func TestCompleteSchedulingFlow(t *testing.T) {
	ts := NewTestSetup(t)
	defer ts.Cleanup() // Ensure cleanup runs after test

	// Create organizer and participants
	organizer, err := ts.CreateUser("Meeting Organizer", "organizer")
	if err != nil {
		t.Fatalf("Failed to create organizer: %v", err)
	}

	participants := make([]*TestUser, 0, 4)
	for i := 1; i <= 4; i++ {
		user, err := ts.CreateUser(
			fmt.Sprintf("Participant %d", i),
			fmt.Sprintf("participant%d", i),
		)
		if err != nil {
			t.Fatalf("Failed to create participant %d: %v", i, err)
		}
		participants = append(participants, user)
	}

	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	// Create meeting with multiple time slots across different days
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	dayAfter := now.Add(48 * time.Hour)

	proposedSlots := []models.TimeSlot{
		{
			StartTime: tomorrow.Add(9 * time.Hour),  // 9 AM tomorrow
			EndTime:   tomorrow.Add(10 * time.Hour), // 10 AM tomorrow
		},
		{
			StartTime: tomorrow.Add(14 * time.Hour), // 2 PM tomorrow
			EndTime:   tomorrow.Add(15 * time.Hour), // 3 PM tomorrow
		},
		{
			StartTime: dayAfter.Add(11 * time.Hour), // 11 AM day after
			EndTime:   dayAfter.Add(12 * time.Hour), // 12 PM day after
		},
	}

	// Create meeting
	meetingID, err := ts.CreateMeeting(
		"Project Kickoff",
		organizer.ID,
		participantIDs,
		proposedSlots,
	)
	if err != nil {
		t.Fatalf("Failed to create meeting: %v", err)
	}

	// Simulate real-world scenario where participants submit availability at different times

	// Step 1: First two participants submit their availability immediately
	// Participant 1 available for all slots
	if err := ts.AddAvailability(participants[0].ID, meetingID, proposedSlots); err != nil {
		t.Fatalf("Failed to add availability for participant 1: %v", err)
	}

	// Participant 2 available for first and last slots
	availableSlots := []models.TimeSlot{proposedSlots[0], proposedSlots[2]}
	if err := ts.AddAvailability(participants[1].ID, meetingID, availableSlots); err != nil {
		t.Fatalf("Failed to add availability for participant 2: %v", err)
	}

	// Check initial recommendations with partial responses
	initialRecs, err := ts.GetRecommendations(meetingID)
	if err != nil {
		t.Fatalf("Failed to get initial recommendations: %v", err)
	}

	if len(initialRecs.RecommendedSlots) == 0 {
		t.Fatal("Expected initial recommendations with partial responses, got none")
	}

	// Step 2: Organizer submits availability
	if err := ts.AddAvailability(organizer.ID, meetingID, proposedSlots); err != nil {
		t.Fatalf("Failed to add organizer availability: %v", err)
	}

	// Step 3: Remaining participants submit their availability
	// Participant 3 only available for morning slots
	morningSlots := []models.TimeSlot{proposedSlots[0], proposedSlots[2]}
	if err := ts.AddAvailability(participants[2].ID, meetingID, morningSlots); err != nil {
		t.Fatalf("Failed to add availability for participant 3: %v", err)
	}

	// Participant 4 only available for afternoon slot
	afternoonSlot := []models.TimeSlot{proposedSlots[1]}
	if err := ts.AddAvailability(participants[3].ID, meetingID, afternoonSlot); err != nil {
		t.Fatalf("Failed to add availability for participant 4: %v", err)
	}

	// Get final recommendations
	finalRecs, err := ts.GetRecommendations(meetingID)
	if err != nil {
		t.Fatalf("Failed to get final recommendations: %v", err)
	}

	// Verify final recommendations
	if len(finalRecs.RecommendedSlots) == 0 {
		t.Fatal("Expected final recommendations, got none")
	}

	// First slot should be the most popular (3 participants + organizer)
	firstSlot := finalRecs.RecommendedSlots[0]
	if firstSlot.AvailableCount != 4 {
		t.Errorf("Expected 4 available participants for best slot, got %d", firstSlot.AvailableCount)
	}

	// Verify slots are ordered by availability (lowest to highest)
	for i := 1; i < len(finalRecs.RecommendedSlots); i++ {
		if finalRecs.RecommendedSlots[i-1].AvailableCount > finalRecs.RecommendedSlots[i].AvailableCount {
			t.Errorf("Slots not properly ordered by availability count: slot[%d]=%d > slot[%d]=%d",
				i-1, finalRecs.RecommendedSlots[i-1].AvailableCount,
				i, finalRecs.RecommendedSlots[i].AvailableCount)
		}
	}
}

func TestConflictingMeetingsScenario(t *testing.T) {
	ts := NewTestSetup(t)
	defer ts.Cleanup() // Ensure cleanup runs after test

	// Create organizer and participants
	organizer, err := ts.CreateUser("Busy Organizer", "busy.organizer")
	if err != nil {
		t.Fatalf("Failed to create organizer: %v", err)
	}

	participants := make([]*TestUser, 0, 3)
	for i := 1; i <= 3; i++ {
		user, err := ts.CreateUser(
			fmt.Sprintf("Busy Participant %d", i),
			fmt.Sprintf("busy.participant%d", i),
		)
		if err != nil {
			t.Fatalf("Failed to create participant %d: %v", i, err)
		}
		participants = append(participants, user)
	}

	participantIDs := make([]string, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.ID
	}

	// Create overlapping time slots for two meetings
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	meeting1Slots := []models.TimeSlot{
		{
			StartTime: tomorrow.Add(9 * time.Hour),  // 9 AM
			EndTime:   tomorrow.Add(10 * time.Hour), // 10 AM
		},
		{
			StartTime: tomorrow.Add(10 * time.Hour), // 10 AM
			EndTime:   tomorrow.Add(11 * time.Hour), // 11 AM
		},
	}

	meeting2Slots := []models.TimeSlot{
		{
			StartTime: tomorrow.Add(9*time.Hour + 30*time.Minute),  // 9:30 AM
			EndTime:   tomorrow.Add(10*time.Hour + 30*time.Minute), // 10:30 AM
		},
		{
			StartTime: tomorrow.Add(14 * time.Hour), // 2 PM
			EndTime:   tomorrow.Add(15 * time.Hour), // 3 PM
		},
	}

	// Create first meeting
	meeting1ID, err := ts.CreateMeeting(
		"Morning Meeting",
		organizer.ID,
		participantIDs,
		meeting1Slots,
	)
	if err != nil {
		t.Fatalf("Failed to create first meeting: %v", err)
	}

	// Create second meeting
	meeting2ID, err := ts.CreateMeeting(
		"Overlapping Meeting",
		organizer.ID,
		participantIDs,
		meeting2Slots,
	)
	if err != nil {
		t.Fatalf("Failed to create second meeting: %v", err)
	}

	// Add availability for first meeting
	for _, user := range append([]*TestUser{organizer}, participants...) {
		if err := ts.AddAvailability(user.ID, meeting1ID, meeting1Slots); err != nil {
			t.Fatalf("Failed to add availability for user %s in meeting 1: %v", user.Name, err)
		}
	}

	// Add availability for second meeting
	for _, user := range append([]*TestUser{organizer}, participants...) {
		if err := ts.AddAvailability(user.ID, meeting2ID, meeting2Slots); err != nil {
			t.Fatalf("Failed to add availability for user %s in meeting 2: %v", user.Name, err)
		}
	}

	// Get recommendations for both meetings
	meeting1Recs, err := ts.GetRecommendations(meeting1ID)
	if err != nil {
		t.Fatalf("Failed to get recommendations for meeting 1: %v", err)
	}

	meeting2Recs, err := ts.GetRecommendations(meeting2ID)
	if err != nil {
		t.Fatalf("Failed to get recommendations for meeting 2: %v", err)
	}

	// Verify that recommendations are returned
	if len(meeting1Recs.RecommendedSlots) == 0 || len(meeting2Recs.RecommendedSlots) == 0 {
		t.Fatal("Expected recommendations for both meetings")
	}

	// For meeting 1, both slots should be equally available since all participants said they're available
	expectedCount := len(participants) + 1 // +1 for organizer
	if meeting1Recs.RecommendedSlots[0].AvailableCount < expectedCount-1 {
		t.Errorf("Expected at least %d participants to be available for meeting 1, got %d available",
			expectedCount-1, meeting1Recs.RecommendedSlots[0].AvailableCount)
	}

	// For meeting 2, both slots should be equally available since all participants said they're available
	if meeting2Recs.RecommendedSlots[0].AvailableCount < expectedCount-1 {
		t.Errorf("Expected at least %d participants to be available for meeting 2, got %d available",
			expectedCount-1, meeting2Recs.RecommendedSlots[0].AvailableCount)
	}

	// Verify that at least one non-overlapping slot is recommended in the top slots
	hasNonOverlapping1 := false
	for _, rec := range meeting1Recs.RecommendedSlots {
		if rec.TimeSlot.StartTime.Equal(meeting1Slots[1].StartTime) {
			hasNonOverlapping1 = true
			break
		}
	}
	if !hasNonOverlapping1 {
		t.Error("Expected non-overlapping slot (10-11 AM) to be recommended for meeting 1")
	}

	hasNonOverlapping2 := false
	for _, rec := range meeting2Recs.RecommendedSlots {
		if rec.TimeSlot.StartTime.Equal(meeting2Slots[1].StartTime) {
			hasNonOverlapping2 = true
			break
		}
	}
	if !hasNonOverlapping2 {
		t.Error("Expected non-overlapping slot (2-3 PM) to be recommended for meeting 2")
	}
}
