package api

import (
	"encoding/json"
	"net/http"
	"votify/internal/domain"
)

// CreateOptionRequest is the JSON body clients send when they add an option to a poll.
type CreateOptionRequest struct {
	Title       string         `json:"title"`
	PollID      string         `json:"pollId"`
	Description string         `json:"description"`
	ImageURL    string         `json:"imageUrl"`
	PosterURL   string         `json:"posterUrl"`
	ReleaseYear int            `json:"releaseYear"`
	Metadata    map[string]any `json:"metadata"`
}

// CreateOptionResponse is the JSON response sent back after an option is created.
type CreateOptionResponse struct {
	ID          string         `json:"id"`
	PollID      string         `json:"pollId"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	ImageURL    string         `json:"imageUrl"`
	PosterURL   string         `json:"posterUrl,omitempty"`
	ReleaseYear int            `json:"releaseYear,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Compatibility aliases keep older movie clients working while options become the main model.
type CreateMovieRequest = CreateOptionRequest
type CreateMovieResponse = CreateOptionResponse

// CreateOptionHandler handles POST /options.
func (server *Server) CreateOptionHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateOptionRequest

	// Decode the JSON request body into a Go struct.
	decodeErr := json.NewDecoder(r.Body).Decode(&req)
	if decodeErr != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	createdOption, err := server.Service.CreateOption(domain.CreateOptionInput{
		Title:       req.Title,
		PollID:      req.PollID,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		PosterURL:   req.PosterURL,
		ReleaseYear: req.ReleaseYear,
		Metadata:    req.Metadata,
	})
	if err != nil {
		writeServiceError(w, err, "failed to save option")
		return
	}

	writeJSON(w, http.StatusCreated, optionResponse(createdOption))
}

// OptionsHandler routes /options requests by HTTP method.
func (server *Server) OptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		server.CreateOptionHandler(w, r)
		return
	}

	if r.Method == http.MethodGet {
		server.ListOptionsHandler(w, r)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// ListOptionsHandler handles GET /options.
func (server *Server) ListOptionsHandler(w http.ResponseWriter, r *http.Request) {
	options, err := server.Service.ListOptions()
	if err != nil {
		writeServiceError(w, err, "failed to load options")
		return
	}

	writeJSON(w, http.StatusOK, options)
}

func optionResponse(option domain.Option) CreateOptionResponse {
	return CreateOptionResponse{
		ID:          option.ID,
		PollID:      option.PollID,
		Title:       option.Title,
		Description: option.Description,
		ImageURL:    option.ImageURL,
		PosterURL:   option.PosterURL,
		ReleaseYear: option.ReleaseYear,
		Metadata:    option.Metadata,
	}
}

func (server *Server) CreateMovieHandler(w http.ResponseWriter, r *http.Request) {
	server.CreateOptionHandler(w, r)
}

func (server *Server) MoviesHandler(w http.ResponseWriter, r *http.Request) {
	server.OptionsHandler(w, r)
}

func (server *Server) ListMoviesHandler(w http.ResponseWriter, r *http.Request) {
	server.ListOptionsHandler(w, r)
}

func CreateOptionHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().CreateOptionHandler(w, r)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().OptionsHandler(w, r)
}

func ListOptionsHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().ListOptionsHandler(w, r)
}

func CreateMovieHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().CreateMovieHandler(w, r)
}

func MoviesHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().MoviesHandler(w, r)
}

func ListMoviesHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().ListMoviesHandler(w, r)
}
