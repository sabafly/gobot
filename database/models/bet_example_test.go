package models_test

import (
	"fmt"

	"github.com/disgoorg/snowflake/v2"
	"github.com/google/uuid"
	"github.com/sabafly/gobot/database/models"
)

// Example demonstrates how to use the IsOwner method to check authorization
// before allowing sensitive operations like deciding bet results.
func ExampleBetHost_IsOwner() {
	// Create a bet session with an organizer
	organizerID := snowflake.ID(123456789)
	betSession := &models.BetHost{
		ID:      uuid.New(),
		OwnerID: organizerID,
		Title:   "Who will win the race?",
		Mode:    string(models.BetVoteTypeRace),
		Status:  string(models.BetStatusEntry),
	}

	// Check if a user is authorized to perform operations
	regularUserID := snowflake.ID(987654321)

	// Only the organizer can decide results
	if betSession.IsOwner(organizerID) {
		fmt.Println("Organizer can decide results: true")
	}

	if !betSession.IsOwner(regularUserID) {
		fmt.Println("Regular user can decide results: false")
	}

	// Output:
	// Organizer can decide results: true
	// Regular user can decide results: false
}

// Example of checking authorization for closing a bet
func ExampleBetHost_IsOwner_closeBet() {
	organizerID := snowflake.ID(111111)
	betSession := &models.BetHost{
		ID:      uuid.New(),
		OwnerID: organizerID,
		Title:   "Predict the winner",
		Mode:    string(models.BetVoteTypeGuess),
		Status:  string(models.BetStatusVoting),
	}

	userID := snowflake.ID(222222)

	// Check authorization before allowing close operation
	if betSession.IsOwner(userID) {
		fmt.Println("User can close the bet")
	} else {
		fmt.Println("User is not authorized to close the bet")
	}

	// Output:
	// User is not authorized to close the bet
}
