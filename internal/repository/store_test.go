package repository

import (
	"errors"
	"testing"
	"time"
	"votify/internal/domain"

	"github.com/DATA-DOG/go-sqlmock"
)

// newMockDatabase gives API tests a fake database connection.
// The application code still uses database.DB, but sqlmock controls every query result.
func newMockStore(t *testing.T) (*Store, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return NewStore(db), mock
}

// requireExpectations makes sure the code sent exactly the SQL the test expected.
func requireExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

// expectEmptyRelations tells sqlmock that a poll has no movies and no votes.
func expectEmptyRelations(mock sqlmock.Sqlmock, pollID string) {
	mock.ExpectQuery(`SELECT id, poll_id, title, release_year, description, COALESCE\(poster_url, ''\) AS poster_url FROM movies WHERE poll_id`).
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description", "poster_url"}))

	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}))
}

func TestSavePollWritesPollToDatabase(t *testing.T) {
	store, mock := newMockStore(t)
	deadline := time.Now().Add(24 * time.Hour)
	p := domain.Poll{
		ID:                "poll-1",
		PollCode:          "12345678",
		Name:              "Movie Night",
		IsClosed:          false,
		IsVotingActive:    false,
		MaxVotesPerPerson: 2,
		Deadline:          deadline,
	}

	mock.ExpectExec("INSERT INTO polls").
		WithArgs(p.ID, p.PollCode, p.Name, p.IsClosed, p.IsVotingActive, p.MaxVotesPerPerson, p.Deadline).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.SavePoll(p); err != nil {
		t.Fatalf("expected SavePoll to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestFindPollByCodeUsesPollCodeColumn(t *testing.T) {
	store, mock := newMockStore(t)
	deadline := time.Now().Add(24 * time.Hour)

	mock.ExpectQuery(`SELECT id, COALESCE\(poll_code, ''\) AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline\s+FROM polls\s+WHERE poll_code = \$1`).
		WithArgs("03739172").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"poll_code",
			"name",
			"is_closed",
			"is_voting_active",
			"max_votes_per_person",
			"deadline",
		}).AddRow("poll-1", "03739172", "Movie Night", false, false, 2, deadline))

	expectEmptyRelations(mock, "poll-1")

	foundPoll, found := store.FindPollByCode("03739172")
	if !found {
		t.Fatal("expected FindPollByCode to find the poll")
	}

	if foundPoll.ID != "poll-1" || foundPoll.PollCode != "03739172" {
		t.Fatalf("unexpected poll returned: %+v", foundPoll)
	}

	requireExpectations(t, mock)
}

func TestActivateVotingUpdatesPollPhase(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectExec("UPDATE polls SET is_voting_active = TRUE WHERE poll_code").
		WithArgs("03739172").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.ActivateVoting("03739172"); err != nil {
		t.Fatalf("expected ActivateVoting to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveMovieWritesMovieToDatabase(t *testing.T) {
	store, mock := newMockStore(t)
	m := domain.Movie{
		ID:          "movie-1",
		PollID:      "poll-1",
		Title:       "Dune",
		ReleaseYear: 2021,
		Description: "Desert politics",
		PosterURL:   "https://image.test/dune.jpg",
	}

	mock.ExpectExec("INSERT INTO movies").
		WithArgs(m.ID, m.PollID, m.Title, m.ReleaseYear, m.Description, m.PosterURL).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.SaveMovie(m); err != nil {
		t.Fatalf("expected SaveMovie to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveUserWritesUserToDatabase(t *testing.T) {
	store, mock := newMockStore(t)
	u := domain.User{ID: "user-1", Name: "Hela"}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.SaveUser(u); err != nil {
		t.Fatalf("expected SaveUser to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestUpdateUserNameKeepsUserID(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("UPDATE users SET name").
		WithArgs("New Hela", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("user-1", "New Hela"))

	updatedUser, err := store.UpdateUserName("user-1", "New Hela")
	if err != nil {
		t.Fatalf("expected UpdateUserName to succeed, got %v", err)
	}

	if updatedUser.ID != "user-1" {
		t.Fatalf("expected user ID to stay user-1, got %q", updatedUser.ID)
	}

	if updatedUser.Name != "New Hela" {
		t.Fatalf("expected updated name, got %q", updatedUser.Name)
	}

	requireExpectations(t, mock)
}

func TestSaveVoteCommitsVoteAndSelectedMovies(t *testing.T) {
	store, mock := newMockStore(t)
	v := domain.Vote{
		ID:       "vote-1",
		PollID:   "poll-1",
		UserID:   "user-1",
		MovieIDs: []string{"movie-1", "movie-2"},
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO votes").
		WithArgs(v.ID, v.PollID, v.UserID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO vote_movies").
		WithArgs(v.ID, "movie-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO vote_movies").
		WithArgs(v.ID, "movie-2").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := store.SaveVote(v); err != nil {
		t.Fatalf("expected SaveVote to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveVoteRollsBackWhenSelectedMovieInsertFails(t *testing.T) {
	store, mock := newMockStore(t)
	v := domain.Vote{
		ID:       "vote-1",
		PollID:   "poll-1",
		UserID:   "user-1",
		MovieIDs: []string{"movie-1"},
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO votes").
		WithArgs(v.ID, v.PollID, v.UserID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO vote_movies").
		WithArgs(v.ID, "movie-1").
		WillReturnError(errors.New("join insert failed"))
	mock.ExpectRollback()

	if err := store.SaveVote(v); err == nil {
		t.Fatal("expected SaveVote to return the join insert error")
	}

	requireExpectations(t, mock)
}

func TestPollExistsReadsBooleanFromDatabase(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := store.PollExists("poll-1")
	if err != nil {
		t.Fatalf("expected PollExists to succeed, got %v", err)
	}

	if !exists {
		t.Fatal("expected poll to exist")
	}

	requireExpectations(t, mock)
}

func TestGetMovieIDsByVoteIDReadsJoinRows(t *testing.T) {
	store, mock := newMockStore(t)

	mock.ExpectQuery("SELECT movie_id FROM vote_movies").
		WithArgs("vote-1").
		WillReturnRows(sqlmock.NewRows([]string{"movie_id"}).
			AddRow("movie-1").
			AddRow("movie-2"))

	movieIDs, err := store.GetMovieIDsByVoteID("vote-1")
	if err != nil {
		t.Fatalf("expected GetMovieIDsByVoteID to succeed, got %v", err)
	}

	if len(movieIDs) != 2 || movieIDs[0] != "movie-1" || movieIDs[1] != "movie-2" {
		t.Fatalf("expected movie IDs [movie-1 movie-2], got %v", movieIDs)
	}

	requireExpectations(t, mock)
}
