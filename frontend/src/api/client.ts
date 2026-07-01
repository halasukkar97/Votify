// API_BASE_URL points the React app at the Go backend.
// VITE_API_BASE_URL can override it for deployed or shared environments.
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';

// request wraps fetch so every API call gets the same base URL, JSON headers, and error handling.
async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(API_BASE_URL + path, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  });

  if (!response.ok) {
    throw new Error('Request failed with status ' + response.status);
  }

  return response.json() as Promise<T>;
}

export type CreatePollPayload = {
  name: string;
  maxVotesPerPerson: number;
  deadline: string;
  pollType: string;
};

export type CreateOptionPayload = {
  title: string;
  pollId: string;
  releaseYear: number;
  description: string;
  imageUrl: string;
  posterUrl?: string;
};

export type CreateUserPayload = {
  name: string;
};

export type UpdateUserPayload = {
  name: string;
};

export type SubmitVotePayload = {
  pollCode: string;
  userId: string;
  optionIds: string[];
  movieIds?: string[];
};

export type Option = {
  id: string;
  pollId: string;
  title: string;
  releaseYear: number;
  description: string;
  imageUrl: string;
  posterUrl?: string;
};

export type ExternalOption = {
  id: number;
  title: string;
  release_date: string;
  overview: string;
  poster_path: string;
  poster_url?: string;
  posterUrl?: string;
};

export type User = {
  id: string;
  name: string;
};

export type Vote = {
  id: string;
  pollId: string;
  userId: string;
  optionIds: string[];
  movieIds?: string[];
};

export type Poll = {
  id: string;
  pollCode: string;
  name: string;
  maxVotesPerPerson: number;
  isClosed: boolean;
  isVotingActive: boolean;
  deadline: string;
  pollType: string;
  options: Option[];
  movies?: Option[];
  votes: Vote[];
};

export type PollResults = Record<string, number>;

export const apiClient = {
  // listPolls gets every poll for the History page.
  listPolls: () => request<Poll[]>('/polls'),

  // createPoll sends the Create Poll form data and returns the new poll.
  createPoll: (payload: CreatePollPayload) =>
    request<Poll>('/polls', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  // getPoll opens one poll by its public poll code.
  getPoll: (pollCode: string) => request<Poll>('/polls/' + encodeURIComponent(pollCode)),

  // activateVoting moves a poll from setup into the voting phase.
  activateVoting: (pollCode: string) =>
    request<Poll>('/polls/' + encodeURIComponent(pollCode) + '/activate-voting', {
      method: 'PATCH',
    }),

  // searchOptions asks the backend provider for suggestions that match the poll type.
  searchOptions: (pollType: string, query: string) =>
    request<ExternalOption[]>('/options/search?type=' + encodeURIComponent(pollType) + '&q=' + encodeURIComponent(query)),

  // searchMovies keeps older movie-specific callers working during the generic refactor.
  searchMovies: (query: string) =>
    request<ExternalOption[]>('/options/search?type=movie&q=' + encodeURIComponent(query)),

  // createOption adds one selected option to the current poll.
  createOption: (payload: CreateOptionPayload) =>
    request<Option>('/options', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  // createUser creates a backend voter record for the saved display name.
  createUser: (payload: CreateUserPayload) =>
    request<User>('/users', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  // updateUserName changes display text while keeping the same backend user ID.
  updateUserName: (userId: string, payload: UpdateUserPayload) =>
    request<User>('/users/' + encodeURIComponent(userId), {
      method: 'PATCH',
      body: JSON.stringify(payload),
    }),

  // submitVote sends all selected option IDs in one vote request.
  submitVote: (payload: SubmitVotePayload) =>
    request<Vote>('/votes', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  // getPollResults loads vote totals for one public poll code.
  getPollResults: (pollCode: string) =>
    request<PollResults>('/results?pollCode=' + encodeURIComponent(pollCode)),
};

export type Movie = Option;
export type ExternalMovie = ExternalOption;
export const createMovie = apiClient.createOption;
