package domain

import (
	"github.com/google/uuid"
)

// Movie represents one movie option that users can vote for in a poll.
type Movie struct {
	ID          string `json:"id"`
	PollID      string `json:"pollId"`
	Title       string `json:"title"`
	ReleaseYear int    `json:"releaseYear"`
	Description string `json:"description"`
	PosterURL   string `json:"posterUrl"`
}

// CreateMovieInput contains the data needed to create a movie.
type CreateMovieInput struct {
	Title       string `json:"title"`
	PollID      string `json:"pollId"`
	ReleaseYear int    `json:"releaseYear"`
	Description string `json:"description"`
	PosterURL   string `json:"posterUrl"`
}

// CreateNewMovie creates a Movie with a new unique ID.
func CreateNewMovie(input CreateMovieInput) Movie {
	return Movie{
		ID:          uuid.New().String(),
		PollID:      input.PollID,
		Title:       input.Title,
		ReleaseYear: input.ReleaseYear,
		Description: input.Description,
		PosterURL:   input.PosterURL,
	}
}
