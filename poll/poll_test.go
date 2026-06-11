package poll

import (
	"testing"
	"time"
	"votify/movie"
	"votify/vote"
)

// newTestPoll builds a poll with only the fields each test cares about.
// Test helpers keep each test short and focused on one behavior.
func newTestPoll(maxVotes int, deadline time.Time) Poll {
	return CreateNewPoll(CreatePollInput{
		Name:              "Friday Movie Night",
		MaxVotesPerPerson: maxVotes,
		Deadline:          deadline,
	})
}

// newTestMovie creates a movie that belongs to a specific poll.
func newTestMovie(pollID, title string) movie.Movie {
	return movie.CreateNewMovie(movie.CreateMovieInput{
		Title:       title,
		PollID:      pollID,
		ReleaseYear: 2021,
		Description: title + " description",
	})
}

// newTestVote creates a vote with the selected movie IDs already filled in.
func newTestVote(pollID, userID string, movieIDs []string) vote.Vote {
	return vote.CreateNewVote(vote.CreateVoteInput{
		PollID:   pollID,
		UserID:   userID,
		MovieIDs: movieIDs,
	})
}

// requireError checks both that an error happened and that it has the expected message.
func requireError(t *testing.T, err error, expected string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error %q, got nil", expected)
	}

	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestCreateNewPollSetsDefaults(t *testing.T) {
	deadline := time.Now().Add(24 * time.Hour)
	p := newTestPoll(2, deadline)

	if p.ID == "" {
		t.Fatal("expected poll ID to be generated")
	}

	if p.Name != "Friday Movie Night" {
		t.Errorf("expected poll name to be copied, got %q", p.Name)
	}

	if p.MaxVotesPerPerson != 2 {
		t.Errorf("expected max votes 2, got %d", p.MaxVotesPerPerson)
	}

	if !p.Deadline.Equal(deadline) {
		t.Errorf("expected deadline %v, got %v", deadline, p.Deadline)
	}

	if p.IsClosed {
		t.Error("expected new poll to start open")
	}

	if len(p.Movies) != 0 || len(p.Votes) != 0 {
		t.Fatalf("expected new poll to start with empty movies and votes")
	}
}

func TestPollMutationHelpers(t *testing.T) {
	p := newTestPoll(2, time.Now().Add(24*time.Hour))
	m := newTestMovie(p.ID, "Arrival")
	v := newTestVote(p.ID, "user-1", []string{m.ID})

	p.AddMovie(m)
	if len(p.Movies) != 1 || p.Movies[0].ID != m.ID {
		t.Fatalf("expected AddMovie to append the movie")
	}

	p.AddVote(v)
	if len(p.Votes) != 1 || p.Votes[0].ID != v.ID {
		t.Fatalf("expected AddVote to append the vote")
	}

	p.Close()
	if !p.IsClosed {
		t.Fatal("expected Close to mark the poll closed")
	}
}

func TestPollSmallPredicateHelpers(t *testing.T) {
	p := newTestPoll(2, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")
	p.AddMovie(movie1)

	if !p.ValidateVoteCount([]string{movie1.ID, movie2.ID}) {
		t.Error("expected two movies to be allowed")
	}

	if p.ValidateVoteCount([]string{movie1.ID, movie2.ID, "movie-3"}) {
		t.Error("expected three movies to be too many")
	}

	if !p.HasMovie(movie1.ID) {
		t.Error("expected poll to contain movie1")
	}

	if p.HasMovie(movie2.ID) {
		t.Error("did not expect poll to contain movie2 before adding it")
	}

	duplicateVote := newTestVote(p.ID, "user-1", []string{movie1.ID, movie1.ID})
	if !p.HasDuplicateMovies(duplicateVote) {
		t.Error("expected duplicate movie IDs to be detected")
	}

	uniqueVote := newTestVote(p.ID, "user-1", []string{movie1.ID})
	if p.HasDuplicateMovies(uniqueVote) {
		t.Error("did not expect unique movie IDs to count as duplicates")
	}
}

func TestAlreadyVotedAndIsExpired(t *testing.T) {
	expiredPoll := newTestPoll(2, time.Now().Add(-time.Hour))
	if !expiredPoll.IsExpired() {
		t.Error("expected past deadline to be expired")
	}

	openPoll := newTestPoll(2, time.Now().Add(time.Hour))
	m := newTestMovie(openPoll.ID, "Dune")
	openPoll.AddMovie(m)
	openPoll.AddVote(newTestVote(openPoll.ID, "user-1", []string{m.ID}))

	if !openPoll.AlreadyVoted("user-1") {
		t.Error("expected user-1 to be marked as already voted")
	}

	if openPoll.AlreadyVoted("user-2") {
		t.Error("did not expect user-2 to be marked as already voted")
	}
}

func TestGetResultsCountsRepeatedMovieIDsAcrossVotes(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")

	p.AddVote(newTestVote(p.ID, "user-1", []string{movie1.ID, movie2.ID}))
	p.AddVote(newTestVote(p.ID, "user-2", []string{movie1.ID}))

	results := p.GetResults()
	if results[movie1.ID] != 2 {
		t.Errorf("expected movie1 to have 2 votes, got %d", results[movie1.ID])
	}

	if results[movie2.ID] != 1 {
		t.Errorf("expected movie2 to have 1 vote, got %d", results[movie2.ID])
	}
}

func TestSubmitVoteSuccess(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")

	p.AddMovie(movie1)
	p.AddMovie(movie2)

	v := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie2.ID})

	err := p.SubmitVote(v)
	if err != nil {
		t.Fatalf("expected vote to succeed, got error: %v", err)
	}

	results := p.GetResults()
	if results[movie1.ID] != 1 {
		t.Errorf("expected Interstellar to have 1 vote, got %d", results[movie1.ID])
	}

	if results[movie2.ID] != 1 {
		t.Errorf("expected Dune to have 1 vote, got %d", results[movie2.ID])
	}
}

func TestSubmitVotePollExpired(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(-24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")

	p.AddMovie(movie1)
	p.AddMovie(movie2)

	v := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie2.ID})

	err := p.SubmitVote(v)

	requireError(t, err, "poll has expired")
}

func TestSubmitVotePollClosed(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")

	p.Close()
	p.AddMovie(movie1)
	p.AddMovie(movie2)

	v := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie2.ID})

	err := p.SubmitVote(v)

	requireError(t, err, "poll is closed")
}

func TestSubmitVoteAlreadyVoted(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")
	movie3 := newTestMovie(p.ID, "Titanic")

	p.AddMovie(movie1)
	p.AddMovie(movie2)
	p.AddMovie(movie3)

	firstVote := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie2.ID, movie3.ID})
	secondVote := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie2.ID, movie3.ID})

	if err := p.SubmitVote(firstVote); err != nil {
		t.Fatalf("expected first vote to succeed, got error: %v", err)
	}

	err := p.SubmitVote(secondVote)

	requireError(t, err, "you have already voted for this poll")
}

func TestSubmitVoteMovieDoesNotExist(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")

	p.AddMovie(movie1)

	v := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie2.ID})

	err := p.SubmitVote(v)

	requireError(t, err, "this movie doesn't exist in this poll")
}

func TestSubmitVoteDuplicateMovie(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")

	p.AddMovie(movie1)

	v := newTestVote(p.ID, "hela-user", []string{movie1.ID, movie1.ID})

	err := p.SubmitVote(v)

	requireError(t, err, "duplicated votes for the same movie are not allowed")
}

func TestSubmitVoteTooManyMovies(t *testing.T) {
	p := newTestPoll(3, time.Now().Add(24*time.Hour))
	movie1 := newTestMovie(p.ID, "Interstellar")
	movie2 := newTestMovie(p.ID, "Dune")
	movie3 := newTestMovie(p.ID, "Titanic")
	movie4 := newTestMovie(p.ID, "Arrival")

	p.AddMovie(movie1)
	p.AddMovie(movie2)
	p.AddMovie(movie3)
	p.AddMovie(movie4)

	v := newTestVote(p.ID, "hela-user", []string{
		movie1.ID,
		movie2.ID,
		movie3.ID,
		movie4.ID,
	})

	err := p.SubmitVote(v)

	requireError(t, err, "too many movies selected")
}
