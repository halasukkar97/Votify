package domain

import (
	"github.com/google/uuid"
)

// Vote represents one user's selected options for one poll.
type Vote struct {
	ID        string   `json:"id"`
	PollID    string   `json:"pollId"`
	UserID    string   `json:"userId"`
	OptionIDs []string `json:"optionIds"`
	MovieIDs  []string `json:"movieIds,omitempty"`
}

// CreateVoteInput contains the data needed to create a vote.
type CreateVoteInput struct {
	PollID    string   `json:"pollId"`
	UserID    string   `json:"userId"`
	OptionIDs []string `json:"optionIds"`
	MovieIDs  []string `json:"movieIds,omitempty"`
}

// CreateNewVote creates a Vote with a new unique ID.
func CreateNewVote(input CreateVoteInput) Vote {
	return Vote{
		ID:        uuid.New().String(),
		PollID:    input.PollID,
		UserID:    input.UserID,
		OptionIDs: normalizeOptionIDs(input.OptionIDs, input.MovieIDs),
		MovieIDs:  normalizeOptionIDs(input.OptionIDs, input.MovieIDs),
	}
}

func normalizeOptionIDs(optionIDs []string, movieIDs []string) []string {
	if len(optionIDs) > 0 {
		return optionIDs
	}

	return movieIDs
}
