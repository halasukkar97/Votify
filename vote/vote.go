package vote

import (
	"github.com/google/uuid"
)

// Vote represents one user's selected movies for one poll.
type Vote struct {
	ID       string
	PollID   string
	UserID   string
	MovieIDs []string
}

// CreateVoteInput contains the data needed to create a vote.
type CreateVoteInput struct {
	PollID   string
	UserID   string
	MovieIDs []string
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
