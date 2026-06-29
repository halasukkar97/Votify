package domain

import "github.com/google/uuid"

// User represents a person who can vote in movie polls.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateUserInput contains the data needed to create a user.
type CreateUserInput struct {
	Name string `json:"name"`
}

// CreateNewUser creates a User with a new unique ID.
func CreateNewUser(input CreateUserInput) User {
	return User{
		ID:   uuid.New().String(),
		Name: input.Name,
	}
}
