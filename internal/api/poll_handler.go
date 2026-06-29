package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"
	"votify/internal/database"
	"votify/internal/domain"
)

// CreatePollRequest is the JSON body clients send when they create a poll.
type CreatePollRequest struct {
	Name              string    `json:"name"`
	MaxVotesPerPerson int       `json:"maxVotesPerPerson"`
	Deadline          time.Time `json:"deadline"`
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
}

// CreatePollHandler handles POST /polls.
// It reads JSON from the request, creates a poll model, stores it, and returns it.
func CreatePollHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePollRequest

	// Decode turns the incoming JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	pollCode, err := GenerateUniquePollCode()
	if err != nil {
		http.Error(w, "failed to generate poll code", http.StatusInternalServerError)
		return
	}

	// The poll package owns the rules for building a new poll.
	createdPoll := domain.CreateNewPoll(domain.CreatePollInput{
		PollCode:          pollCode,
		Name:              req.Name,
		MaxVotesPerPerson: req.MaxVotesPerPerson,
		Deadline:          req.Deadline,
	})

	// Save the poll in PostgreSQL so later requests can list or find it.
	err = database.SavePoll(createdPoll)
	if err != nil {
		http.Error(w, "failed to save poll", http.StatusInternalServerError)
		return
	}

	// Only expose the fields the API should return to the client.
	response := CreatePollResponse{
		ID:                createdPoll.ID,
		PollCode:          createdPoll.PollCode,
		Name:              createdPoll.Name,
		MaxVotesPerPerson: createdPoll.MaxVotesPerPerson,
		IsClosed:          createdPoll.IsClosed,
		IsVotingActive:    createdPoll.IsVotingActive,
		Deadline:          createdPoll.Deadline,
	}

	// StatusCreated means the request succeeded and created a new resource.
	w.WriteHeader(http.StatusCreated)

	// Encode writes the Go response struct back to the client as JSON.
	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)

		return
	}
}

// PollsHandler routes /polls requests by HTTP method.
func PollsHandler(w http.ResponseWriter, r *http.Request) {
	// POST /polls creates a new poll.
	if r.Method == http.MethodPost {
		CreatePollHandler(w, r)
		return
	}

	// GET /polls lists the polls stored in PostgreSQL.
	if r.Method == http.MethodGet {
		ListPollsHandler(w, r)
		return
	}

	// Any other method, such as PUT or DELETE, is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ResultsHandler handles GET /results?pollCode=...
// It finds the requested poll and returns vote totals keyed by movie ID.
func ResultsHandler(w http.ResponseWriter, r *http.Request) {
	// Query parameters come from the URL after the question mark.
	pollCode := r.URL.Query().Get("pollCode")
	pollID := r.URL.Query().Get("pollId")

	var foundPoll *domain.Poll
	var found bool

	if pollCode != "" {
		foundPoll, found = database.FindPollByCode(pollCode)
	}
	if !found && pollID != "" {
		foundPoll, found = database.FindPollByID(pollID)
	}

	if !found {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	results := foundPoll.GetResults()

	err := json.NewEncoder(w).Encode(results)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ListPollsHandler handles GET /polls.
func ListPollsHandler(w http.ResponseWriter, r *http.Request) {
	// Load all stored polls before encoding them as JSON.
	polls, err := database.GetAllPolls()

	if err != nil {
		log.Printf("failed to load polls: %v", err)
		http.Error(w, "failed to load polls", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(polls)
}

// PollByIDHandler handles GET /polls/{id}.
// It extracts the ID from the URL path, loads that poll, and returns it as JSON.
func PollByIDHandler(w http.ResponseWriter, r *http.Request) {
	pollIdentifier := strings.TrimPrefix(r.URL.Path, "/polls/")

	if strings.HasSuffix(pollIdentifier, "/activate-voting") {
		ActivateVotingHandler(w, r, strings.TrimSuffix(pollIdentifier, "/activate-voting"))
		return
	}

	foundPoll, found, codeErr := database.FindPollByCodeWithError(pollIdentifier)
	if !found {
		var idErr error
		foundPoll, found, idErr = database.FindPollByIDWithError(pollIdentifier)
		if !found {
			log.Printf("poll not found for identifier %q: pollCodeErr=%v pollIDErr=%v", pollIdentifier, codeErr, idErr)
		}
	}

	if !found {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	// Encode writes the full poll, including loaded movies and votes, as JSON.
	err := json.NewEncoder(w).Encode(foundPoll)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ActivateVotingHandler handles PATCH /polls/{pollCode}/activate-voting.
func ActivateVotingHandler(w http.ResponseWriter, r *http.Request, pollCode string) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	foundPoll, found := database.FindPollByCode(pollCode)
	if !found {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	if foundPoll.IsVotingActive {
		http.Error(w, "voting is already active", http.StatusConflict)
		return
	}

	// Activating voting locks the setup phase so no more movies can be added.
	err := database.ActivateVoting(pollCode)
	if err != nil {
		http.Error(w, "failed to activate voting", http.StatusInternalServerError)
		return
	}

	updatedPoll, found := database.FindPollByCode(pollCode)
	if !found {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	err = json.NewEncoder(w).Encode(updatedPoll)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func GenerateUniquePollCode() (string, error) {
	for {
		number, err := rand.Int(rand.Reader, big.NewInt(100000000))
		if err != nil {
			return "", err
		}

		code := fmt.Sprintf("%08d", number.Int64())

		exists, err := database.PollCodeExists(code)
		if err != nil {
			return "", err
		}

		if !exists {
			return code, nil
		}
	}
}
