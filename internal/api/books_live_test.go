package api

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"votify/internal/config"
)

// This opt-in check uses the real Google Books API, never fixtures.
func TestBooksLive(t *testing.T) {
	if os.Getenv("VOTIFY_LIVE_BOOKS_TEST") != "1" {
		t.Skip("set VOTIFY_LIVE_BOOKS_TEST=1 to call Google Books")
	}
	t.Chdir("../..")
	cfg := config.Load()
	t.Logf("Google Books key loaded: %t", cfg.GoogleBooksKey != "")
	server := NewServer(nil, "", cfg.GoogleBooksKey, "")
	for _, query := range []string{"mon", "Atomic Habits", "Harry Potter", "James Clear", "9780735211292"} {
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
			if len(books) == 0 {
				t.Fatal("no actual book results")
			}
			for _, book := range books {
				if book.Provider != "google-books" || book.ExternalID == "" || book.Title == "" {
					t.Fatalf("invalid result: %+v", book)
				}
			}
			t.Logf("%d real results; first title=%q", len(books), books[0].Title)
		})
	}
}
