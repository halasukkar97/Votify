package api

import (
	"database/sql"
	"errors"
	"testing"
	"time"
	"votify/database"
	"votify/movie"
	"votify/poll"
	"votify/user"
	"votify/vote"

	"github.com/DATA-DOG/go-sqlmock"
)

// newMockDatabase gives API tests a fake database connection.
// The application code still uses database.DB, but sqlmock controls every query result.
func newMockDatabase(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	previousDB := database.DB
	database.DB = db

	t.Cleanup(func() {
		db.Close()
		database.DB = previousDB
	})

	return db, mock
}

// requireExpectations makes sure the code sent exactly the SQL the test expected.
func requireExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet database expectations: %v", err)
	}
}

func TestSavePollWritesPollToDatabase(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)
	p := poll.Poll{
		ID:                "poll-1",
		Name:              "Movie Night",
		IsClosed:          false,
		MaxVotesPerPerson: 2,
		Deadline:          deadline,
	}

	mock.ExpectExec("INSERT INTO polls").
		WithArgs(p.ID, p.Name, p.IsClosed, p.MaxVotesPerPerson, p.Deadline).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := SavePoll(p); err != nil {
		t.Fatalf("expected SavePoll to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveMovieWritesMovieToDatabase(t *testing.T) {
	_, mock := newMockDatabase(t)
	m := movie.Movie{
		ID:          "movie-1",
		PollID:      "poll-1",
		Title:       "Dune",
		ReleaseYear: 2021,
		Description: "Desert politics",
	}

	mock.ExpectExec("INSERT INTO movies").
		WithArgs(m.ID, m.PollID, m.Title, m.ReleaseYear, m.Description).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := SaveMovie(m); err != nil {
		t.Fatalf("expected SaveMovie to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveUserWritesUserToDatabase(t *testing.T) {
	_, mock := newMockDatabase(t)
	u := user.User{ID: "user-1", Name: "Hela"}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := SaveUser(u); err != nil {
		t.Fatalf("expected SaveUser to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveVoteCommitsVoteAndSelectedMovies(t *testing.T) {
	_, mock := newMockDatabase(t)
	v := vote.Vote{
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

	if err := SaveVote(v); err != nil {
		t.Fatalf("expected SaveVote to succeed, got %v", err)
	}

	requireExpectations(t, mock)
}

func TestSaveVoteRollsBackWhenSelectedMovieInsertFails(t *testing.T) {
	_, mock := newMockDatabase(t)
	v := vote.Vote{
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

	if err := SaveVote(v); err == nil {
		t.Fatal("expected SaveVote to return the join insert error")
	}

	requireExpectations(t, mock)
}

func TestPollExistsReadsBooleanFromDatabase(t *testing.T) {
	_, mock := newMockDatabase(t)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := PollExists("poll-1")
	if err != nil {
		t.Fatalf("expected PollExists to succeed, got %v", err)
	}

	if !exists {
		t.Fatal("expected poll to exist")
	}

	requireExpectations(t, mock)
}

func TestGetMovieIDsByVoteIDReadsJoinRows(t *testing.T) {
	_, mock := newMockDatabase(t)

	mock.ExpectQuery("SELECT movie_id FROM vote_movies").
		WithArgs("vote-1").
		WillReturnRows(sqlmock.NewRows([]string{"movie_id"}).
			AddRow("movie-1").
			AddRow("movie-2"))

	movieIDs, err := GetMovieIDsByVoteID("vote-1")
	if err != nil {
		t.Fatalf("expected GetMovieIDsByVoteID to succeed, got %v", err)
	}

	if len(movieIDs) != 2 || movieIDs[0] != "movie-1" || movieIDs[1] != "movie-2" {
		t.Fatalf("expected movie IDs [movie-1 movie-2], got %v", movieIDs)
	}

	requireExpectations(t, mock)
}
