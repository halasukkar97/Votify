package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"votify/database"
	"votify/poll"
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
	Name              string    `json:"name"`
	MaxVotesPerPerson int       `json:"maxVotesPerPerson"`
	IsClosed          bool      `json:"isClosed"`
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

	// The poll package owns the rules for building a new poll.
	createdPoll := poll.CreateNewPoll(poll.CreatePollInput{
		Name:              req.Name,
		MaxVotesPerPerson: req.MaxVotesPerPerson,
		Deadline:          req.Deadline,
	})

	// Save the poll in PostgreSQL so later requests can list or find it.
	err := SavePoll(createdPoll)
	if err != nil {
		http.Error(w, "failed to save poll", http.StatusInternalServerError)
		return
	}

	// Only expose the fields the API should return to the client.
	response := CreatePollResponse{
		ID:                createdPoll.ID,
		Name:              createdPoll.Name,
		MaxVotesPerPerson: createdPoll.MaxVotesPerPerson,
		IsClosed:          createdPoll.IsClosed,
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

// ResultsHandler handles GET /results?pollId=...
// It finds the requested poll and returns vote totals keyed by movie ID.
func ResultsHandler(w http.ResponseWriter, r *http.Request) {
	// Query parameters come from the URL after the question mark.
	pollID := r.URL.Query().Get("pollId")

	foundPoll, found := FindPollByID(pollID)

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
	polls, err := GetAllPolls()

	if err != nil {
		http.Error(w, "failed to load polls", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(polls)
}

// GetAllPolls reads every poll row from PostgreSQL and converts each row into a poll.Poll.
// It also loads each poll's movies and votes so clients can see the full poll state.
func GetAllPolls() ([]poll.Poll, error) {
	// Query returns rows, which must be scanned one at a time.
	rows, err := database.DB.Query(
		"SELECT id, name, is_closed, max_votes_per_person, deadline FROM polls",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var polls []poll.Poll

	// rows.Next moves through the result set one database row at a time.
	for rows.Next() {
		var currentPoll poll.Poll

		// Scan copies the current row's columns into the poll struct fields.
		err := rows.Scan(
			&currentPoll.ID,
			&currentPoll.Name,
			&currentPoll.IsClosed,
			&currentPoll.MaxVotesPerPerson,
			&currentPoll.Deadline,
		)

		if err != nil {
			return nil, err
		}

		// Load the movies connected to this poll before adding it to the response list.
		movies, err := GetMoviesByPollID(currentPoll.ID)
		if err != nil {
			return nil, err
		}

		currentPoll.Movies = movies

		// Load the votes connected to this poll, including the selected movie IDs.
		votes, err := GetVotesByPollID(currentPoll.ID)
		if err != nil {
			return nil, err
		}

		currentPoll.Votes = votes

		polls = append(polls, currentPoll)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return polls, nil
}

// PollByIDHandler handles GET /polls/{id}.
// It extracts the ID from the URL path, loads that poll, and returns it as JSON.
func PollByIDHandler(w http.ResponseWriter, r *http.Request) {
	// TrimPrefix removes "/polls/" so the remaining path is the poll ID.
	pollID := strings.TrimPrefix(r.URL.Path, "/polls/")

	foundPoll, found := FindPollByID(pollID)
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
