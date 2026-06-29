package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"votify/api"
	"votify/database"

	"github.com/joho/godotenv"
)

// main is the first function Go runs when the application starts.
// It loads environment variables, connects to the database, registers every
// HTTP route, and then starts listening for requests on port 8080.
func main() {

	// godotenv.Load reads key/value pairs from .env so os.Getenv can use them.
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	err = database.Connect()
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(":"+port, enableCORS(http.DefaultServeMux)))
}

// MovieVoteHandler handles the root route and returns a simple health message.
func MovieVoteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Movie Vote API updated")
}

// enableCORS allows the known frontend apps to call this backend.
// Without this, browsers block requests from the React app to the Go API.
func enableCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:5173":         true,
		"https://votify-six.vercel.app": true,
	}

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
