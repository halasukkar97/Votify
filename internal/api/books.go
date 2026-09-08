package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var openLibraryClient = &http.Client{Timeout: 15 * time.Second}
var openLibraryRequests struct {
	sync.Mutex
	last time.Time
}
var openLibraryWorkKey = regexp.MustCompile(`^/works/OL[0-9]+W$`)

// Pace public API calls across searches and selected-work requests.
func openLibraryGet(path string, target any) error {
	openLibraryRequests.Lock()
	if delay := time.Second - time.Since(openLibraryRequests.last); delay > 0 {
		time.Sleep(delay)
	}
	openLibraryRequests.last = time.Now()
	openLibraryRequests.Unlock()
	request, err := http.NewRequest(http.MethodGet, "https://openlibrary.org"+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "Votify/1.0 (book poll search)")
	request.Header.Set("Accept", "application/json")
	response, err := openLibraryClient.Do(request)
	if err != nil {
		return fmt.Errorf("Open Library request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return fmt.Errorf("Open Library request failed: status=%d, body=%s", response.StatusCode, body)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("Open Library response parsing failed: %w", err)
	}
	return nil
}

type openLibraryDoc struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	Authors          []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	CoverID          int64    `json:"cover_i"`
	ISBN             []string `json:"isbn"`
	Publisher        []string `json:"publisher"`
}

// SearchBooks uses the public Open Library search API; no API key is needed.
func SearchBooks(query string) ([]ExternalOption, error) {
	params := url.Values{
		"q": {query}, "limit": {"10"},
		"fields": {"key,title,author_name,first_publish_year,cover_i,isbn,publisher"},
	}
	var result struct {
		Docs []openLibraryDoc `json:"docs"`
	}
	if err := openLibraryGet("/search.json?"+params.Encode(), &result); err != nil {
		return nil, err
	}
	options := make([]ExternalOption, 0, len(result.Docs))
	for _, doc := range result.Docs {
		if strings.TrimSpace(doc.Title) == "" || !openLibraryWorkKey.MatchString(doc.Key) {
			continue
		}
		authors := doc.Authors
		if authors == nil {
			authors = []string{}
		}
		isbn := preferredISBN(doc.ISBN)
		metadata := map[string]any{
			"authors": authors, "openLibraryKey": doc.Key,
			"openLibraryUrl": "https://openlibrary.org" + doc.Key,
			"goodreadsUrl":   goodreadsSearchURL(isbn, doc.Title, authors),
		}
		if isbn != "" {
			metadata["isbn"] = isbn
		}
		for _, publisher := range doc.Publisher {
			if publisher != "" {
				metadata["publisher"] = publisher
				break
			}
		}
		imageURL := ""
		if doc.CoverID > 0 {
			imageURL = "https://covers.openlibrary.org/b/id/" + strconv.FormatInt(doc.CoverID, 10) + "-L.jpg"
		}
		year := max(0, doc.FirstPublishYear)
		options = append(options, ExternalOption{
			ID: doc.Key, Title: doc.Title, ReleaseYear: year,
			ImageURL: imageURL, PosterURL: imageURL,
			Provider: "open-library", ExternalID: doc.Key, Metadata: metadata,
		})
	}
	return options, nil
}

func preferredISBN(values []string) string {
	for _, length := range []int{13, 10} {
		for _, value := range values {
			value = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "-", ""), " ", "")
			if len(value) != length {
				continue
			}
			valid := true
			for i, char := range value {
				if (char < '0' || char > '9') && !(length == 10 && i == 9 && (char == 'X' || char == 'x')) {
					valid = false
					break
				}
			}
			if valid {
				return strings.ToUpper(value)
			}
		}
	}
	return ""
}

// BookDetailsHandler loads only a selected work, avoiding one request per result.
func (server *Server) BookDetailsHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if !openLibraryWorkKey.MatchString(key) {
		http.Error(w, "invalid Open Library work key", http.StatusBadRequest)
		return
	}
	var work struct {
		Description json.RawMessage `json:"description"`
	}
	if err := openLibraryGet(key+".json", &work); err != nil {
		log.Printf("book details failed for %q: %v", key, err)
		http.Error(w, "failed to load book details", http.StatusBadGateway)
		return
	}
	description := ""
	if len(work.Description) > 0 {
		if err := json.Unmarshal(work.Description, &description); err != nil {
			var value struct {
				Value string `json:"value"`
			}
			if json.Unmarshal(work.Description, &value) == nil {
				description = value.Value
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]string{"description": description})
}
