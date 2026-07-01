package api

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// ExternalOption is the normalized shape returned by external search providers.
type ExternalOption struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
	Overview    string `json:"overview"`
	PosterPath  string `json:"poster_path"`
	PosterURL   string `json:"poster_url"`
}

// SearchProvider returns option suggestions for a poll type.
type SearchProvider interface {
	Search(query string) ([]ExternalOption, error)
}

type movieSearchProvider struct {
	apiKey string
}

func (provider movieSearchProvider) Search(query string) ([]ExternalOption, error) {
	return SearchMovies(query, provider.apiKey)
}

// SearchProviderForType returns the search strategy for a poll type.
func SearchProviderForType(pollType string, tmdbAPIKey string) SearchProvider {
	switch pollType {
	case "movie", "movies":
		return movieSearchProvider{apiKey: tmdbAPIKey}
	default:
		return nil
	}
}

// SearchResponse matches the top-level JSON object TMDB returns for a movie search.
type SearchResponse struct {
	Page    int              `json:"page"`
	Results []ExternalOption `json:"results"`
}

// SearchOptionsHandler handles GET /options/search?type=movie&q=...
// Providers make it easy to add new search sources later.
func (server *Server) SearchOptionsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	if query == "" {
		http.Error(w, "missing search query", http.StatusBadRequest)
		return
	}

	pollType := r.URL.Query().Get("type")
	if pollType == "" {
		pollType = "movie"
	}

	provider := SearchProviderForType(pollType, server.TMDBAPIKey)
	if provider == nil {
		writeJSON(w, http.StatusOK, []ExternalOption{})
		return
	}

	options, err := provider.Search(query)
	if err != nil {
		http.Error(w, "failed to search options", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(options)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SearchMovies calls TMDB's movie search API and converts the response into generic option suggestions.
func SearchMovies(query string, apiKey string) ([]ExternalOption, error) {
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

func (server *Server) SearchMoviesHandler(w http.ResponseWriter, r *http.Request) {
	server.SearchOptionsHandler(w, r)
}

func SearchOptionsHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().SearchOptionsHandler(w, r)
}

func SearchMoviesHandler(w http.ResponseWriter, r *http.Request) {
	defaultServer().SearchMoviesHandler(w, r)
}
