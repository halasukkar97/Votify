package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"votify/internal/config"
	"votify/internal/database"
	"votify/internal/repository"
	"votify/internal/service"
)

// Server holds the dependencies used by HTTP handlers.
// Passing this into routes keeps handlers testable and avoids hidden package globals.
type Server struct {
	Service    *service.Service
	TMDBAPIKey string
}

// NewServer builds the HTTP adapter for the application service layer.
func NewServer(appService *service.Service, tmdbAPIKey string) *Server {
	return &Server{Service: appService, TMDBAPIKey: tmdbAPIKey}
}

func defaultServer() *Server {
	cfg := config.Load()
	store := repository.NewStore(database.DB)
	return NewServer(service.New(store), cfg.TMDBAPIKey)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(value)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	if err == nil {
		return
	}

	status := http.StatusInternalServerError
	message := fallback
	if message == "" {
		message = err.Error()
	}

	switch {
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
		message = "not found"
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, service.ErrInvalidInput):
		status = http.StatusBadRequest
		message = err.Error()
	}

	http.Error(w, message, status)
}
