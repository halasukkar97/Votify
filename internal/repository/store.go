package repository

import (
	"database/sql"
	"encoding/json"
	"log"
	"votify/internal/domain"
)

// Store groups PostgreSQL repositories behind one small dependency.
// Each method owns SQL for one persistence operation.
type Store struct {
	DB *sql.DB
}

// NewStore creates a repository store backed by the provided database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{DB: db}
}

// FindPollByID searches for a poll by the internal UUID.
// This ID is mainly for the database and backend relations.
func (store *Store) FindPollByID(pollID string) (*domain.Poll, bool) {
	foundPoll, found, err := store.FindPollByIDWithError(pollID)
	if err != nil {
		log.Printf("FindPollByID failed for identifier %q: %v", pollID, err)
	}

	return foundPoll, found
}

// FindPollByCode searches for a poll by the public 8-digit poll code.
// This is the code users will type when they join a poll.
func (store *Store) FindPollByCode(pollCode string) (*domain.Poll, bool) {
	foundPoll, found, err := store.FindPollByCodeWithError(pollCode)
	if err != nil {
		log.Printf("FindPollByCode failed for identifier %q: %v", pollCode, err)
	}

	return foundPoll, found
}

// FindPollByIDWithError searches by internal UUID and returns the database error to callers that need it.
func (store *Store) FindPollByIDWithError(pollID string) (*domain.Poll, bool, error) {
	foundPoll, found, err := store.findPollByQuery(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE(poll_type, 'movie') AS poll_type
		FROM polls
		WHERE id = $1`,
		pollID,
	)
	if err == nil || err == sql.ErrNoRows {
		return foundPoll, found, err
	}

	return store.findLegacyPollByQuery(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline
		FROM polls
		WHERE id = $1`,
		pollID,
	)
}

// FindPollByCodeWithError searches by public poll code and returns the database error to callers that need it.
func (store *Store) FindPollByCodeWithError(pollCode string) (*domain.Poll, bool, error) {
	foundPoll, found, err := store.findPollByQuery(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE(poll_type, 'movie') AS poll_type
		FROM polls
		WHERE poll_code = $1`,
		pollCode,
	)
	if err == nil || err == sql.ErrNoRows {
		return foundPoll, found, err
	}

	return store.findLegacyPollByQuery(
		`SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline
		FROM polls
		WHERE poll_code = $1`,
		pollCode,
	)
}

// findPollByQuery loads a poll using one of the fixed poll lookup queries.
func (store *Store) findPollByQuery(query string, value string) (*domain.Poll, bool, error) {
	var foundPoll domain.Poll

	// QueryRow expects one row back. Scan copies each selected database column
	// into the matching field on foundPoll.
	err := store.DB.QueryRow(query, value).Scan(
		&foundPoll.ID,
		&foundPoll.PollCode,
		&foundPoll.Name,
		&foundPoll.IsClosed,
		&foundPoll.IsVotingActive,
		&foundPoll.MaxVotesPerPerson,
		&foundPoll.Deadline,
		&foundPoll.PollType,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}

		return nil, false, err
	}

	// A single poll response should include its related options and votes.
	options, err := store.GetOptionsByPollID(foundPoll.ID)
	if err != nil {
		return nil, false, err
	}

	votes, err := store.GetVotesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false, err
	}

	foundPoll.Votes = votes
	foundPoll.Options = options
	foundPoll.Movies = options

	return &foundPoll, true, nil
}

func (store *Store) findLegacyPollByQuery(query string, value string) (*domain.Poll, bool, error) {
	var foundPoll domain.Poll

	err := store.DB.QueryRow(query, value).Scan(
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

	foundPoll.PollType = "movie"
	options, err := store.GetOptionsByPollID(foundPoll.ID)
	if err != nil {
		return nil, false, err
	}
	votes, err := store.GetVotesByPollID(foundPoll.ID)
	if err != nil {
		return nil, false, err
	}

	foundPoll.Options = options
	foundPoll.Movies = options
	foundPoll.Votes = votes
	return &foundPoll, true, nil
}

// SavePoll stores a newly created poll in PostgreSQL.
// We store both the internal UUID and the public poll code.
func (store *Store) SavePoll(poll domain.Poll) error {
	_, err := store.DB.Exec(
		`INSERT INTO polls
		(id, poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, poll_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		poll.ID,
		poll.PollCode,
		poll.Name,
		poll.IsClosed,
		poll.IsVotingActive,
		poll.MaxVotesPerPerson,
		poll.Deadline,
		poll.PollType,
	)
	if err == nil {
		return nil
	}

	_, err = store.DB.Exec(
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
func (store *Store) ActivateVoting(pollCode string) error {
	_, err := store.DB.Exec(
		"UPDATE polls SET is_voting_active = TRUE WHERE poll_code = $1",
		pollCode,
	)

	return err
}

// PollCodeExists checks if a public 8-digit poll code is already used.
// This prevents two polls from getting the same join code.
func (store *Store) PollCodeExists(pollCode string) (bool, error) {
	var exists bool

	err := store.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM polls WHERE poll_code = $1)",
		pollCode,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

// SaveOption stores a newly created option in PostgreSQL.
// It falls back to the legacy movies table until older databases are migrated.
func (store *Store) SaveOption(option domain.Option) error {
	metadataJSON, err := json.Marshal(option.Metadata)
	if err != nil {
		return err
	}
	imageURL := option.ImageURL
	if imageURL == "" {
		imageURL = option.PosterURL
	}

	_, err = store.DB.Exec(
		`INSERT INTO options
		(id, poll_id, title, description, image_url, release_year, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		option.ID,
		option.PollID,
		option.Title,
		option.Description,
		imageURL,
		option.ReleaseYear,
		string(metadataJSON),
	)
	if err == nil {
		return nil
	}

	_, err = store.DB.Exec(
		`INSERT INTO movies
		(id, poll_id, title, release_year, description, poster_url)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		option.ID,
		option.PollID,
		option.Title,
		option.ReleaseYear,
		option.Description,
		imageURL,
	)

	return err
}

// SaveMovie keeps older callers working while options become the main model.
func (store *Store) SaveMovie(option domain.Movie) error {
	return store.SaveOption(option)
}

// SaveUser stores a newly created user in PostgreSQL.
// Returning an error lets the HTTP handler send a clear failure response.
func (store *Store) SaveUser(user domain.User) error {
	_, err := store.DB.Exec(
		"INSERT INTO users (id, name) VALUES ($1, $2)",
		user.ID,
		user.Name,
	)

	return err
}

// UpdateUserName changes the display name for an existing user ID.
// Keeping the same ID means old votes still belong to the same person.
func (store *Store) UpdateUserName(userID string, name string) (domain.User, error) {
	var updatedUser domain.User

	err := store.DB.QueryRow(
		"UPDATE users SET name = $1 WHERE id = $2 RETURNING id, name",
		name,
		userID,
	).Scan(&updatedUser.ID, &updatedUser.Name)

	return updatedUser, err
}

// SaveVote stores a valid vote in PostgreSQL after the poll accepts it.
// The votes table stores the vote owner, and vote_options stores the selected options.
func (store *Store) SaveVote(vote domain.Vote) error {
	// A transaction keeps the vote and its movie selections together.
	// If any insert fails, Rollback cancels everything from this SaveVote call.
	tx, err := store.DB.Begin()
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

	optionIDs := vote.OptionIDs
	if len(optionIDs) == 0 {
		optionIDs = vote.MovieIDs
	}

	// Insert one row per selected option.
	for _, optionID := range optionIDs {
		_, err = tx.Exec(
			"INSERT INTO vote_options (vote_id, option_id) VALUES ($1, $2)",
			vote.ID,
			optionID,
		)
		if err != nil {
			_, err = tx.Exec(
				"INSERT INTO vote_movies (vote_id, movie_id) VALUES ($1, $2)",
				vote.ID,
				optionID,
			)
		}

		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// PollExists checks PostgreSQL for a poll ID without loading the full poll.
// SELECT EXISTS returns one true/false value, which is cheaper than reading every column.
func (store *Store) PollExists(pollID string) (bool, error) {
	var exists bool

	err := store.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM polls WHERE id = $1)",
		pollID,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetOptionIDsByVoteID reads the selected movie IDs for one vote.
// Votes and options are connected through the vote_options join table.
func (store *Store) GetOptionIDsByVoteID(voteID string) ([]string, error) {
	// Each row contains one movie selected by this vote.
	rows, err := store.DB.Query(
		"SELECT option_id FROM vote_options WHERE vote_id = $1",
		voteID,
	)
	if err != nil {
		rows, err = store.DB.Query(
			"SELECT movie_id FROM vote_movies WHERE vote_id = $1",
			voteID,
		)
		if err != nil {
			return nil, err
		}
	}

	defer rows.Close()

	var optionIDs []string

	// Collect every option_id into a plain []string for the vote model.
	for rows.Next() {
		var optionID string

		err := rows.Scan(&optionID)
		if err != nil {
			return nil, err
		}

		optionIDs = append(optionIDs, optionID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return optionIDs, nil
}

// GetAllPolls reads every poll row from PostgreSQL and converts each row into a domain.Poll.
// It also loads each poll's options and votes so clients can see the full poll state.
func (store *Store) GetAllPolls() ([]domain.Poll, error) {
	// Query returns rows, which must be scanned one at a time.
	rows, err := store.DB.Query(
		"SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE(poll_type, 'movie') AS poll_type FROM polls",
	)

	if err != nil {
		rows, err = store.DB.Query(
			"SELECT id, COALESCE(poll_code, '') AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline FROM polls",
		)
		if err != nil {
			return nil, err
		}

		return store.scanPollRows(rows, false)
	}

	return store.scanPollRows(rows, true)
}

func (store *Store) scanPollRows(rows *sql.Rows, hasPollType bool) ([]domain.Poll, error) {
	defer rows.Close()

	polls := make([]domain.Poll, 0)

	// rows.Next moves through the result set one database row at a time.
	for rows.Next() {
		var currentPoll domain.Poll

		var err error

		// Scan copies the current row's columns into the poll struct fields.
		if hasPollType {
			err = rows.Scan(
				&currentPoll.ID,
				&currentPoll.PollCode,
				&currentPoll.Name,
				&currentPoll.IsClosed,
				&currentPoll.IsVotingActive,
				&currentPoll.MaxVotesPerPerson,
				&currentPoll.Deadline,
				&currentPoll.PollType,
			)
		} else {
			err = rows.Scan(
				&currentPoll.ID,
				&currentPoll.PollCode,
				&currentPoll.Name,
				&currentPoll.IsClosed,
				&currentPoll.IsVotingActive,
				&currentPoll.MaxVotesPerPerson,
				&currentPoll.Deadline,
			)
			currentPoll.PollType = "movie"
		}

		if err != nil {
			return nil, err
		}

		// Load the options connected to this poll before adding it to the response list.
		options, err := store.GetOptionsByPollID(currentPoll.ID)
		if err != nil {
			return nil, err
		}

		currentPoll.Options = options
		currentPoll.Movies = options

		// Load the votes connected to this poll, including the selected movie IDs.
		votes, err := store.GetVotesByPollID(currentPoll.ID)
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

// GetAllOptions reads every movie row from PostgreSQL and converts each row into a domain.Option.
func (store *Store) GetAllOptions() ([]domain.Option, error) {
	// Query returns rows, which must be scanned one at a time.
	rows, err := store.DB.Query(
		"SELECT id, poll_id, title, description, COALESCE(image_url, '') AS image_url, release_year FROM options",
	)

	if err != nil {
		rows, err = store.DB.Query(
			"SELECT id, poll_id, title, release_year, description FROM movies",
		)
		if err != nil {
			return nil, err
		}

		return store.scanOptionRows(rows, false)
	}

	return store.scanOptionRows(rows, true)
}

// GetOptionsByPollID reads only the options that belong to one poll.
// Poll listing uses this to include each poll's movie options in the response.
func (store *Store) GetOptionsByPollID(pollID string) ([]domain.Option, error) {
	// The WHERE clause filters the options table down to the requested poll ID.
	rows, err := store.DB.Query(
		"SELECT id, poll_id, title, description, COALESCE(image_url, '') AS image_url, release_year FROM options WHERE poll_id = $1",
		pollID,
	)

	if err != nil {
		rows, err = store.DB.Query(
			"SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id = $1",
			pollID,
		)
		if err != nil {
			return nil, err
		}

		return store.scanOptionRows(rows, false)
	}

	return store.scanOptionRows(rows, true)
}

func (store *Store) scanOptionRows(rows *sql.Rows, hasPosterURL bool) ([]domain.Option, error) {
	defer rows.Close()

	options := make([]domain.Option, 0)

	// Build one movie struct for each returned database row.
	for rows.Next() {
		var currentOption domain.Option
		var err error

		if hasPosterURL {
			err = rows.Scan(
				&currentOption.ID,
				&currentOption.PollID,
				&currentOption.Title,
				&currentOption.Description,
				&currentOption.ImageURL,
				&currentOption.ReleaseYear,
			)
		} else {
			err = rows.Scan(
				&currentOption.ID,
				&currentOption.PollID,
				&currentOption.Title,
				&currentOption.ReleaseYear,
				&currentOption.Description,
			)
		}

		if err != nil {
			return nil, err
		}

		if currentOption.ImageURL == "" {
			currentOption.ImageURL = currentOption.PosterURL
		}
		if currentOption.PosterURL == "" {
			currentOption.PosterURL = currentOption.ImageURL
		}

		options = append(options, currentOption)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return options, nil
}

// GetVotesByPollID reads all votes submitted for one poll.
// It also loads each vote's selected movie IDs from the vote_options table.
func (store *Store) GetVotesByPollID(pollID string) ([]domain.Vote, error) {
	// First load the vote rows for this poll.
	rows, err := store.DB.Query(
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

		// The selected options live in the vote_options join table.
		optionIDs, err := store.GetOptionIDsByVoteID(currentVote.ID)
		if err != nil {
			return nil, err
		}

		currentVote.OptionIDs = optionIDs
		currentVote.MovieIDs = optionIDs
		votes = append(votes, currentVote)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return votes, nil
}

// GetAllUsers reads every user row from PostgreSQL and converts each row into a domain.User.
func (store *Store) GetAllUsers() ([]domain.User, error) {

	rows, err := store.DB.Query(
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

// GetMovieIDsByVoteID keeps older callers working while vote_options becomes the main table.
func (store *Store) GetMovieIDsByVoteID(voteID string) ([]string, error) {
	return store.GetOptionIDsByVoteID(voteID)
}

// GetMoviesByPollID keeps older callers working while options become the main model.
func (store *Store) GetMoviesByPollID(pollID string) ([]domain.Movie, error) {
	return store.GetOptionsByPollID(pollID)
}
