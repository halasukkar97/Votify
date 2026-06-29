package main

import (
	"fmt"
	"log"
	"net/http"
	"votify/internal/api"
	"votify/internal/config"
	"votify/internal/database"
	"votify/internal/repository"
	"votify/internal/service"
)

// main is the first function Go runs when the application starts.
// It loads environment variables, connects to the database, registers every
// HTTP route, and then starts listening for requests on port 8080.
func main() {

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database connected!")

	store := repository.NewStore(db)
	appService := service.New(store)
	server := api.NewServer(appService, cfg.TMDBAPIKey)

	// http.HandleFunc connects a URL path to the function that should handle it.
	http.HandleFunc("/", MovieVoteHandler)
	http.HandleFunc("/polls", server.PollsHandler)
	http.HandleFunc("/users", server.UsersHandler)
	http.HandleFunc("/users/", server.UserByIDHandler)
	http.HandleFunc("/movies", server.MoviesHandler)
	http.HandleFunc("/votes", server.CreateVoteHandler)
	http.HandleFunc("/results", server.ResultsHandler)
	http.HandleFunc("/movies/search", server.SearchMoviesHandler)
	http.HandleFunc("/polls/", server.PollByIDHandler)

	handler := recoverPanic(logRequests(enableCORS(http.DefaultServeMux, cfg.AllowedOrigins)))

	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
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

// logRequests records one line for each HTTP request.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// recoverPanic keeps an unexpected panic from crashing the server process.
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic while handling %s %s: %v", r.Method, r.URL.Path, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
