package api

import (
	"encoding/json"
	"net/http"
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
func (server *Server) CreateMovieHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateMovieRequest

	// Decode the JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	createdMovie, err := server.Service.CreateMovie(domain.CreateMovieInput{
		Title:       req.Title,
		PollID:      req.PollID,
		ReleaseYear: req.ReleaseYear,
		Description: req.Description,
		PosterURL:   req.PosterURL,
	})
	if err != nil {
		writeServiceError(w, err, "failed to save movie")
		return
	}

	response := CreateMovieResponse{
		ID:          createdMovie.ID,
		PollID:      createdMovie.PollID,
		Title:       createdMovie.Title,
		ReleaseYear: createdMovie.ReleaseYear,
		Description: createdMovie.Description,
		PosterURL:   createdMovie.PosterURL,
	}

	writeJSON(w, http.StatusCreated, response)
}

// MoviesHandler routes /movies requests by HTTP method.
func (server *Server) MoviesHandler(w http.ResponseWriter, r *http.Request) {
	// POST /movies creates a new movie.
	if r.Method == http.MethodPost {
		server.CreateMovieHandler(w, r)
		return
	}

	// GET /movies lists every movie from PostgreSQL.
	if r.Method == http.MethodGet {
		server.ListMoviesHandler(w, r)
		return
	}

	// Any other method is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ListMoviesHandler handles GET /movies.
func (server *Server) ListMoviesHandler(w http.ResponseWriter, r *http.Request) {
	movies, err := server.Service.ListMovies()
	if err != nil {
		writeServiceError(w, err, "failed to load movies")
		return
	}

	writeJSON(w, http.StatusOK, movies)
}

func CreateMovieHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().CreateMovieHandler(w, r)
}

func MoviesHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().MoviesHandler(w, r)
}

func ListMoviesHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().ListMoviesHandler(w, r)
}
