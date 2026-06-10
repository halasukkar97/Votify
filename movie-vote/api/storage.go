package api

import (
	"movie-vote/vote"
)

// votes is still simple in-memory storage.
// Data saved here is lost when the server restarts.
var votes []vote.Vote
