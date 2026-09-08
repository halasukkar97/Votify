package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
	"votify/internal/repository"
	"votify/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
)

// This opt-in check uses the real Open Library API, never fixtures.
func TestBooksLive(t *testing.T) {
	if os.Getenv("VOTIFY_LIVE_BOOKS_TEST") != "1" {
		t.Skip("set VOTIFY_LIVE_BOOKS_TEST=1 to call Open Library")
	}
	t.Chdir("../..")
	server := NewServer(nil, "", "")
	for _, query := range []string{"Atomic Habits", "Harry Potter", "James Clear", "9780735211292", "some-random-nonsense-book-xyz123456"} {
		t.Run(query, func(t *testing.T) {
			response := httptest.NewRecorder()
			server.SearchOptionsHandler(response, httptest.NewRequest("GET", "/options/search?type=book&q="+url.QueryEscape(query), nil))
			if response.Code != 200 {
				t.Fatalf("backend status=%d body=%s", response.Code, response.Body.String())
			}
			var books []ExternalOption
			if err := json.Unmarshal(response.Body.Bytes(), &books); err != nil {
				t.Fatal(err)
			}
			if query == "some-random-nonsense-book-xyz123456" {
				if len(books) != 0 {
					t.Fatal("expected no books")
				}
				return
			}
			if len(books) == 0 {
				t.Fatal("no actual book results")
			}
			for _, book := range books {
				if book.Provider != "open-library" || book.ExternalID == "" || book.Title == "" {
					t.Fatalf("invalid result: %+v", book)
				}
			}
			book := books[0]
			if len(book.Metadata["authors"].([]any)) == 0 || book.ReleaseYear <= 0 || !strings.HasPrefix(book.ImageURL, "https://covers.openlibrary.org/b/id/") || book.Metadata["isbn"] == nil || book.Metadata["goodreadsUrl"] == nil {
				t.Fatalf("expected real book metadata: %+v", book)
			}
			t.Logf("%d real results; first=%+v", len(books), book)
			if query == "Atomic Habits" {
				details := httptest.NewRecorder()
				server.BookDetailsHandler(details, httptest.NewRequest("GET", "/options/book-details?key="+url.QueryEscape(book.ExternalID), nil))
				if details.Code != 200 {
					t.Fatalf("details status=%d body=%s", details.Code, details.Body.String())
				}
				var detail map[string]string
				if err := json.Unmarshal(details.Body.Bytes(), &detail); err != nil {
					t.Fatal(err)
				}
				book.Overview = detail["description"]
				t.Logf("selected work description: %d characters", len(book.Overview))
				verifyRealBookAddToPoll(t, book)
				response, err := http.Get(book.ImageURL)
				if err != nil {
					t.Fatal(err)
				}
				response.Body.Close()
				if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Type"), "image/") {
					t.Fatalf("cover status=%d type=%s", response.StatusCode, response.Header.Get("Content-Type"))
				}
			}
		})
	}
}

// Use the real provider result with an isolated SQL test double, leaving user polls untouched.
func verifyRealBookAddToPoll(t *testing.T, book ExternalOption) {
	db, mock := newMockDatabase(t)
	server := NewServer(service.New(repository.NewStore(db)), "", "")
	expectPollLookupByIDWithVoting(mock, "poll-1", time.Now().Add(24*time.Hour), false)
	expectEmptyRelations(mock, "poll-1")
	metadata, err := json.Marshal(book.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("INSERT INTO options").WithArgs(sqlmock.AnyArg(), "poll-1", book.Title, book.Overview, book.ImageURL, book.ReleaseYear, book.Provider, book.ExternalID, string(metadata)).WillReturnResult(sqlmock.NewResult(1, 1))
	payload, err := json.Marshal(CreateOptionRequest{Title: book.Title, PollID: "poll-1", Description: book.Overview, ImageURL: book.ImageURL, ReleaseYear: book.ReleaseYear, Provider: book.Provider, ExternalID: book.ExternalID, Metadata: book.Metadata})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.OptionsHandler(response, httptest.NewRequest("POST", "/options", bytes.NewReader(payload)))
	if response.Code != 201 {
		t.Fatalf("add to poll status=%d body=%s", response.Code, response.Body.String())
	}
	var saved CreateOptionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Provider != book.Provider || saved.ExternalID != book.ExternalID || saved.ImageURL != book.ImageURL || saved.Description != book.Overview {
		t.Fatalf("saved book=%+v", saved)
	}
	requireExpectations(t, mock)
}
