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
	http.HandleFunc("/movies", api.MoviesHandler)
	http.HandleFunc("/votes", api.CreateVoteHandler)
	http.HandleFunc("/results", api.ResultsHandler)
	http.HandleFunc("/movies/search", api.SearchMoviesHandler)
	http.HandleFunc("/polls/", api.PollByIDHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// MovieVoteHandler handles the root route and returns a simple health message.
func MovieVoteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Movie Vote API updated")
}
