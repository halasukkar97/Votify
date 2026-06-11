package api

import (
	"votify/database"
	"votify/movie"
	"votify/poll"
	"votify/user"
	"votify/vote"
)

// FindPollByID searches for a poll by the internal UUID.
// This ID is mainly for the database and backend relations.
func FindPollByID(pollID string) (*poll.Poll, bool) {
	return findPollByColumn("id", pollID)
}

// FindPollByCode searches for a poll by the public 8-digit poll code.
// This is the code users will type when they join a poll.
func FindPollByCode(pollCode string) (*poll.Poll, bool) {
	return findPollByColumn("poll_code", pollCode)
}

// findPollByColumn loads a poll by one allowed database column.
// We keep this helper private so callers cannot pass random SQL columns.
func findPollByColumn(column string, value string) (*poll.Poll, bool) {
	var foundPoll poll.Poll

	// QueryRow expects one row back. Scan copies each selected database column
	// into the matching field on foundPoll.
	err := database.DB.QueryRow(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, max_votes_per_person, deadline
		FROM polls
		WHERE `+column+` = $1`,
		value,
	).Scan(
		&foundPoll.ID,
		&foundPoll.PollCode,
		&foundPoll.Name,
		&foundPoll.IsClosed,
		&foundPoll.MaxVotesPerPerson,
		&foundPoll.Deadline,
	)

	if err != nil {
		return nil, false
	}

	// A single poll response should include its related movies and votes.
	movies, err := GetMoviesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false
	}

	votes, err := GetVotesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false
	}

	foundPoll.Votes = votes
	foundPoll.Movies = movies

	return &foundPoll, true
}

// SavePoll stores a newly created poll in PostgreSQL.
// We store both the internal UUID and the public poll code.
func SavePoll(poll poll.Poll) error {
	_, err := database.DB.Exec(
		`INSERT INTO polls
		(id, poll_code, name, is_closed, max_votes_per_person, deadline)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		poll.ID,
		poll.PollCode,
		poll.Name,
		poll.IsClosed,
		poll.MaxVotesPerPerson,
		poll.Deadline,
	)

	return err
}

// PollCodeExists checks if a public 8-digit poll code is already used.
// This prevents two polls from getting the same join code.
func PollCodeExists(pollCode string) (bool, error) {
	var exists bool

	err := database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM polls WHERE poll_code = $1)",
		pollCode,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
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

// SaveVote stores a valid vote in PostgreSQL after the poll accepts it.
// The votes table stores the vote owner, and vote_movies stores the selected movies.
func SaveVote(vote vote.Vote) error {
	// A transaction keeps the vote and its movie selections together.
	// If any insert fails, Rollback cancels everything from this SaveVote call.
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}

	// Insert the vote itself.
	_, err = tx.Exec(
		"INSERT INTO votes (id, poll_id, user_id) VALUES ($1, $2, $3)",
		vote.ID,
		vote.PollID,
		vote.UserID,
	)

	if err != nil {
		tx.Rollback()
		return err
	}

	// Insert one row per selected movie.
	for _, movieID := range vote.MovieIDs {
		_, err = tx.Exec(
			"INSERT INTO vote_movies (vote_id, movie_id) VALUES ($1, $2)",
			vote.ID,
			movieID,
		)

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
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

// GetMovieIDsByVoteID reads the selected movie IDs for one vote.
// Votes and movies are connected through the vote_movies join table.
func GetMovieIDsByVoteID(voteID string) ([]string, error) {
	// Each row contains one movie selected by this vote.
	rows, err := database.DB.Query(
		"SELECT movie_id FROM vote_movies WHERE vote_id = $1",
		voteID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var movieIDs []string

	// Collect every movie_id into a plain []string for the vote model.
	for rows.Next() {
		var movieID string

		err := rows.Scan(&movieID)
		if err != nil {
			return nil, err
		}

		movieIDs = append(movieIDs, movieID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movieIDs, nil
}
