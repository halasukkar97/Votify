package api

import (
	"encoding/json"
	"net/http"
)

// createVoteRequest is the JSON body clients send when they vote in a poll.
type createVoteRequest struct {
	OptionIDs []string `json:"optionIds"`
	MovieIDs  []string `json:"movieIds"`
	PollID    string   `json:"pollId"`
	PollCode  string   `json:"pollCode"`
	UserID    string   `json:"userId"`
}

// createVoteResponse is the JSON response sent back after a vote is accepted.
type createVoteResponse struct {
	ID        string   `json:"id"`
	PollID    string   `json:"pollId"`
	PollCode  string   `json:"pollCode"`
	UserID    string   `json:"userId"`
	OptionIDs []string `json:"optionIds"`
	MovieIDs  []string `json:"movieIds"`
}

// CreateVoteHandler handles POST /votes.
// It creates a vote, asks the poll to validate it, stores it, and returns it.
func (server *Server) CreateVoteHandler(w http.ResponseWriter, r *http.Request) {
	var req createVoteRequest

	// Decode the JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	optionIDs := req.OptionIDs
	if len(optionIDs) == 0 {
		optionIDs = req.MovieIDs
	}

	createdVote, pollCode, err := server.Service.SubmitVote(req.PollCode, req.PollID, req.UserID, optionIDs)
	if err != nil {
		writeServiceError(w, err, "failed to save vote")
		return
	}

	response := createVoteResponse{
		ID:        createdVote.ID,
		PollID:    createdVote.PollID,
		PollCode:  pollCode,
		UserID:    createdVote.UserID,
		OptionIDs: createdVote.OptionIDs,
		MovieIDs:  createdVote.OptionIDs,
	}

	writeJSON(w, http.StatusCreated, response)
}

func CreateVoteHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().CreateVoteHandler(w, r)
}
