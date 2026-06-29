package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"votify/database"
	"votify/user"
)

// CreateUserRequest is the JSON body clients send when they create a user.
type CreateUserRequest struct {
	Name string `json:"name"`
}

// CreateUserResponse is the JSON response sent back after a user is created.
type CreateUserResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpdateUserRequest is the JSON body clients send when they rename a user.
type UpdateUserRequest struct {
	Name string `json:"name"`
}

// CreateUserHandler handles POST /users.
// It reads the new user's name, creates a user model, saves it in PostgreSQL,
// and returns the created user data.
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	// Decode the incoming JSON request body into req.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	// The user package creates the ID and fills the user struct.
	createdUser := user.CreateNewUser(user.CreateUserInput{
		Name: req.Name,
	})

	// SaveUser writes the user to PostgreSQL, so this can fail if the DB is down.
	err := SaveUser(createdUser)
	if err != nil {
		http.Error(w, "failed to save user", http.StatusInternalServerError)
		return
	}

	// Build a response struct instead of exposing internal storage details.
	response := CreateUserResponse{
		ID:   createdUser.ID,
		Name: createdUser.Name,
	}

	// Send 201 Created and then write the response as JSON.
	w.WriteHeader(http.StatusCreated)
	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)

		return
	}
}

// UsersHandler routes requests to the correct user handler.
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	// POST /users creates a new user.
	if r.Method == http.MethodPost {
		CreateUserHandler(w, r)
		return
	}

	// GET /users lists users from PostgreSQL.
	if r.Method == http.MethodGet {
		ListUsersHandler(w, r)
		return
	}

	// Any other method is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// UserByIDHandler routes requests that target one existing user.
func UserByIDHandler(w http.ResponseWriter, r *http.Request) {
	// PATCH /users/{id} updates the saved display name without changing identity.
	if r.Method == http.MethodPatch {
		UpdateUserHandler(w, r)
		return
	}

	// Any other method is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// UpdateUserHandler handles PATCH /users/{id}.
// It lets the frontend edit a display name while keeping the same user ID for voting.
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/users/")
	if userID == "" {
		http.Error(w, "missing user id", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest

	// Decode the incoming JSON request body into req.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	updatedUser, err := UpdateUserName(userID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedUser)
}

// ListUsersHandler handles GET /users.
// It loads all users from PostgreSQL and returns them as JSON.
func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := GetAllUsers()

	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}

// GetAllUsers reads every user row from PostgreSQL and converts each row into a user.User.
func GetAllUsers() ([]user.User, error) {

	rows, err := database.DB.Query(
		"SELECT id, name FROM users",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	users := make([]user.User, 0)

	// rows.Next moves through the result set one database row at a time.
	for rows.Next() {
		var currentUser user.User

		// Scan copies the current row's columns into Go variables.
		err := rows.Scan(
			&currentUser.ID,
			&currentUser.Name,
		)

		if err != nil {
			return nil, err
		}

		users = append(users, currentUser)
	}

	return users, nil
}
