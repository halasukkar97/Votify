package api

import (
	"encoding/json"
	"net/http"
	"votify/internal/database"
	"votify/internal/domain"
)

// createVoteRequest is the JSON body clients send when they vote in a poll.
type createVoteRequest struct {
	MovieIDs []string `json:"movieIds"`
	PollID   string   `json:"pollId"`
	PollCode string   `json:"pollCode"`
	UserID   string   `json:"userId"`
}

// createVoteResponse is the JSON response sent back after a vote is accepted.
type createVoteResponse struct {
	ID       string   `json:"id"`
	PollID   string   `json:"pollId"`
	PollCode string   `json:"pollCode"`
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

	// A vote can only be submitted to an existing poll.
	var foundPoll *domain.Poll
	var found bool

	if req.PollCode != "" {
		foundPoll, found = database.FindPollByCode(req.PollCode)
	}
	if !found && req.PollID != "" {
		foundPoll, found = database.FindPollByID(req.PollID)
	}

	if !found {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	// Build the vote model from the request data.
	// PollID stays internal, even when the client joins with a public pollCode.
	createdVote := domain.CreateNewVote(domain.CreateVoteInput{
		PollID:   foundPoll.ID,
		UserID:   req.UserID,
		MovieIDs: req.MovieIDs,
	})

	// SubmitVote checks the poll rules before adding the vote to the poll.
	submitErr := foundPoll.SubmitVote(createdVote)
	if submitErr != nil {
		http.Error(w, submitErr.Error(), http.StatusBadRequest)
		return
	}

	// SaveVote writes the vote and its selected movie IDs to PostgreSQL.
	saveErr := database.SaveVote(createdVote)
	if saveErr != nil {
		http.Error(w, "failed to save vote", http.StatusInternalServerError)
		return
	}
	response := createVoteResponse{
		ID:       createdVote.ID,
		PollID:   createdVote.PollID,
		PollCode: foundPoll.PollCode,
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
