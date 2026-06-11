package user

import "github.com/google/uuid"

// User represents a person who can vote in movie polls.
type User struct {
	ID   string
	Name string
}

// CreateUserInput contains the data needed to create a user.
type CreateUserInput struct {
	Name string
}

// CreateNewUser creates a User with a new unique ID.
func CreateNewUser(input CreateUserInput) User {
	return User{
		ID:   uuid.New().String(),
		Name: input.Name,
	}
}
