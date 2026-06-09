package main

import (
	"fmt"
	"log"
	"movie-vote/api"
	"movie-vote/database"
	"net/http"
)

func main() {

	err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database connected!")

	http.HandleFunc("/", MovieVoteHandler)
	http.HandleFunc("/polls", api.PollsHandler)
	http.HandleFunc("/users", api.UsersHandler)
	http.HandleFunc("/movies", api.MoviesHandler)
	http.HandleFunc("/votes", api.CreateVoteHandler)
	http.HandleFunc("/results", api.ResultsHandler)

	http.ListenAndServe(":8080", nil)
}

func MovieVoteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Movie Vote API updated")
}
