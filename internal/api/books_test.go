package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func mockOpenLibrary(t *testing.T, status int, body string) {
	t.Helper()
	previous := openLibraryClient
	openLibraryClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "openlibrary.org" || !strings.Contains(r.UserAgent(), "Votify") {
			t.Fatalf("unexpected request: %s, UA=%s", r.URL, r.UserAgent())
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	t.Cleanup(func() { openLibraryClient = previous })
}

func TestOpenLibraryOptionalFields(t *testing.T) {
	mockOpenLibrary(t, 200, `{"docs":[{"key":"/works/OL1W","title":"Minimal"}]}`)
	books, err := SearchBooks("Minimal & title")
	if err != nil || len(books) != 1 {
		t.Fatalf("books=%+v error=%v", books, err)
	}
	book := books[0]
	if book.ImageURL != "" || book.ReleaseYear != 0 || book.Overview != "" {
		t.Fatalf("unexpected optional fields: %+v", book)
	}
	if book.Metadata["authors"] == nil || book.Metadata["goodreadsUrl"] != "https://www.goodreads.com/search?q=Minimal" {
		t.Fatalf("metadata=%+v", book.Metadata)
	}
}

func TestOpenLibraryProviderFailureIsNotEmptySuccess(t *testing.T) {
	mockOpenLibrary(t, 429, `{"error":"rate limited"}`)
	_, err := SearchBooks("book")
	if err == nil || !strings.Contains(err.Error(), "status=429, body={\"error\":\"rate limited\"}") {
		t.Fatalf("error=%v", err)
	}
	response := httptest.NewRecorder()
	NewServer(nil, "", "").SearchOptionsHandler(response, httptest.NewRequest("GET", "/options/search?type=book&q=book", nil))
	if response.Code != 500 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestOpenLibraryMalformedResponse(t *testing.T) {
	mockOpenLibrary(t, 200, `{"docs":`)
	if _, err := SearchBooks("book"); err == nil {
		t.Fatal("expected parsing error")
	}
}

func TestBookDetailsDescriptionForms(t *testing.T) {
	for _, test := range []struct{ body, want string }{
		{`{"description":"plain"}`, "plain"},
		{`{"description":{"value":"object"}}`, "object"},
		{`{}`, ""}, {`{"description":null}`, ""},
	} {
		t.Run(test.body, func(t *testing.T) {
			mockOpenLibrary(t, 200, test.body)
			response := httptest.NewRecorder()
			NewServer(nil, "", "").BookDetailsHandler(response, httptest.NewRequest("GET", "/options/book-details?key="+url.QueryEscape("/works/OL1W"), nil))
			var result map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != 200 || result["description"] != test.want {
				t.Fatalf("status=%d body=%s error=%v", response.Code, response.Body.String(), err)
			}
		})
	}
}

func TestBookDetailsRejectsInvalidKey(t *testing.T) {
	for _, key := range []string{"https://example.com", "/works/../secret", "", "/books/OL1M"} {
		response := httptest.NewRecorder()
		NewServer(nil, "", "").BookDetailsHandler(response, httptest.NewRequest("GET", "/options/book-details?key="+url.QueryEscape(key), nil))
		if response.Code != 400 {
			t.Fatalf("key=%q status=%d", key, response.Code)
		}
	}
}

func TestBookISBNAndGoodreadsFallback(t *testing.T) {
	if got := preferredISBN([]string{"0735211299", "978-0735211292"}); got != "9780735211292" {
		t.Fatal(got)
	}
	if got := preferredISBN([]string{"junk", "012345678X"}); got != "012345678X" {
		t.Fatal(got)
	}
	if got := goodreadsSearchURL("", "Title & more", []string{"First Author", "Second Author"}); got != "https://www.goodreads.com/search?q=Title+%26+more+First+Author" {
		t.Fatal(got)
	}
}

func TestBookSearchUsesLimitedEncodedFields(t *testing.T) {
	previous := openLibraryClient
	openLibraryClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/search.json" || r.URL.Query().Get("q") != "title & author" || r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("fields") != "key,title,author_name,first_publish_year,cover_i,isbn,publisher" {
			t.Fatalf("unexpected URL: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"docs":[]}`))}, nil
	})}
	t.Cleanup(func() { openLibraryClient = previous })
	provider := SearchProviderForType("book", "unused-tmdb-key")
	books, err := provider.Search("title & author")
	if err != nil || books == nil || len(books) != 0 {
		t.Fatalf("books=%+v err=%v", books, err)
	}
}
