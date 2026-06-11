package movie

import (
	"github.com/google/uuid"
)

// Movie represents one movie option that users can vote for in a poll.
type Movie struct {
	ID          string
	PollID      string
	Title       string
	ReleaseYear int
	Description string
}

// CreateMovieInput contains the data needed to create a movie.
type CreateMovieInput struct {
	Title       string
	PollID      string
	ReleaseYear int
	Description string
}

// CreateNewMovie creates a Movie with a new unique ID.
func CreateNewMovie(input CreateMovieInput) Movie {
	return Movie{
		ID:          uuid.New().String(),
		PollID:      input.PollID,
		Title:       input.Title,
		ReleaseYear: input.ReleaseYear,
		Description: input.Description,
	}
}
