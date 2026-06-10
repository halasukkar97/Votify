package api

import (
	"encoding/json"
	"movie-vote/database"
	"movie-vote/user"
	"net/http"
)

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Name string `json:"name"`
}

// CreateUserResponse is returned after a user is created.
type CreateUserResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateUserHandler handles POST /users.
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	createdUser := user.CreateNewUser(user.CreateUserInput{
		Name: req.Name,
	})

	err := SaveUser(createdUser)
	if err != nil {
		http.Error(w, "failed to save user", http.StatusInternalServerError)
		return
	}

	response := CreateUserResponse{
		ID:   createdUser.ID,
		Name: createdUser.Name,
	}

	w.WriteHeader(http.StatusCreated)
	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)

		return
	}
}

// UsersHandler routes requests to the correct user handler.
func UsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		CreateUserHandler(w, r)
		return
	}

	if r.Method == http.MethodGet {
		ListUsersHandler(w, r)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// return all users as a json file 
func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := GetAllUsers()

	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(users)
}

func GetAllUsers() ([]user.User, error) {

	rows, err := database.DB.Query(
		"SELECT id, name FROM users",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []user.User

	for rows.Next() {
		var currentUser user.User

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
