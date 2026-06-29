// Package service contains application use cases.
// Services coordinate domain rules and repositories without knowing about HTTP.
package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
	"votify/internal/domain"
	"votify/internal/repository"
)

var (
	// ErrNotFound means a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means the request conflicts with the current resource state.
	ErrConflict = errors.New("conflict")
	// ErrInvalidInput means the request is valid JSON but violates app rules.
	ErrInvalidInput = errors.New("invalid input")
)

// Service owns the Movie Vote application use cases.
type Service struct {
	Store *repository.Store
}

// New creates a service layer backed by repositories.
func New(store *repository.Store) *Service {
	return &Service{Store: store}
}

// CreatePoll creates a poll with a unique public poll code.
func (service *Service) CreatePoll(name string, maxVotesPerPerson int, deadline time.Time) (domain.Poll, error) {
	pollCode, err := service.GenerateUniquePollCode()
	if err != nil {
		return domain.Poll{}, err
	}

	createdPoll := domain.CreateNewPoll(domain.CreatePollInput{
		PollCode:          pollCode,
		Name:              name,
		MaxVotesPerPerson: maxVotesPerPerson,
		Deadline:          deadline,
	})

	if err := service.Store.SavePoll(createdPoll); err != nil {
		return domain.Poll{}, err
	}

	return createdPoll, nil
}

// ListPolls returns every poll with related movies and votes.
func (service *Service) ListPolls() ([]domain.Poll, error) {
	return service.Store.GetAllPolls()
}

// GetPoll loads a poll by public poll code first, then internal ID for older callers.
func (service *Service) GetPoll(identifier string) (*domain.Poll, error) {
	foundPoll, found, codeErr := service.Store.FindPollByCodeWithError(identifier)
	if found {
		return foundPoll, nil
	}

	foundPoll, found, idErr := service.Store.FindPollByIDWithError(identifier)
	if found {
		return foundPoll, nil
	}

	if codeErr != nil || idErr != nil {
		return nil, fmt.Errorf("poll lookup failed: pollCodeErr=%v pollIDErr=%v", codeErr, idErr)
	}

	return nil, ErrNotFound
}

// GetResults returns vote totals for a poll.
func (service *Service) GetResults(pollCode string, pollID string) (map[string]int, error) {
	foundPoll, err := service.getPollByCodeOrID(pollCode, pollID)
	if err != nil {
		return nil, err
	}

	return foundPoll.GetResults(), nil
}

// ActivateVoting moves a poll from setup into voting.
func (service *Service) ActivateVoting(pollCode string) (*domain.Poll, error) {
	foundPoll, found := service.Store.FindPollByCode(pollCode)
	if !found {
		return nil, ErrNotFound
	}

	if foundPoll.IsVotingActive {
		return nil, fmt.Errorf("%w: voting is already active", ErrConflict)
	}

	if err := service.Store.ActivateVoting(pollCode); err != nil {
		return nil, err
	}

	updatedPoll, found := service.Store.FindPollByCode(pollCode)
	if !found {
		return nil, ErrNotFound
	}

	return updatedPoll, nil
}

// CreateMovie adds a movie while the poll is still in setup.
func (service *Service) CreateMovie(input domain.CreateMovieInput) (domain.Movie, error) {
	createdMovie := domain.CreateNewMovie(input)

	foundPoll, found, err := service.Store.FindPollByIDWithError(input.PollID)
	if err != nil {
		return domain.Movie{}, err
	}
	if !found {
		return domain.Movie{}, ErrNotFound
	}

	if foundPoll.IsVotingActive {
		return domain.Movie{}, fmt.Errorf("%w: voting has already started, movies can no longer be added", ErrInvalidInput)
	}
	if foundPoll.IsClosed || foundPoll.IsExpired() {
		return domain.Movie{}, fmt.Errorf("%w: poll is closed or expired, movies can no longer be added", ErrInvalidInput)
	}

	if err := service.Store.SaveMovie(createdMovie); err != nil {
		return domain.Movie{}, err
	}

	return createdMovie, nil
}

// ListMovies returns every stored movie.
func (service *Service) ListMovies() ([]domain.Movie, error) {
	return service.Store.GetAllMovies()
}

// CreateUser creates a voter identity.
func (service *Service) CreateUser(name string) (domain.User, error) {
	createdUser := domain.CreateNewUser(domain.CreateUserInput{Name: name})
	if err := service.Store.SaveUser(createdUser); err != nil {
		return domain.User{}, err
	}

	return createdUser, nil
}

// UpdateUserName renames a user without changing their voting identity.
func (service *Service) UpdateUserName(userID string, name string) (domain.User, error) {
	updatedUser, err := service.Store.UpdateUserName(userID, name)
	if err != nil {
		return domain.User{}, err
	}

	return updatedUser, nil
}

// ListUsers returns every user.
func (service *Service) ListUsers() ([]domain.User, error) {
	return service.Store.GetAllUsers()
}

// SubmitVote validates a vote against the poll and saves it.
func (service *Service) SubmitVote(pollCode string, pollID string, userID string, movieIDs []string) (domain.Vote, string, error) {
	foundPoll, err := service.getPollByCodeOrID(pollCode, pollID)
	if err != nil {
		return domain.Vote{}, "", err
	}

	createdVote := domain.CreateNewVote(domain.CreateVoteInput{
		PollID:   foundPoll.ID,
		UserID:   userID,
		MovieIDs: movieIDs,
	})

	if err := foundPoll.SubmitVote(createdVote); err != nil {
		return domain.Vote{}, "", fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if err := service.Store.SaveVote(createdVote); err != nil {
		return domain.Vote{}, "", err
	}

	return createdVote, foundPoll.PollCode, nil
}

// GenerateUniquePollCode creates an unused 8-digit public poll code.
func (service *Service) GenerateUniquePollCode() (string, error) {
	for {
		number, err := rand.Int(rand.Reader, big.NewInt(100000000))
		if err != nil {
			return "", err
		}

		code := fmt.Sprintf("%08d", number.Int64())

		exists, err := service.Store.PollCodeExists(code)
		if err != nil {
			return "", err
		}

		if !exists {
			return code, nil
		}
	}
}

func (service *Service) getPollByCodeOrID(pollCode string, pollID string) (*domain.Poll, error) {
	var foundPoll *domain.Poll
	var found bool

	if pollCode != "" {
		foundPoll, found = service.Store.FindPollByCode(pollCode)
	}
	if !found && pollID != "" {
		foundPoll, found = service.Store.FindPollByID(pollID)
	}

	if !found {
		return nil, ErrNotFound
	}

	return foundPoll, nil
}
