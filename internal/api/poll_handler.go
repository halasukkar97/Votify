package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// CreatePollRequest is the JSON body clients send when they create a poll.
type CreatePollRequest struct {
	Name              string    `json:"name"`
	MaxVotesPerPerson int       `json:"maxVotesPerPerson"`
	Deadline          time.Time `json:"deadline"`
	PollType          string    `json:"pollType"`
}

// CreatePollResponse is the JSON response sent back after a poll is created.
type CreatePollResponse struct {
	ID                string    `json:"id"`
	PollCode          string    `json:"pollCode"`
	Name              string    `json:"name"`
	MaxVotesPerPerson int       `json:"maxVotesPerPerson"`
	IsClosed          bool      `json:"isClosed"`
	IsVotingActive    bool      `json:"isVotingActive"`
	Deadline          time.Time `json:"deadline"`
	PollType          string    `json:"pollType"`
}

// CreatePollHandler handles POST /polls.
// It reads JSON from the request, creates a poll model, stores it, and returns it.
func (server *Server) CreatePollHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePollRequest

	// Decode turns the incoming JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	createdPoll, err := server.Service.CreatePoll(req.Name, req.MaxVotesPerPerson, req.Deadline, req.PollType)
	if err != nil {
		writeServiceError(w, err, "failed to save poll")
		return
	}

	response := CreatePollResponse{
		ID:                createdPoll.ID,
		PollCode:          createdPoll.PollCode,
		Name:              createdPoll.Name,
		MaxVotesPerPerson: createdPoll.MaxVotesPerPerson,
		IsClosed:          createdPoll.IsClosed,
		IsVotingActive:    createdPoll.IsVotingActive,
		Deadline:          createdPoll.Deadline,
		PollType:          createdPoll.PollType,
	}

	writeJSON(w, http.StatusCreated, response)
}

// PollsHandler routes /polls requests by HTTP method.
func (server *Server) PollsHandler(w http.ResponseWriter, r *http.Request) {
	// POST /polls creates a new poll.
	if r.Method == http.MethodPost {
		server.CreatePollHandler(w, r)
		return
	}

	// GET /polls lists the polls stored in PostgreSQL.
	if r.Method == http.MethodGet {
		server.ListPollsHandler(w, r)
		return
	}

	// Any other method, such as PUT or DELETE, is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ResultsHandler handles GET /results?pollCode=...
// It finds the requested poll and returns vote totals keyed by option ID.
func (server *Server) ResultsHandler(w http.ResponseWriter, r *http.Request) {
	// Query parameters come from the URL after the question mark.
	pollCode := r.URL.Query().Get("pollCode")
	pollID := r.URL.Query().Get("pollId")

	results, err := server.Service.GetResults(pollCode, pollID)
	if err != nil {
		writeServiceError(w, err, "poll not found")
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// ListPollsHandler handles GET /polls.
func (server *Server) ListPollsHandler(w http.ResponseWriter, r *http.Request) {
	polls, err := server.Service.ListPolls()
	if err != nil {
		writeServiceError(w, err, "failed to load polls")
		return
	}

	writeJSON(w, http.StatusOK, polls)
}

// PollByIDHandler handles GET /polls/{id}.
// It extracts the ID from the URL path, loads that poll, and returns it as JSON.
func (server *Server) PollByIDHandler(w http.ResponseWriter, r *http.Request) {
	pollIdentifier := strings.TrimPrefix(r.URL.Path, "/polls/")

	if strings.HasSuffix(pollIdentifier, "/activate-voting") {
		server.ActivateVotingHandler(w, r, strings.TrimSuffix(pollIdentifier, "/activate-voting"))
		return
	}

	foundPoll, err := server.Service.GetPoll(pollIdentifier)
	if err != nil {
		writeServiceError(w, err, "poll not found")
		return
	}

	writeJSON(w, http.StatusOK, foundPoll)
}

// ActivateVotingHandler handles PATCH /polls/{pollCode}/activate-voting.
func (server *Server) ActivateVotingHandler(w http.ResponseWriter, r *http.Request, pollCode string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	updatedPoll, err := server.Service.ActivateVoting(pollCode)
	if err != nil {
		writeServiceError(w, err, "failed to activate voting")
		return
	}

	writeJSON(w, http.StatusOK, updatedPoll)
}

func CreatePollHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().CreatePollHandler(w, r)
}

func PollsHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().PollsHandler(w, r)
}

func ResultsHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().ResultsHandler(w, r)
}

func ListPollsHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().ListPollsHandler(w, r)
}

func PollByIDHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().PollByIDHandler(w, r)
}

func ActivateVotingHandler(w http.ResponseWriter, r *http.Request, pollCode string) {
	defaultServer().ActivateVotingHandler(w, r, pollCode)
}
