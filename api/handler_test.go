package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"votify/movie"

	"github.com/DATA-DOG/go-sqlmock"
)

// expectEmptyRelations tells sqlmock that a poll has no movies and no votes.
// Several handlers call FindPollByID, which loads these related rows.
func expectEmptyRelations(mock sqlmock.Sqlmock, pollID string) {
	mock.ExpectQuery("SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id").
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description"}))
	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}))
}

// expectPollLookup prepares the SQL rows needed by FindPollByID.
func expectPollLookup(mock sqlmock.Sqlmock, pollID string, deadline time.Time) {
	mock.ExpectQuery("SELECT id, name, is_closed, max_votes_per_person, deadline").
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_closed", "max_votes_per_person", "deadline"}).
			AddRow(pollID, "Movie Night", false, 2, deadline))
}

func TestCreatePollHandlerCreatesPoll(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour).UTC()
	body := bytes.NewBufferString(`{"name":"Movie Night","maxVotesPerPerson":2,"deadline":"` + deadline.Format(time.RFC3339Nano) + `"}`)

	mock.ExpectExec("INSERT INTO polls").
		WithArgs(sqlmock.AnyArg(), "Movie Night", false, 2, deadline).
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(http.MethodPost, "/polls", body)
	response := httptest.NewRecorder()

	CreatePollHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %q", response.Code, response.Body.String())
	}

	var created CreatePollResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode poll response: %v", err)
	}

	if created.ID == "" || created.Name != "Movie Night" {
		t.Fatalf("unexpected created poll response: %+v", created)
	}

	requireExpectations(t, mock)
}

func TestPollsHandlerRejectsUnsupportedMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodDelete, "/polls", nil)
	response := httptest.NewRecorder()

	PollsHandler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
}

func TestListPollsHandlerReturnsPollsWithMoviesAndVotes(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	mock.ExpectQuery("SELECT id, name, is_closed, max_votes_per_person, deadline FROM polls").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "is_closed", "max_votes_per_person", "deadline"}).
			AddRow("poll-1", "Movie Night", false, 2, deadline))
	mock.ExpectQuery("SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description"}).
			AddRow("movie-1", "poll-1", "Dune", 2021, "Desert politics"))
	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}).
			AddRow("vote-1", "poll-1", "user-1"))
	mock.ExpectQuery("SELECT movie_id FROM vote_movies").
		WithArgs("vote-1").
		WillReturnRows(sqlmock.NewRows([]string{"movie_id"}).AddRow("movie-1"))

	request := httptest.NewRequest(http.MethodGet, "/polls", nil)
	response := httptest.NewRecorder()

	ListPollsHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	var polls []struct {
		ID     string        `json:"ID"`
		Movies []movie.Movie `json:"Movies"`
	}
	if err := json.NewDecoder(response.Body).Decode(&polls); err != nil {
		t.Fatalf("failed to decode polls response: %v", err)
	}

	if len(polls) != 1 || polls[0].ID != "poll-1" || len(polls[0].Movies) != 1 {
		t.Fatalf("expected one hydrated poll, got %+v", polls)
	}

	requireExpectations(t, mock)
}

func TestPollByIDHandlerReturnsOnePoll(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookup(mock, "poll-1", deadline)
	expectEmptyRelations(mock, "poll-1")

	request := httptest.NewRequest(http.MethodGet, "/polls/poll-1", nil)
	response := httptest.NewRecorder()

	PollByIDHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestResultsHandlerReturnsVoteTotals(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookup(mock, "poll-1", deadline)
	mock.ExpectQuery("SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description"}).
			AddRow("movie-1", "poll-1", "Dune", 2021, "Desert politics"))
	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}).
			AddRow("vote-1", "poll-1", "user-1"))
	mock.ExpectQuery("SELECT movie_id FROM vote_movies").
		WithArgs("vote-1").
		WillReturnRows(sqlmock.NewRows([]string{"movie_id"}).AddRow("movie-1"))

	request := httptest.NewRequest(http.MethodGet, "/results?pollId=poll-1", nil)
	response := httptest.NewRecorder()

	ResultsHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	var results map[string]int
	if err := json.NewDecoder(response.Body).Decode(&results); err != nil {
		t.Fatalf("failed to decode results response: %v", err)
	}

	if results["movie-1"] != 1 {
		t.Fatalf("expected movie-1 to have 1 vote, got %d", results["movie-1"])
	}

	requireExpectations(t, mock)
}

func TestCreateUserHandlerCreatesUser(t *testing.T) {
	_, mock := newMockDatabase(t)
	mock.ExpectExec("INSERT INTO users").
		WithArgs(sqlmock.AnyArg(), "Hela").
		WillReturnResult(sqlmock.NewResult(1, 1))

	request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{"name":"Hela"}`))
	response := httptest.NewRecorder()

	CreateUserHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestListUsersHandlerReturnsUsers(t *testing.T) {
	_, mock := newMockDatabase(t)
	mock.ExpectQuery("SELECT id, name FROM users").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("user-1", "Hela"))

	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	response := httptest.NewRecorder()

	ListUsersHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestUsersHandlerRejectsUnsupportedMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/users", nil)
	response := httptest.NewRecorder()

	UsersHandler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
}

func TestCreateMovieHandlerCreatesMovieWhenPollExists(t *testing.T) {
	_, mock := newMockDatabase(t)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectExec("INSERT INTO movies").
		WithArgs(sqlmock.AnyArg(), "poll-1", "Dune", 2021, "Desert politics").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"title":"Dune","pollId":"poll-1","releaseYear":2021,"description":"Desert politics"}`
	request := httptest.NewRequest(http.MethodPost, "/movies", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	CreateMovieHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestListMoviesHandlerReturnsMovies(t *testing.T) {
	_, mock := newMockDatabase(t)
	mock.ExpectQuery("SELECT id, poll_id, title, release_year, description FROM movies$").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description"}).
			AddRow("movie-1", "poll-1", "Dune", 2021, "Desert politics"))

	request := httptest.NewRequest(http.MethodGet, "/movies", nil)
	response := httptest.NewRecorder()

	ListMoviesHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestMoviesHandlerRejectsUnsupportedMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/movies", nil)
	response := httptest.NewRecorder()

	MoviesHandler(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
}

func TestCreateVoteHandlerCreatesVote(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookup(mock, "poll-1", deadline)
	mock.ExpectQuery("SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description"}).
			AddRow("movie-1", "poll-1", "Dune", 2021, "Desert politics"))
	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO votes").
		WithArgs(sqlmock.AnyArg(), "poll-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO vote_movies").
		WithArgs(sqlmock.AnyArg(), "movie-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := `{"pollId":"poll-1","userId":"user-1","movieIds":["movie-1"]}`
	request := httptest.NewRequest(http.MethodPost, "/votes", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	CreateVoteHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}
