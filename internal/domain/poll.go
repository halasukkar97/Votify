package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Poll represents a generic voting poll.
// ID is the internal UUID used by the database.
// PollCode is the short public code users can share with friends.
type Poll struct {
	ID                string    `json:"id"`
	PollCode          string    `json:"pollCode"`
	Name              string    `json:"name"`
	IsClosed          bool      `json:"isClosed"`
	IsVotingActive    bool      `json:"isVotingActive"`
	MaxVotesPerPerson int       `json:"maxVotesPerPerson"`
	Deadline          time.Time `json:"deadline"`
	PollType          string    `json:"pollType"`
	Options           []Option  `json:"options"`
	Movies            []Option  `json:"movies,omitempty"`
	Votes             []Vote    `json:"votes"`
}

// CreatePollInput contains the fields needed to create a new poll.
type CreatePollInput struct {
	PollCode          string    `json:"pollCode"`
	Name              string    `json:"name"`
	MaxVotesPerPerson int       `json:"maxVotesPerPerson"`
	Deadline          time.Time `json:"deadline"`
	PollType          string    `json:"pollType"`
}

// CreateNewPoll creates a new poll.
// The UUID stays internal, while PollCode is the short public join code.
func CreateNewPoll(input CreatePollInput) Poll {
	return Poll{
		ID:                uuid.New().String(),
		PollCode:          input.PollCode,
		Name:              input.Name,
		IsVotingActive:    false,
		MaxVotesPerPerson: input.MaxVotesPerPerson,
		Deadline:          input.Deadline,
		PollType:          normalizePollType(input.PollType),
		Options:           []Option{},
		Movies:            []Option{},
		Votes:             []Vote{},
	}
}

// AddOption adds an option to the poll.
// The pointer receiver (*Poll) means this method changes the existing poll.
func (p *Poll) AddOption(option Option) {
	p.Options = append(p.Options, option)
	p.Movies = p.Options
}

// AddMovie keeps older tests and callers working while options become the main model.
func (p *Poll) AddMovie(option Option) {
	p.AddOption(option)
}

// AddVote adds a vote to the poll.
// This is called only after SubmitVote has checked the rules.
func (p *Poll) AddVote(vote Vote) {
	p.Votes = append(p.Votes, vote)
}

// Close marks the poll as closed.
func (p *Poll) Close() {
	p.IsClosed = true
}

// ValidateVoteCount checks if the user selected no more than allowed.
func (p *Poll) ValidateVoteCount(selectedOptionIDs []string) bool {
	return len(selectedOptionIDs) <= p.MaxVotesPerPerson
}

// GetResults returns the vote count per option.
// The map key is an option ID and the value is the number of votes for that option.
func (p *Poll) GetResults() map[string]int {
	result := make(map[string]int)

	for _, v := range p.Votes {
		for _, optionID := range v.OptionIDs {
			result[optionID]++
		}
	}

	return result
}

// HasOption checks if an option belongs to the poll.
// This prevents users from voting for options that are not part of this poll.
func (p *Poll) HasOption(optionID string) bool {
	for _, option := range p.Options {
		if option.ID == optionID {
			return true
		}
	}

	return false
}

// HasMovie keeps older tests and callers working while options become the main model.
func (p *Poll) HasMovie(optionID string) bool {
	return p.HasOption(optionID)
}

// HasDuplicateOptions checks if the same option was selected twice in one vote.
func (p *Poll) HasDuplicateOptions(v Vote) bool {
	seenOptions := make(map[string]bool)

	for _, optionID := range v.OptionIDs {
		if seenOptions[optionID] {
			return true
		}

		seenOptions[optionID] = true
	}

	return false
}

// HasDuplicateMovies keeps older tests and callers working while options become the main model.
func (p *Poll) HasDuplicateMovies(v Vote) bool {
	return p.HasDuplicateOptions(v)
}

// AlreadyVoted checks if a user has already voted in this poll.
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
func (p *Poll) SubmitVote(v Vote) error {
	// Stop immediately if the poll is no longer accepting votes.
	if p.IsClosed {
		return errors.New("poll is closed")
	}

	// Do not accept votes after the deadline.
	if p.IsExpired() {
		return errors.New("poll has expired")
	}

	// Voting only opens after the creator activates the poll.
	if !p.IsVotingActive {
		return errors.New("voting has not started yet")
	}

	// Do not allow the same user to vote twice in the same poll.
	if p.AlreadyVoted(v.UserID) {
		return errors.New("you have already voted for this poll")
	}

	// Enforce the poll's maximum number of selected options.
	if !p.ValidateVoteCount(v.OptionIDs) {
		return errors.New("too many options selected")
	}

	// Prevent one vote from counting the same option more than once.
	if p.HasDuplicateOptions(v) {
		return errors.New("duplicated votes for the same option are not allowed")
	}

	// Make sure every selected option actually belongs to this poll.
	for _, optionID := range v.OptionIDs {
		if !p.HasOption(optionID) {
			return errors.New("this option does not exist in this poll")
		}
	}

	// All rules passed, so the vote can be stored on the poll.
	p.AddVote(v)

	return nil
}

func normalizePollType(pollType string) string {
	if pollType == "" {
		return "movie"
	}

	return pollType
}
