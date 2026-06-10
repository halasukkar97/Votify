package api

import (
	"movie-vote/database"
	"movie-vote/movie"
	"movie-vote/poll"
	"movie-vote/user"
	"movie-vote/vote"
)

// FindPollByID searches the PostgreSQL polls table for a poll with the given ID.
// It returns the poll and true when a row exists, otherwise nil and false.
func FindPollByID(pollID string) (*poll.Poll, bool) {
	var foundPoll poll.Poll

	// QueryRow expects one row back. Scan copies each selected database column
	// into the matching field on foundPoll.
	err := database.DB.QueryRow(
		`SELECT id, name, is_closed, max_votes_per_person, deadline
		FROM polls
		WHERE id = $1`,
		pollID,
	).Scan(
		&foundPoll.ID,
		&foundPoll.Name,
		&foundPoll.IsClosed,
		&foundPoll.MaxVotesPerPerson,
		&foundPoll.Deadline,
	)

	if err != nil {
		return nil, false
	}

	return &foundPoll, true
}

// SavePoll stores a newly created poll in PostgreSQL.
// Returning an error lets the HTTP handler send a clear failure response.
func SavePoll(poll poll.Poll) error {
	_, err := database.DB.Exec(
		`INSERT INTO polls
		(id, name, is_closed, max_votes_per_person, deadline)
		VALUES ($1, $2, $3, $4, $5)`,
		poll.ID,
		poll.Name,
		poll.IsClosed,
		poll.MaxVotesPerPerson,
		poll.Deadline,
	)

	return err
}

// SaveMovie stores a newly created movie in PostgreSQL.
// Returning an error lets the HTTP handler report database save failures.
func SaveMovie(movie movie.Movie) error {
	_, err := database.DB.Exec(
		`INSERT INTO movies
		(id, poll_id, title, release_year, description)
		VALUES ($1, $2, $3, $4, $5)`,
		movie.ID,
		movie.PollID,
		movie.Title,
		movie.ReleaseYear,
		movie.Description,
	)

	return err
}

// SaveUser stores a newly created user in PostgreSQL.
// Returning an error lets the HTTP handler send a clear failure response.
func SaveUser(user user.User) error {
	_, err := database.DB.Exec(
		"INSERT INTO users (id, name) VALUES ($1, $2)",
		user.ID,
		user.Name,
	)

	return err
}

// SaveVote stores a valid vote in memory after the poll accepts it.
func SaveVote(vote vote.Vote) {
	votes = append(votes, vote)
}

// PollExists checks PostgreSQL for a poll ID without loading the full poll.
// SELECT EXISTS returns one true/false value, which is cheaper than reading every column.
func PollExists(pollID string) (bool, error) {
	var exists bool

	err := database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM polls WHERE id = $1)",
		pollID,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
