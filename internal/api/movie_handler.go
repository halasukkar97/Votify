package api

import (
	"encoding/json"
	"net/http"
	"votify/internal/database"
	"votify/internal/domain"
)

// CreateMovieRequest is the JSON body clients send when they add a movie to a poll.
type CreateMovieRequest struct {
	Title       string `json:"title"`
	PollID      string `json:"pollId"`
	ReleaseYear int    `json:"releaseYear"`
	Description string `json:"description"`
	PosterURL   string `json:"posterUrl"`
}

// CreateMovieResponse is the JSON response sent back after a movie is created.
type CreateMovieResponse struct {
	ID          string `json:"id"`
	PollID      string `json:"pollId"`
	Title       string `json:"title"`
	ReleaseYear int    `json:"releaseYear"`
	Description string `json:"description"`
	PosterURL   string `json:"posterUrl"`
}

// CreateMovieHandler handles POST /movies.
// It creates a movie model, confirms the poll exists, saves the movie in PostgreSQL,
// and returns the saved data.
func CreateMovieHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateMovieRequest

	// Decode the JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	// Build the movie model before saving it.
	createdMovie := domain.CreateNewMovie(domain.CreateMovieInput{
		Title:       req.Title,
		PollID:      req.PollID,
		ReleaseYear: req.ReleaseYear,
		Description: req.Description,
		PosterURL:   req.PosterURL,
	})

	// A movie must point to an existing poll, so check that before saving.
	foundPoll, pollFound, pollError := database.FindPollByIDWithError(req.PollID)
	if pollError != nil {
		http.Error(w, "failed to check poll", http.StatusInternalServerError)
		return
	}

	if !pollFound {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	// Once voting starts, the setup phase is locked and movies cannot change.
	if foundPoll.IsVotingActive {
		http.Error(w, "voting has already started, movies can no longer be added", http.StatusBadRequest)
		return
	}

	if foundPoll.IsClosed || foundPoll.IsExpired() {
		http.Error(w, "poll is closed or expired, movies can no longer be added", http.StatusBadRequest)
		return
	}

	// Save the movie in PostgreSQL.
	err := database.SaveMovie(createdMovie)
	if err != nil {
		http.Error(w, "failed to save movie", http.StatusInternalServerError)
		return
	}

	// Return the created movie data to the client.
	response := CreateMovieResponse{
		ID:          createdMovie.ID,
		PollID:      createdMovie.PollID,
		Title:       createdMovie.Title,
		ReleaseYear: createdMovie.ReleaseYear,
		Description: createdMovie.Description,
		PosterURL:   createdMovie.PosterURL,
	}

	// Send 201 Created before writing the JSON body.
	w.WriteHeader(http.StatusCreated)

	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)

		return
	}
}

// MoviesHandler routes /movies requests by HTTP method.
func MoviesHandler(w http.ResponseWriter, r *http.Request) {
	// POST /movies creates a new movie.
	if r.Method == http.MethodPost {
		CreateMovieHandler(w, r)
		return
	}

	// GET /movies lists every movie from PostgreSQL.
	if r.Method == http.MethodGet {
		ListMoviesHandler(w, r)
		return
	}

	// Any other method is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ListMoviesHandler handles GET /movies.
func ListMoviesHandler(w http.ResponseWriter, r *http.Request) {
	// Load all stored movies before encoding them as JSON.
	movies, err := database.GetAllMovies()

	if err != nil {
		http.Error(w, "failed to load movies", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(movies)
}
