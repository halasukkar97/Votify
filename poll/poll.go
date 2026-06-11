package poll

import (
	"errors"
	"time"
	"votify/movie"
	"votify/vote"

	"github.com/google/uuid"
)

// Poll represents a movie voting poll.
// It stores the movies people can vote on and the votes that have been submitted.
type Poll struct {
	ID                string
	Name              string
	IsClosed          bool
	MaxVotesPerPerson int
	Deadline          time.Time
	Movies            []movie.Movie
	Votes             []vote.Vote
}

// CreatePollInput contains the fields needed to create a new poll.
// Keeping input in its own struct makes CreateNewPoll easier to call and test.
type CreatePollInput struct {
	Name              string
	MaxVotesPerPerson int
	Deadline          time.Time
}

// CreateNewPoll creates a new poll.
// It generates a unique ID and starts with empty movie and vote lists.
func CreateNewPoll(input CreatePollInput) Poll {
	return Poll{
		ID:                uuid.New().String(),
		Name:              input.Name,
		MaxVotesPerPerson: input.MaxVotesPerPerson,
		Deadline:          input.Deadline,
		Movies:            []movie.Movie{},
		Votes:             []vote.Vote{},
	}
}

// AddMovie adds a movie to the poll.
// The pointer receiver (*Poll) means this method changes the existing poll.
func (p *Poll) AddMovie(movie movie.Movie) {
	p.Movies = append(p.Movies, movie)
}

// AddVote adds a vote to the poll.
// This is called only after SubmitVote has checked the rules.
func (p *Poll) AddVote(vote vote.Vote) {
	p.Votes = append(p.Votes, vote)
}

// Close marks the poll as closed.
func (p *Poll) Close() {
	p.IsClosed = true
}

// ValidateVoteCount checks if the vote count is allowed.
// It returns true when the user selected no more than MaxVotesPerPerson movies.
func (p *Poll) ValidateVoteCount(selectedMovieIDs []string) bool {
	return len(selectedMovieIDs) <= p.MaxVotesPerPerson
}

// GetResults returns the vote count per movie.
// The map key is a movie ID and the value is the number of votes for that movie.
func (p *Poll) GetResults() map[string]int {
	result := make(map[string]int)

	for _, v := range p.Votes {
		for _, movieID := range v.MovieIDs {
			result[movieID]++
		}
	}

	return result
}

// HasMovie checks if a movie belongs to the poll.
// This prevents users from voting for movies that are not part of this poll.
func (p *Poll) HasMovie(movieID string) bool {
	for _, film := range p.Movies {
		if film.ID == movieID {
			return true
		}
	}

	return false
}

// HasDuplicateMovies checks for duplicate movie votes.
// It returns true if the same movie ID appears more than once in one vote.
func (p *Poll) HasDuplicateMovies(v vote.Vote) bool {
	seenMovies := make(map[string]bool)

	for _, movieID := range v.MovieIDs {
		if seenMovies[movieID] {
			return true
		}

		seenMovies[movieID] = true
	}

	return false
}

// AlreadyVoted checks if a user has already voted.
// Each user can submit only one vote per poll.
func (p *Poll) AlreadyVoted(voterID string) bool {
	for _, voteEntry := range p.Votes {
		if voteEntry.UserID == voterID {
			return true
		}
	}

	return false
}

// IsExpired checks if the deadline has passed.
// time.Now() is the current server time.
func (p *Poll) IsExpired() bool {
	return time.Now().After(p.Deadline)
}

// SubmitVote validates and stores a vote.
// It returns an error when a rule fails, so the API can explain the problem.
func (p *Poll) SubmitVote(v vote.Vote) error {

	// Stop immediately if the poll is no longer accepting votes.
	if p.IsClosed {
		return errors.New("poll is closed")
	}

	// Do not accept votes after the deadline.
	if p.IsExpired() {
		return errors.New("poll has expired")
	}

	// Do not allow the same user to vote twice in the same poll.
	if p.AlreadyVoted(v.UserID) {
		return errors.New("you have already voted for this poll")
	}

	// Enforce the poll's maximum number of selected movies.
	if !p.ValidateVoteCount(v.MovieIDs) {
		return errors.New("too many movies selected")
	}

	// Prevent one vote from counting the same movie more than once.
	if p.HasDuplicateMovies(v) {
		return errors.New("duplicated votes for the same movie are not allowed")
	}

	// Make sure every selected movie actually belongs to this poll.
	for _, movieID := range v.MovieIDs {
		if !p.HasMovie(movieID) {
			return errors.New("this movie doesn't exist in this poll")
		}
	}

	// All rules passed, so the vote can be stored on the poll.
	p.AddVote(v)

	return nil
}
