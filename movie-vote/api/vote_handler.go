package api

import (
	"encoding/json"
	"movie-vote/database"
	"movie-vote/vote"
	"net/http"
)

// createVoteRequest is the JSON body clients send when they vote in a poll.
type createVoteRequest struct {
	MovieIDs []string `json:"movieIds"`
	PollID   string   `json:"pollId"`
	UserID   string   `json:"userId"`
}

// createVoteResponse is the JSON response sent back after a vote is accepted.
type createVoteResponse struct {
	ID       string   `json:"id"`
	PollID   string   `json:"pollId"`
	UserID   string   `json:"userId"`
	MovieIDs []string `json:"movieIds"`
}

// CreateVoteHandler handles POST /votes.
// It creates a vote, asks the poll to validate it, stores it, and returns it.
func CreateVoteHandler(w http.ResponseWriter, r *http.Request) {
	var req createVoteRequest

	// Decode the JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	// Build the vote model from the request data.
	createdVote := vote.CreateNewVote(vote.CreateVoteInput{
		PollID:   req.PollID,
		UserID:   req.UserID,
		MovieIDs: req.MovieIDs,
	})

	// A vote can only be submitted to an existing poll.
	foundPoll, found := FindPollByID(req.PollID)

	if !found {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	// SubmitVote checks the poll rules before adding the vote to the poll.
	submitErr := foundPoll.SubmitVote(createdVote)
	if submitErr != nil {
		http.Error(w, submitErr.Error(), http.StatusBadRequest)
		return
	}

	// SaveVote writes the vote and its selected movie IDs to PostgreSQL.
	saveErr := SaveVote(createdVote)
	if saveErr != nil {
		http.Error(w, "failed to save vote", http.StatusInternalServerError)
		return
	}
	response := createVoteResponse{
		ID:       createdVote.ID,
		PollID:   createdVote.PollID,
		UserID:   createdVote.UserID,
		MovieIDs: createdVote.MovieIDs,
	}

	w.WriteHeader(http.StatusCreated)
	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)

		return
	}
}

// GetVotesByPollID reads all votes submitted for one poll.
// It also loads each vote's selected movie IDs from the vote_movies table.
func GetVotesByPollID(pollID string) ([]vote.Vote, error) {
	// First load the vote rows for this poll.
	rows, err := database.DB.Query(
		"SELECT id, poll_id, user_id FROM votes WHERE poll_id = $1",
		pollID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var votes []vote.Vote

	// Build one vote struct for each returned database row.
	for rows.Next() {
		var currentVote vote.Vote

		err := rows.Scan(
			&currentVote.ID,
			&currentVote.PollID,
			&currentVote.UserID,
		)

		if err != nil {
			return nil, err
		}

		// The selected movies live in the vote_movies join table.
		movieIDs, err := GetMovieIDsByVoteID(currentVote.ID)
		if err != nil {
			return nil, err
		}

		currentVote.MovieIDs = movieIDs
		votes = append(votes, currentVote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return votes, nil
}
