package main

import (
	"fmt"
	"log"
	"net/http"
	"votify/internal/api"
	"votify/internal/config"
	"votify/internal/database"
)

// main is the first function Go runs when the application starts.
// It loads environment variables, connects to the database, registers every
// HTTP route, and then starts listening for requests on port 8080.
func main() {

	cfg := config.Load()

	err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database connected!")

	// http.HandleFunc connects a URL path to the function that should handle it.
	http.HandleFunc("/", MovieVoteHandler)
	http.HandleFunc("/polls", api.PollsHandler)
	http.HandleFunc("/users", api.UsersHandler)
	http.HandleFunc("/users/", api.UserByIDHandler)
	http.HandleFunc("/movies", api.MoviesHandler)
	http.HandleFunc("/votes", api.CreateVoteHandler)
	http.HandleFunc("/results", api.ResultsHandler)
	http.HandleFunc("/movies/search", api.SearchMoviesHandler)
	http.HandleFunc("/polls/", api.PollByIDHandler)

	log.Fatal(http.ListenAndServe(":"+cfg.Port, enableCORS(http.DefaultServeMux, cfg.AllowedOrigins)))
}

// MovieVoteHandler handles the root route and returns a simple health message.
func MovieVoteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Movie Vote API updated")
}

// enableCORS allows the known frontend apps to call this backend.
// Without this, browsers block requests from the React app to the Go API.
func enableCORS(next http.Handler, allowedOrigins map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// OPTIONS is the browser's preflight check before the real request.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
