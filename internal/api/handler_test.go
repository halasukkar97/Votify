package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"votify/internal/database"
	"votify/internal/domain"
	"votify/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
)

// expectEmptyRelations tells sqlmock that a poll has no options and no votes.
// Several handlers call FindPollByID, which loads these related rows.
func expectEmptyRelations(mock sqlmock.Sqlmock, pollID string) {
	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options WHERE poll_id`).
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "description", "image_url", "release_year"}))

	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs(pollID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}))
}

// expectPollLookupByID prepares the SQL rows needed by FindPollByID.
func expectPollLookupByID(mock sqlmock.Sqlmock, pollID string, deadline time.Time) {
	expectPollLookupByIDWithVoting(mock, pollID, deadline, true)
}

func expectPollLookupByIDWithVoting(mock sqlmock.Sqlmock, pollID string, deadline time.Time, isVotingActive bool) {
	mock.ExpectQuery(`SELECT id, COALESCE\(poll_code, ''\) AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE\(poll_type, 'movie'\) AS poll_type\s+FROM polls\s+WHERE id = \$1`).
		WithArgs(pollID).
		WillReturnRows(newPollRows().AddRow(
			pollID,
			"12345678",
			"Movie Night",
			false,
			isVotingActive,
			2,
			deadline,
			"movie",
		))
}

// expectPollLookupByCode prepares the SQL rows needed by FindPollByCode.
func expectPollLookupByCode(mock sqlmock.Sqlmock, pollCode string, pollID string, deadline time.Time) {
	expectPollLookupByCodeWithVoting(mock, pollCode, pollID, deadline, true)
}

func expectPollLookupByCodeWithVoting(mock sqlmock.Sqlmock, pollCode string, pollID string, deadline time.Time, isVotingActive bool) {
	mock.ExpectQuery(`SELECT id, COALESCE\(poll_code, ''\) AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE\(poll_type, 'movie'\) AS poll_type\s+FROM polls\s+WHERE poll_code = \$1`).
		WithArgs(pollCode).
		WillReturnRows(newPollRows().AddRow(
			pollID,
			pollCode,
			"Movie Night",
			false,
			isVotingActive,
			2,
			deadline,
			"movie",
		))
}

func expectPollCodeMiss(mock sqlmock.Sqlmock, pollCode string) {
	mock.ExpectQuery(`SELECT id, COALESCE\(poll_code, ''\) AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE\(poll_type, 'movie'\) AS poll_type\s+FROM polls\s+WHERE poll_code = \$1`).
		WithArgs(pollCode).
		WillReturnError(sql.ErrNoRows)
}

func newPollRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"poll_code",
		"name",
		"is_closed",
		"is_voting_active",
		"max_votes_per_person",
		"deadline",
		"poll_type",
	})
}

func TestCreatePollHandlerCreatesPoll(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour).UTC()
	body := bytes.NewBufferString(`{"name":"Movie Night","maxVotesPerPerson":2,"deadline":"` + deadline.Format(time.RFC3339Nano) + `"}`)

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM polls WHERE poll_code = \$1\)`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectExec("INSERT INTO polls").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Movie Night", false, false, 2, deadline, "movie").
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

	if created.ID == "" || created.PollCode == "" || created.Name != "Movie Night" || created.IsVotingActive {
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

	mock.ExpectQuery(`SELECT id, COALESCE\(poll_code, ''\) AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE\(poll_type, 'movie'\) AS poll_type FROM polls`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"poll_code",
			"name",
			"is_closed",
			"is_voting_active",
			"max_votes_per_person",
			"deadline",
			"poll_type",
		}).AddRow("poll-1", "12345678", "Movie Night", false, true, 2, deadline, "movie"))

	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options WHERE poll_id`).
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "description", "image_url", "release_year"}).
			AddRow("option-1", "poll-1", "Dune", "Desert politics", "https://image.test/dune.jpg", 2021))

	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}).
			AddRow("vote-1", "poll-1", "user-1"))

	mock.ExpectQuery("SELECT option_id FROM vote_options").
		WithArgs("vote-1").
		WillReturnRows(sqlmock.NewRows([]string{"option_id"}).AddRow("option-1"))

	request := httptest.NewRequest(http.MethodGet, "/polls", nil)
	response := httptest.NewRecorder()

	ListPollsHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	var polls []struct {
		ID       string         `json:"id"`
		PollCode string         `json:"pollCode"`
		Movies   []domain.Movie `json:"movies"`
	}
	if err := json.NewDecoder(response.Body).Decode(&polls); err != nil {
		t.Fatalf("failed to decode polls response: %v", err)
	}

	if len(polls) != 1 || polls[0].ID != "poll-1" || polls[0].PollCode != "12345678" || len(polls[0].Movies) != 1 {
		t.Fatalf("expected one hydrated poll, got %+v", polls)
	}

	requireExpectations(t, mock)
}

func TestGetAllPollsHandlesOldRowsWithEmptyPollCode(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	mock.ExpectQuery(`SELECT id, COALESCE\(poll_code, ''\) AS poll_code, name, is_closed, is_voting_active, max_votes_per_person, deadline, COALESCE\(poll_type, 'movie'\) AS poll_type FROM polls`).
		WillReturnRows(newPollRows().AddRow("poll-legacy", "", "Old Movie Night", false, false, 2, deadline, "movie"))

	expectEmptyRelations(mock, "poll-legacy")

	polls, err := repository.NewStore(database.DB).GetAllPolls()
	if err != nil {
		t.Fatalf("expected GetAllPolls to succeed, got %v", err)
	}

	if len(polls) != 1 || polls[0].PollCode != "" {
		t.Fatalf("expected one poll with an empty poll code, got %+v", polls)
	}

	requireExpectations(t, mock)
}

func TestActivateVotingHandlerSetsVotingActive(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookupByCodeWithVoting(mock, "12345678", "poll-1", deadline, false)
	expectEmptyRelations(mock, "poll-1")

	mock.ExpectExec("UPDATE polls SET is_voting_active = TRUE WHERE poll_code").
		WithArgs("12345678").
		WillReturnResult(sqlmock.NewResult(0, 1))

	expectPollLookupByCodeWithVoting(mock, "12345678", "poll-1", deadline, true)
	expectEmptyRelations(mock, "poll-1")

	request := httptest.NewRequest(http.MethodPatch, "/polls/12345678/activate-voting", nil)
	response := httptest.NewRecorder()

	PollByIDHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	var updated struct {
		IsVotingActive bool `json:"isVotingActive"`
	}
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode activation response: %v", err)
	}

	if !updated.IsVotingActive {
		t.Fatal("expected voting to be active after activation")
	}

	requireExpectations(t, mock)
}

func TestActivateVotingHandlerRejectsAlreadyActivePoll(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookupByCodeWithVoting(mock, "12345678", "poll-1", deadline, true)
	expectEmptyRelations(mock, "poll-1")

	request := httptest.NewRequest(http.MethodPatch, "/polls/12345678/activate-voting", nil)
	response := httptest.NewRecorder()

	PollByIDHandler(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestPollByIDHandlerReturnsOnePollByPollCode(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookupByCode(mock, "03739172", "poll-1", deadline)
	expectEmptyRelations(mock, "poll-1")

	request := httptest.NewRequest(http.MethodGet, "/polls/03739172", nil)
	response := httptest.NewRecorder()

	PollByIDHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestPollByIDHandlerFallsBackToInternalID(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollCodeMiss(mock, "poll-1")
	expectPollLookupByID(mock, "poll-1", deadline)
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

	expectPollLookupByID(mock, "poll-1", deadline)

	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options WHERE poll_id`).
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "description", "image_url", "release_year"}).
			AddRow("option-1", "poll-1", "Dune", "Desert politics", "https://image.test/dune.jpg", 2021))

	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}).
			AddRow("vote-1", "poll-1", "user-1"))

	mock.ExpectQuery("SELECT option_id FROM vote_options").
		WithArgs("vote-1").
		WillReturnRows(sqlmock.NewRows([]string{"option_id"}).AddRow("option-1"))

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

	if results["option-1"] != 1 {
		t.Fatalf("expected option-1 to have 1 vote, got %d", results["option-1"])
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

func TestUpdateUserHandlerRenamesExistingUser(t *testing.T) {
	_, mock := newMockDatabase(t)

	mock.ExpectQuery("UPDATE users SET name").
		WithArgs("New Hela", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("user-1", "New Hela"))

	request := httptest.NewRequest(http.MethodPatch, "/users/user-1", bytes.NewBufferString(`{"name":"New Hela"}`))
	response := httptest.NewRecorder()

	UpdateUserHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	var updatedUser CreateUserResponse
	if err := json.NewDecoder(response.Body).Decode(&updatedUser); err != nil {
		t.Fatalf("failed to decode user response: %v", err)
	}

	if updatedUser.ID != "user-1" {
		t.Fatalf("expected user ID to stay user-1, got %q", updatedUser.ID)
	}

	if updatedUser.Name != "New Hela" {
		t.Fatalf("expected updated name, got %q", updatedUser.Name)
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

	expectPollLookupByIDWithVoting(mock, "poll-1", time.Now().Add(24*time.Hour), false)
	expectEmptyRelations(mock, "poll-1")

	mock.ExpectExec("INSERT INTO options").
		WithArgs(sqlmock.AnyArg(), "poll-1", "Dune", "Desert politics", "https://image.test/dune.jpg", 2021, "null").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"title":"Dune","pollId":"poll-1","releaseYear":2021,"description":"Desert politics","posterUrl":"https://image.test/dune.jpg"}`
	request := httptest.NewRequest(http.MethodPost, "/movies", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	CreateMovieHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestCreateMovieHandlerRejectsMovieAfterVotingStarts(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookupByIDWithVoting(mock, "poll-1", deadline, true)
	expectEmptyRelations(mock, "poll-1")

	body := `{"title":"Dune","pollId":"poll-1","releaseYear":2021,"description":"Desert politics","posterUrl":"https://image.test/dune.jpg"}`
	request := httptest.NewRequest(http.MethodPost, "/movies", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	CreateMovieHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestListMoviesHandlerReturnsMovies(t *testing.T) {
	_, mock := newMockDatabase(t)

	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "description", "image_url", "release_year"}).
			AddRow("option-1", "poll-1", "Dune", "Desert politics", "https://image.test/dune.jpg", 2021))

	request := httptest.NewRequest(http.MethodGet, "/movies", nil)
	response := httptest.NewRecorder()

	ListMoviesHandler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestGetMoviesByPollIDFallsBackWhenPosterColumnIsMissing(t *testing.T) {
	_, mock := newMockDatabase(t)

	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options WHERE poll_id`).
		WithArgs("poll-1").
		WillReturnError(errors.New("table options does not exist"))

	mock.ExpectQuery("SELECT id, poll_id, title, release_year, description FROM movies WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "release_year", "description"}).
			AddRow("movie-1", "poll-1", "Dune", 2021, "Desert politics"))

	movies, err := repository.NewStore(database.DB).GetMoviesByPollID("poll-1")
	if err != nil {
		t.Fatalf("expected fallback movie query to succeed, got %v", err)
	}

	if len(movies) != 1 || movies[0].PosterURL != "" {
		t.Fatalf("expected one movie with an empty poster URL, got %+v", movies)
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

func TestCreateVoteHandlerRejectsVoteBeforeVotingStarts(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookupByIDWithVoting(mock, "poll-1", deadline, false)

	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options WHERE poll_id`).
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "description", "image_url", "release_year"}).
			AddRow("option-1", "poll-1", "Dune", "Desert politics", "https://image.test/dune.jpg", 2021))

	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}))

	body := `{"pollId":"poll-1","userId":"user-1","movieIds":["movie-1"]}`
	request := httptest.NewRequest(http.MethodPost, "/votes", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	CreateVoteHandler(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}

func TestCreateVoteHandlerCreatesVote(t *testing.T) {
	_, mock := newMockDatabase(t)
	deadline := time.Now().Add(24 * time.Hour)

	expectPollLookupByID(mock, "poll-1", deadline)

	mock.ExpectQuery(`SELECT id, poll_id, title, description, COALESCE\(image_url, ''\) AS image_url, release_year FROM options WHERE poll_id`).
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "title", "description", "image_url", "release_year"}).
			AddRow("option-1", "poll-1", "Dune", "Desert politics", "https://image.test/dune.jpg", 2021))

	mock.ExpectQuery("SELECT id, poll_id, user_id FROM votes WHERE poll_id").
		WithArgs("poll-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "poll_id", "user_id"}))

	mock.ExpectBegin()

	mock.ExpectExec("INSERT INTO votes").
		WithArgs(sqlmock.AnyArg(), "poll-1", "user-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO vote_options").
		WithArgs(sqlmock.AnyArg(), "option-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	body := `{"pollId":"poll-1","userId":"user-1","optionIds":["option-1"]}`
	request := httptest.NewRequest(http.MethodPost, "/votes", bytes.NewBufferString(body))
	response := httptest.NewRecorder()

	CreateVoteHandler(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d with body %q", response.Code, response.Body.String())
	}

	requireExpectations(t, mock)
}
