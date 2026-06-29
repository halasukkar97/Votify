package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMovieVoteHandler checks the lightweight root route.
// This is the part of main.go that can be unit tested without starting the server.
func TestMovieVoteHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	MovieVoteHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	if !strings.Contains(response.Body.String(), "Movie Vote API") {
		t.Fatalf("expected health response to mention Movie Vote API, got %q", response.Body.String())
	}
}
