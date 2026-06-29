package domain

import (
	"github.com/google/uuid"
)

// Vote represents one user's selected movies for one poll.
type Vote struct {
	ID       string   `json:"id"`
	PollID   string   `json:"pollId"`
	UserID   string   `json:"userId"`
	MovieIDs []string `json:"movieIds"`
}

// CreateVoteInput contains the data needed to create a vote.
type CreateVoteInput struct {
	PollID   string   `json:"pollId"`
	UserID   string   `json:"userId"`
	MovieIDs []string `json:"movieIds"`
}

// CreateNewVote creates a Vote with a new unique ID.
func CreateNewVote(input CreateVoteInput) Vote {
	return Vote{
		ID:       uuid.New().String(),
		PollID:   input.PollID,
		UserID:   input.UserID,
		MovieIDs: input.MovieIDs,
	}
}
