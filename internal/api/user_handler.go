package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
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
func (server *Server) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	// Decode the incoming JSON request body into req.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	createdUser, err := server.Service.CreateUser(req.Name)
	if err != nil {
		writeServiceError(w, err, "failed to save user")
		return
	}

	response := CreateUserResponse{ID: createdUser.ID, Name: createdUser.Name}
	writeJSON(w, http.StatusCreated, response)
}

// UsersHandler routes requests to the correct user handler.
func (server *Server) UsersHandler(w http.ResponseWriter, r *http.Request) {
	// POST /users creates a new user.
	if r.Method == http.MethodPost {
		server.CreateUserHandler(w, r)
		return
	}

	// GET /users lists users from PostgreSQL.
	if r.Method == http.MethodGet {
		server.ListUsersHandler(w, r)
		return
	}

	// Any other method is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// UserByIDHandler routes requests that target one existing user.
func (server *Server) UserByIDHandler(w http.ResponseWriter, r *http.Request) {
	// PATCH /users/{id} updates the saved display name without changing identity.
	if r.Method == http.MethodPatch {
		server.UpdateUserHandler(w, r)
		return
	}

	// Any other method is not supported for this route.
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// UpdateUserHandler handles PATCH /users/{id}.
// It lets the frontend edit a display name while keeping the same user ID for voting.
func (server *Server) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
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

	updatedUser, err := server.Service.UpdateUserName(userID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		writeServiceError(w, err, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, updatedUser)
}

// ListUsersHandler handles GET /users.
// It loads all users from PostgreSQL and returns them as JSON.
func (server *Server) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := server.Service.ListUsers()
	if err != nil {
		writeServiceError(w, err, "failed to load users")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().CreateUserHandler(w, r)
}

func UsersHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().UsersHandler(w, r)
}

func UserByIDHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().UserByIDHandler(w, r)
}

func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().UpdateUserHandler(w, r)
}

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().ListUsersHandler(w, r)
}
