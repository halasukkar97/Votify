package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ExternalOption is the normalized shape returned by external search providers.
type ExternalOption struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	ReleaseDate string         `json:"release_date,omitempty"`
	Overview    string         `json:"overview,omitempty"`
	PosterPath  string         `json:"poster_path,omitempty"`
	PosterURL   string         `json:"poster_url,omitempty"`
	ImageURL    string         `json:"imageUrl,omitempty"`
	ReleaseYear int            `json:"releaseYear,omitempty"`
	Provider    string         `json:"provider"`
	ExternalID  string         `json:"externalId"`
	Metadata    map[string]any `json:"metadata,omitempty"`
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

type bookSearchProvider struct {
	apiKey string
}

func (provider bookSearchProvider) Search(query string) ([]ExternalOption, error) {
	return SearchBooks(query, provider.apiKey)
}

// SearchProviderForType returns the search strategy for a poll type.
func SearchProviderForType(pollType string, tmdbAPIKey string, googleBooksAPIKey string) SearchProvider {
	providers := map[string]SearchProvider{
		"movie": movieSearchProvider{apiKey: tmdbAPIKey},
		"book":  bookSearchProvider{apiKey: googleBooksAPIKey},
	}

	return providers[pollType]
}

type tmdbSearchResponse struct {
	Page    int                `json:"page"`
	Results []tmdbSearchResult `json:"results"`
}

type tmdbSearchResult struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
	Overview    string `json:"overview"`
	PosterPath  string `json:"poster_path"`
}

type googleBooksResponse struct {
	Items []googleBookItem `json:"items"`
}

type googleBookItem struct {
	ID         string           `json:"id"`
	VolumeInfo googleVolumeInfo `json:"volumeInfo"`
}

type googleVolumeInfo struct {
	Title               string                     `json:"title"`
	Authors             []string                   `json:"authors"`
	Publisher           string                     `json:"publisher"`
	PublishedDate       string                     `json:"publishedDate"`
	Description         string                     `json:"description"`
	ImageLinks          googleImageLinks           `json:"imageLinks"`
	IndustryIdentifiers []googleIndustryIdentifier `json:"industryIdentifiers"`
}

type googleImageLinks struct {
	Thumbnail      string `json:"thumbnail"`
	SmallThumbnail string `json:"smallThumbnail"`
}

type googleIndustryIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
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

	provider := SearchProviderForType(pollType, server.TMDBAPIKey, server.GoogleBooksKey)
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
	var searchResponse tmdbSearchResponse

	// Convert JSON from the API into Go structs.
	err = json.NewDecoder(response.Body).Decode(&searchResponse)
	if err != nil {
		return nil, err
	}

	options := make([]ExternalOption, 0, len(searchResponse.Results))
	for _, movie := range searchResponse.Results {
		posterURL := ""
		if movie.PosterPath != "" {
			// TMDB gives a relative poster path, so add the image host to make it usable.
			posterURL = "https://image.tmdb.org/t/p/w500" + movie.PosterPath
		}

		options = append(options, ExternalOption{
			ID:          strconv.Itoa(movie.ID),
			Title:       movie.Title,
			ReleaseDate: movie.ReleaseDate,
			Overview:    movie.Overview,
			PosterPath:  movie.PosterPath,
			PosterURL:   posterURL,
			ImageURL:    posterURL,
			ReleaseYear: yearFromDate(movie.ReleaseDate),
			Provider:    "tmdb",
			ExternalID:  strconv.Itoa(movie.ID),
		})
	}

	// Return only the movie results.
	// The caller doesn't care about page numbers.
	return options, nil
}

// SearchBooks calls Google Books and converts book volumes into generic option suggestions.
func SearchBooks(query string, apiKey string) ([]ExternalOption, error) {
	escapedQuery := url.QueryEscape(query)
	requestURL := "https://www.googleapis.com/books/v1/volumes?q=" + escapedQuery
	if apiKey != "" {
		requestURL += "&key=" + url.QueryEscape(apiKey)
	}

	response, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var booksResponse googleBooksResponse
	if err := json.NewDecoder(response.Body).Decode(&booksResponse); err != nil {
		return nil, err
	}

	options := make([]ExternalOption, 0, len(booksResponse.Items))
	for _, book := range booksResponse.Items {
		info := book.VolumeInfo
		imageURL := info.ImageLinks.Thumbnail
		if imageURL == "" {
			imageURL = info.ImageLinks.SmallThumbnail
		}

		isbn := firstISBN(info.IndustryIdentifiers)
		metadata := map[string]any{
			"authors":   info.Authors,
			"isbn":      isbn,
			"publisher": info.Publisher,
		}

		options = append(options, ExternalOption{
			ID:          book.ID,
			Title:       info.Title,
			ReleaseDate: info.PublishedDate,
			Overview:    info.Description,
			PosterURL:   imageURL,
			ImageURL:    imageURL,
			ReleaseYear: yearFromDate(info.PublishedDate),
			Provider:    "google-books",
			ExternalID:  book.ID,
			Metadata:    metadata,
		})
	}

	return options, nil
}

func firstISBN(identifiers []googleIndustryIdentifier) string {
	for _, identifier := range identifiers {
		if strings.Contains(identifier.Type, "ISBN") && identifier.Identifier != "" {
			return identifier.Identifier
		}
	}

	return ""
}

func yearFromDate(value string) int {
	if len(value) < 4 {
		return 0
	}

	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return 0
	}

	return year
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
