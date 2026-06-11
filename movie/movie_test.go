package movie

import (
	"testing"

	"github.com/google/uuid"
)

// TestCreateNewMovie proves that the constructor copies all input fields
// and generates a fresh UUID for the new movie.
func TestCreateNewMovie(t *testing.T) {
	input := CreateMovieInput{
		Title:       "Interstellar",
		PollID:      "poll-123",
		ReleaseYear: 2014,
		Description: "Space exploration",
	}

	m := CreateNewMovie(input)

	// uuid.Parse fails if the ID is not a real UUID string.
	if _, err := uuid.Parse(m.ID); err != nil {
		t.Fatalf("expected valid UUID, got %q: %v", m.ID, err)
	}

	if m.Title != input.Title {
		t.Errorf("expected title %q, got %q", input.Title, m.Title)
	}

	if m.PollID != input.PollID {
		t.Errorf("expected poll ID %q, got %q", input.PollID, m.PollID)
	}

	if m.ReleaseYear != input.ReleaseYear {
		t.Errorf("expected release year %d, got %d", input.ReleaseYear, m.ReleaseYear)
	}

	if m.Description != input.Description {
		t.Errorf("expected description %q, got %q", input.Description, m.Description)
	}
}
