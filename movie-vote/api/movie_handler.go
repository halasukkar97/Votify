package api

import (
	"encoding/json"
	"movie-vote/database"
	"movie-vote/movie"
	"net/http"
)

// CreateMovieRequest is the JSON body clients send when they add a movie to a poll.
type CreateMovieRequest struct {
	Title       string `json:"title"`
	PollID      string `json:"pollId"`
	ReleaseYear int    `json:"releaseYear"`
	Description string `json:"description"`
}

// CreateMovieResponse is the JSON response sent back after a movie is created.
type CreateMovieResponse struct {
	ID          string `json:"id"`
	PollID      string `json:"pollId"`
	Title       string `json:"title"`
	ReleaseYear int    `json:"releaseYear"`
	Description string `json:"description"`
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
	createdMovie := movie.CreateNewMovie(movie.CreateMovieInput{
		Title:       req.Title,
		PollID:      req.PollID,
		ReleaseYear: req.ReleaseYear,
		Description: req.Description,
	})

	// A movie must point to an existing poll, so check that before saving.
	pollExists, pollError := PollExists(req.PollID)
	if pollError != nil {
		http.Error(w, "failed to check poll", http.StatusInternalServerError)
		return
	}

	if !pollExists {
		http.Error(w, "poll not found", http.StatusNotFound)
		return
	}

	// Save the movie in PostgreSQL.
	err := SaveMovie(createdMovie)
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
	movies, err := GetAllMovies()

	if err != nil {
		http.Error(w, "failed to load movies", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(movies)
}

// GetAllMovies reads every movie row from PostgreSQL and converts each row into a movie.Movie.
func GetAllMovies() ([]movie.Movie, error) {
	// Query returns rows, which must be scanned one at a time.
	rows, err := database.DB.Query(
		"SELECT id, poll_id, title, release_year, description FROM movies",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var movies []movie.Movie

	// rows.Next moves through the result set one database row at a time.
	for rows.Next() {
		var currentMovie movie.Movie

		// Scan copies the current row's columns into the movie struct fields.
		err := rows.Scan(
			&currentMovie.ID,
			&currentMovie.PollID,
			&currentMovie.Title,
			&currentMovie.ReleaseYear,
			&currentMovie.Description,
		)

		if err != nil {
			return nil, err
		}

		movies = append(movies, currentMovie)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

// GetMoviesByPollID reads only the movies that belong to one poll.
// Poll listing uses this to include each poll's movie options in the response.
func GetMoviesByPollID(pollID string) ([]movie.Movie, error) {
	// The WHERE clause filters the movies table down to the requested poll ID.
	rows, err := database.DB.Query(
		"SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id = $1",
		pollID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var movies []movie.Movie

	// Build one movie struct for each returned database row.
	for rows.Next() {
		var currentMovie movie.Movie

		err := rows.Scan(
			&currentMovie.ID,
			&currentMovie.PollID,
			&currentMovie.Title,
			&currentMovie.ReleaseYear,
			&currentMovie.Description,
		)

		if err != nil {
			return nil, err
		}

		movies = append(movies, currentMovie)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}
