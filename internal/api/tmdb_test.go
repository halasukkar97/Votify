package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// roundTripFunc lets a test replace http.DefaultTransport with a small function.
// That means SearchMovies can call http.Get without reaching the real TMDB service.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// withMockTMDB replaces outgoing HTTP calls for the duration of one test.
func withMockTMDB(t *testing.T, fn func(*http.Request) string) {
	t.Helper()

	oldTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := fn(r)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	t.Cleanup(func() {
		http.DefaultTransport = oldTransport
	})
}

func TestSearchMoviesEscapesQueryAndAddsPosterURL(t *testing.T) {
	t.Setenv("TMDB_API_KEY", "test-key")

	withMockTMDB(t, func(r *http.Request) string {
		if r.URL.Query().Get("query") != "star wars & dune" {
			t.Fatalf("expected decoded query to match original search text, got %q", r.URL.Query().Get("query"))
		}

		if strings.Contains(r.URL.RawQuery, "star wars") {
			t.Fatalf("expected raw query to be URL escaped, got %q", r.URL.RawQuery)
		}

		if r.URL.Query().Get("api_key") != "test-key" {
			t.Fatalf("expected api key from environment, got %q", r.URL.Query().Get("api_key"))
		}

		return `{"page":1,"results":[{"id":11,"title":"Dune","release_date":"2021-10-22","overview":"Desert politics","poster_path":"/poster.jpg"}]}`
	})

	movies, err := SearchMovies("star wars & dune")
	if err != nil {
		t.Fatalf("expected SearchMovies to succeed, got %v", err)
	}

	if len(movies) != 1 {
		t.Fatalf("expected one TMDB result, got %d", len(movies))
	}

	if movies[0].PosterURL != "https://image.tmdb.org/t/p/w500/poster.jpg" {
		t.Fatalf("expected poster URL to be expanded, got %q", movies[0].PosterURL)
	}
}

func TestSearchMoviesHandlerRequiresQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/movies/search", nil)
	response := httptest.NewRecorder()

	SearchMoviesHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected missing query to return 400, got %d", response.Code)
	}
}

func TestSearchMoviesHandlerReturnsTMDBResults(t *testing.T) {
	os.Setenv("TMDB_API_KEY", "test-key")
	t.Cleanup(func() {
		os.Unsetenv("TMDB_API_KEY")
	})

	withMockTMDB(t, func(r *http.Request) string {
		return `{"page":1,"results":[{"id":22,"title":"Arrival","release_date":"2016-11-11","overview":"First contact","poster_path":"/arrival.jpg"}]}`
	})

	request := httptest.NewRequest(http.MethodGet, "/movies/search?q=arrival", nil)
	response := httptest.NewRecorder()

	SearchMoviesHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	var movies []ExternalMovie
	if err := json.NewDecoder(response.Body).Decode(&movies); err != nil {
		t.Fatalf("failed to decode handler response: %v", err)
	}

	if len(movies) != 1 || movies[0].Title != "Arrival" {
		t.Fatalf("expected Arrival result, got %+v", movies)
	}
}
