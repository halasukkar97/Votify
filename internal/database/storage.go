package database

import (
	"database/sql"
	"log"
	"votify/internal/domain"
)

// FindPollByID searches for a poll by the internal UUID.
// This ID is mainly for the database and backend relations.
func FindPollByID(pollID string) (*domain.Poll, bool) {
	foundPoll, found, err := FindPollByIDWithError(pollID)
	if err != nil {
		log.Printf("FindPollByID failed for identifier %q: %v", pollID, err)
	}

	return foundPoll, found
}

// FindPollByCode searches for a poll by the public 8-digit poll code.
// This is the code users will type when they join a poll.
func FindPollByCode(pollCode string) (*domain.Poll, bool) {
	foundPoll, found, err := FindPollByCodeWithError(pollCode)
	if err != nil {
		log.Printf("FindPollByCode failed for identifier %q: %v", pollCode, err)
	}

	return foundPoll, found
}

// FindPollByIDWithError searches by internal UUID and returns the database error to callers that need it.
func FindPollByIDWithError(pollID string) (*domain.Poll, bool, error) {
	return findPollByQuery(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline
		FROM polls
		WHERE id = $1`,
		pollID,
	)
}

// FindPollByCodeWithError searches by public poll code and returns the database error to callers that need it.
func FindPollByCodeWithError(pollCode string) (*domain.Poll, bool, error) {
	return findPollByQuery(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline
		FROM polls
		WHERE poll_code = $1`,
		pollCode,
	)
}

// findPollByQuery loads a poll using one of the fixed poll lookup queries.
func findPollByQuery(query string, value string) (*domain.Poll, bool, error) {
	var foundPoll domain.Poll

	// QueryRow expects one row back. Scan copies each selected database column
	// into the matching field on foundPoll.
	err := DB.QueryRow(query, value).Scan(
		&foundPoll.ID,
		&foundPoll.PollCode,
		&foundPoll.Name,
		&foundPoll.IsClosed,
		&foundPoll.IsVotingActive,
		&foundPoll.MaxVotesPerPerson,
		&foundPoll.Deadline,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}

		return nil, false, err
	}

	// A single poll response should include its related movies and votes.
	movies, err := GetMoviesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false, err
	}

	votes, err := GetVotesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false, err
	}

	foundPoll.Votes = votes
	foundPoll.Movies = movies

	return &foundPoll, true, nil
}

// SavePoll stores a newly created poll in PostgreSQL.
// We store both the internal UUID and the public poll code.
func SavePoll(poll domain.Poll) error {
	_, err := DB.Exec(
		`INSERT INTO polls
		(id, poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		poll.ID,
		poll.PollCode,
		poll.Name,
		poll.IsClosed,
		poll.IsVotingActive,
		poll.MaxVotesPerPerson,
		poll.Deadline,
	)

	return err
}

// ActivateVoting starts the voting phase for a poll identified by its public code.
func ActivateVoting(pollCode string) error {
	_, err := DB.Exec(
		"UPDATE polls SET is_voting_active = TRUE WHERE poll_code = $1",
		pollCode,
	)

	return err
}

// PollCodeExists checks if a public 8-digit poll code is already used.
// This prevents two polls from getting the same join code.
func PollCodeExists(pollCode string) (bool, error) {
	var exists bool

	err := DB.QueryRow(
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
func SaveMovie(movie domain.Movie) error {
	_, err := DB.Exec(
		`INSERT INTO movies
		(id, poll_id, title, release_year, description, poster_url)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		movie.ID,
		movie.PollID,
		movie.Title,
		movie.ReleaseYear,
		movie.Description,
		movie.PosterURL,
	)

	return err
}

// SaveUser stores a newly created user in PostgreSQL.
// Returning an error lets the HTTP handler send a clear failure response.
func SaveUser(user domain.User) error {
	_, err := DB.Exec(
		"INSERT INTO users (id, name) VALUES ($1, $2)",
		user.ID,
		user.Name,
	)

	return err
}

// UpdateUserName changes the display name for an existing user ID.
// Keeping the same ID means old votes still belong to the same person.
func UpdateUserName(userID string, name string) (domain.User, error) {
	var updatedUser domain.User

	err := DB.QueryRow(
		"UPDATE users SET name = $1 WHERE id = $2 RETURNING id, name",
		name,
		userID,
	).Scan(&updatedUser.ID, &updatedUser.Name)

	return updatedUser, err
}

// SaveVote stores a valid vote in PostgreSQL after the poll accepts it.
// The votes table stores the vote owner, and vote_movies stores the selected movies.
func SaveVote(vote domain.Vote) error {
	// A transaction keeps the vote and its movie selections together.
	// If any insert fails, Rollback cancels everything from this SaveVote call.
	tx, err := DB.Begin()
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

	err := DB.QueryRow(
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
	rows, err := DB.Query(
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

// GetAllPolls reads every poll row from PostgreSQL and converts each row into a domain.Poll.
// It also loads each poll's movies and votes so clients can see the full poll state.
func GetAllPolls() ([]domain.Poll, error) {
	// Query returns rows, which must be scanned one at a time.
	rows, err := DB.Query(
		"SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline FROM polls",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	polls := make([]domain.Poll, 0)

	// rows.Next moves through the result set one database row at a time.
	for rows.Next() {
		var currentPoll domain.Poll

		// Scan copies the current row's columns into the poll struct fields.
		err := rows.Scan(
			&currentPoll.ID,
			&currentPoll.PollCode,
			&currentPoll.Name,
			&currentPoll.IsClosed,
			&currentPoll.IsVotingActive,
			&currentPoll.MaxVotesPerPerson,
			&currentPoll.Deadline,
		)

		if err != nil {
			return nil, err
		}

		// Load the movies connected to this poll before adding it to the response list.
		movies, err := GetMoviesByPollID(currentPoll.ID)
		if err != nil {
			return nil, err
		}

		currentPoll.Movies = movies

		// Load the votes connected to this poll, including the selected movie IDs.
		votes, err := GetVotesByPollID(currentPoll.ID)
		if err != nil {
			return nil, err
		}

		currentPoll.Votes = votes

		polls = append(polls, currentPoll)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return polls, nil
}

// GetAllMovies reads every movie row from PostgreSQL and converts each row into a domain.Movie.
func GetAllMovies() ([]domain.Movie, error) {
	// Query returns rows, which must be scanned one at a time.
	rows, err := DB.Query(
		"SELECT id, poll_id, title, release_year, description, COALESCE(poster_url, '') AS poster_url FROM movies",
	)

	if err != nil {
		rows, err = DB.Query(
			"SELECT id, poll_id, title, release_year, description FROM movies",
		)
		if err != nil {
			return nil, err
		}

		return scanMovieRows(rows, false)
	}

	return scanMovieRows(rows, true)
}

// GetMoviesByPollID reads only the movies that belong to one poll.
// Poll listing uses this to include each poll's movie options in the response.
func GetMoviesByPollID(pollID string) ([]domain.Movie, error) {
	// The WHERE clause filters the movies table down to the requested poll ID.
	rows, err := DB.Query(
		"SELECT id, poll_id, title, release_year, description, COALESCE(poster_url, '') AS poster_url FROM movies WHERE poll_id = $1",
		pollID,
	)

	if err != nil {
		rows, err = DB.Query(
			"SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id = $1",
			pollID,
		)
		if err != nil {
			return nil, err
		}

		return scanMovieRows(rows, false)
	}

	return scanMovieRows(rows, true)
}

func scanMovieRows(rows *sql.Rows, hasPosterURL bool) ([]domain.Movie, error) {
	defer rows.Close()

	movies := make([]domain.Movie, 0)

	// Build one movie struct for each returned database row.
	for rows.Next() {
		var currentMovie domain.Movie
		var err error

		if hasPosterURL {
			err = rows.Scan(
				&currentMovie.ID,
				&currentMovie.PollID,
				&currentMovie.Title,
				&currentMovie.ReleaseYear,
				&currentMovie.Description,
				&currentMovie.PosterURL,
			)
		} else {
			err = rows.Scan(
				&currentMovie.ID,
				&currentMovie.PollID,
				&currentMovie.Title,
				&currentMovie.ReleaseYear,
				&currentMovie.Description,
			)
		}

		if err != nil {
			return nil, err
		}

		movies = append(movies, currentMovie)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return movies, nil
}

// GetVotesByPollID reads all votes submitted for one poll.
// It also loads each vote's selected movie IDs from the vote_movies table.
func GetVotesByPollID(pollID string) ([]domain.Vote, error) {
	// First load the vote rows for this poll.
	rows, err := DB.Query(
		"SELECT id, poll_id, user_id FROM votes WHERE poll_id = $1",
		pollID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	votes := make([]domain.Vote, 0)

	// Build one vote struct for each returned database row.
	for rows.Next() {
		var currentVote domain.Vote

		err := rows.Scan(
			&currentVote.ID,
			&currentVote.PollID,
			&currentVote.UserID,
		)

		if err != nil {
			return nil, err
		}

		// The selected movies live in the vote_movies join table.
		movieIDs, err := GetMovieIDsByVoteID(currentVote.ID)
		if err != nil {
			return nil, err
		}

		currentVote.MovieIDs = movieIDs
		votes = append(votes, currentVote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return votes, nil
}

// GetAllUsers reads every user row from PostgreSQL and converts each row into a domain.User.
func GetAllUsers() ([]domain.User, error) {

	rows, err := DB.Query(
		"SELECT id, name FROM users",
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	users := make([]domain.User, 0)

	// rows.Next moves through the result set one database row at a time.
	for rows.Next() {
		var currentUser domain.User

		// Scan copies the current row's columns into Go variables.
		err := rows.Scan(
			&currentUser.ID,
			&currentUser.Name,
		)

		if err != nil {
			return nil, err
		}

		users = append(users, currentUser)
	}

	return users, nil
}
