package api

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// ExternalMovie is the movie shape returned by the TMDB search endpoint.
// It is separate from domain.Movie because TMDB uses its own IDs and field names.
type ExternalMovie struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
	Overview    string `json:"overview"`
	PosterPath  string `json:"poster_path"`
	PosterURL   string `json:"poster_url"`
}

// SearchResponse matches the top-level JSON object TMDB returns for a movie search.
type SearchResponse struct {
	Page    int             `json:"page"`
	Results []ExternalMovie `json:"results"`
}

// SearchMoviesHandler handles GET /movies/search?q=...
// It reads the search text from the query string and returns matching TMDB movies.
func (server *Server) SearchMoviesHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if query == "" {
		http.Error(w, "missing search query", http.StatusBadRequest)
		return
	}

	movies, err := SearchMovies(query, server.TMDBAPIKey)
	if err != nil {
		http.Error(w, "failed to search movies", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(movies)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SearchMovies calls TMDB's movie search API and converts the response into Go structs.
func SearchMovies(query string, apiKey string) ([]ExternalMovie, error) {
	// QueryEscape makes the search text safe to place inside a URL.
	// For example, "star wars" becomes "star+wars".
	escapedQuery := url.QueryEscape(query)

	// TMDB_API_KEY comes from the .env file loaded in main.
	url := "https://api.themoviedb.org/3/search/movie?query=" +
		escapedQuery + "&api_key=" + apiKey

	// Send a GET request to the external API.
	// Think: Go becomes a client, just like Postman.
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	// Close the connection when we're done reading.
	// Same idea as rows.Close() with PostgreSQL.
	defer response.Body.Close()

	// Create a struct variable that will hold
	// the decoded JSON response.
	var searchResponse SearchResponse

	// Convert JSON from the API into Go structs.
	err = json.NewDecoder(response.Body).Decode(&searchResponse)
	if err != nil {
		return nil, err
	}

	// TMDB gives a relative poster path, so add the image host to make it usable.
	for i := range searchResponse.Results {
		searchResponse.Results[i].PosterURL =
			"https://image.tmdb.org/t/p/w500" +
				searchResponse.Results[i].PosterPath
	}
	// Return only the movie results.
	// The caller doesn't care about page numbers.
	return searchResponse.Results, nil
}

func SearchMoviesHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().SearchMoviesHandler(w, r)
}
