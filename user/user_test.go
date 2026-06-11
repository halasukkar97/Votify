package user

import (
	"testing"

	"github.com/google/uuid"
)

// TestCreateNewUser proves that the constructor copies the name
// and generates a fresh UUID for the user ID.
func TestCreateNewUser(t *testing.T) {
	input := CreateUserInput{Name: "Hela"}

	u := CreateNewUser(input)

	// uuid.Parse fails if the ID is not a real UUID string.
	if _, err := uuid.Parse(u.ID); err != nil {
		t.Fatalf("expected valid UUID, got %q: %v", u.ID, err)
	}

	if u.Name != input.Name {
		t.Errorf("expected name %q, got %q", input.Name, u.Name)
	}
}
