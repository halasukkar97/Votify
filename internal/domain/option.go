package domain

import "github.com/google/uuid"

// Option represents one generic choice that users can vote for in a poll.
// It can be a movie, book, restaurant, activity, destination, or any custom item.
type Option struct {
	ID          string         `json:"id"`
	PollID      string         `json:"pollId"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	ImageURL    string         `json:"imageUrl"`
	PosterURL   string         `json:"posterUrl,omitempty"`
	ReleaseYear int            `json:"releaseYear,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CreateOptionInput contains the data needed to create a generic poll option.
type CreateOptionInput struct {
	Title       string         `json:"title"`
	PollID      string         `json:"pollId"`
	Description string         `json:"description"`
	ImageURL    string         `json:"imageUrl"`
	PosterURL   string         `json:"posterUrl"`
	ReleaseYear int            `json:"releaseYear,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CreateNewOption creates an Option with a new unique ID.
func CreateNewOption(input CreateOptionInput) Option {
	imageURL := input.ImageURL
	if imageURL == "" {
		imageURL = input.PosterURL
	}

	return Option{
		ID:          uuid.New().String(),
		PollID:      input.PollID,
		Title:       input.Title,
		Description: input.Description,
		ImageURL:    imageURL,
		PosterURL:   imageURL,
		ReleaseYear: input.ReleaseYear,
		Metadata:    input.Metadata,
	}
}

// Movie remains as a compatibility alias for older code and existing JSON clients.
type Movie = Option

// CreateMovieInput remains as a compatibility alias while the app moves to options.
type CreateMovieInput = CreateOptionInput

// CreateNewMovie keeps old callers working while creating a generic option.
func CreateNewMovie(input CreateMovieInput) Movie {
	return CreateNewOption(input)
}
