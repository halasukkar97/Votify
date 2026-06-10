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

	movies, err := GetMoviesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false
	}

	foundPoll.Movies = movies

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

// SaveVote stores a valid vote in PostgreSQL after the poll accepts it.
// The votes table stores the vote owner, and vote_movies stores the selected movies.
func SaveVote(vote vote.Vote) error {
	// Insert the vote itself.
	_, err := database.DB.Exec(
		"INSERT INTO votes (id, poll_id, user_id) VALUES ($1, $2, $3)",
		vote.ID,
		vote.PollID,
		vote.UserID,
	)

	if err != nil {
		return err
	}

	// Insert one row per selected movie.
	for _, movieID := range vote.MovieIDs {
		_, err := database.DB.Exec(
			"INSERT INTO vote_movies (vote_id, movie_id) VALUES ($1, $2)",
			vote.ID,
			movieID,
		)

		if err != nil {
			return err
		}
	}
	return nil
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
