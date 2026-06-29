import { useEffect, useMemo, useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';
import { useLocation, useParams } from 'react-router-dom';
import { apiClient } from '../../api/client';
import { LoadingIndicator } from '../../components/LoadingIndicator';
import { usePageTitle } from '../../hooks/usePageTitle';
import type { ExternalMovie, Movie, PollResults } from '../../api/client';
import type {
  LoadedPollState,
  MovieDraftValues,
  MovieSearchState,
  PollPageProps,
  PollRouteState,
  ToastState,
} from './interfaces';
import './Poll.scss';

const initialMovieDraft: MovieDraftValues = {
  title: '',
};

const initialMovieSearch: MovieSearchState = {
  suggestions: [],
  selectedMovie: null,
  isSearching: false,
  searchError: '',
  hasSearched: false,
};

const toastDurationMs = 30000;
const userNameStorageKey = 'votify:userName';
const userIDStorageKey = 'votify:userId';

function hasVotingEnded(deadline: string, isClosed: boolean) {
  return isClosed || (deadline ? new Date(deadline).getTime() < Date.now() : false);
}

function formatDate(deadline: string) {
  if (!deadline) {
    return '';
  }

  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(deadline));
}

function getReleaseYear(movie: ExternalMovie) {
  return movie.release_date ? new Date(movie.release_date).getFullYear() : 0;
}

function getExternalPosterURL(movie: ExternalMovie) {
  return movie.poster_url ?? movie.posterUrl ?? '';
}

function countVotesForMovie(movieID: string, votes: { movieIds: string[] }[]) {
  return votes.reduce((total, vote) => total + (vote.movieIds.includes(movieID) ? 1 : 0), 0);
}

function formatVoteCount(template: string, count: number) {
  return template.replace('{count}', String(count));
}

function formatSelectedCount(template: string, selected: number, max: number) {
  return template.replace('{selected}', String(selected)).replace('{max}', String(max));
}

// PollPage shows one public poll by pollCode and will become the voting workspace.
export function PollPage({ t }: PollPageProps) {
  const { pollCode = '' } = useParams();
  const location = useLocation();
  const routeState = location.state as PollRouteState | null;
  // pollState keeps the loaded poll and request status together.
  const [pollState, setPollState] = useState<LoadedPollState>({
    poll: null,
    isLoading: true,
    errorMessage: '',
  });
  const [movieDraft, setMovieDraft] = useState<MovieDraftValues>(initialMovieDraft);
  const [movieSearch, setMovieSearch] = useState<MovieSearchState>(initialMovieSearch);
  const [isAddingMovie, setIsAddingMovie] = useState(false);
  const [selectedMovieIds, setSelectedMovieIds] = useState<string[]>([]);
  const [currentUserId, setCurrentUserId] = useState(() => localStorage.getItem(userIDStorageKey) ?? '');
  const [voteLimitMessage, setVoteLimitMessage] = useState('');
  const [pollResults, setPollResults] = useState<PollResults>({});
  const [isVoteConfirmOpen, setIsVoteConfirmOpen] = useState(false);
  const [isStartVotingConfirmOpen, setIsStartVotingConfirmOpen] = useState(false);
  const [isSubmittingVote, setIsSubmittingVote] = useState(false);
  const [isActivatingVoting, setIsActivatingVoting] = useState(false);
  const [toast, setToast] = useState<ToastState | null>(
    routeState?.createdPollCode
      ? {
          id: Date.now(),
          type: 'success',
          message: t('poll.created'),
          detail: routeState.createdPollCode,
        }
      : null,
  );

  usePageTitle(pollState.poll?.name ?? 'Poll');

  const votingEnded = useMemo(
    () => hasVotingEnded(pollState.poll?.deadline ?? '', pollState.poll?.isClosed ?? false),
    [pollState.poll],
  );

  const isVotingActive = pollState.poll?.isVotingActive ?? false;

  const currentUserVote = useMemo(
    () => pollState.poll?.votes.find((vote) => vote.userId === currentUserId) ?? null,
    [currentUserId, pollState.poll],
  );

  const displayedSelectedMovieIds = currentUserVote?.movieIds ?? selectedMovieIds;
  const hasAlreadyVoted = Boolean(currentUserVote);

  const sortedMovies = useMemo(() => {
    if (!pollState.poll) {
      return [] as Movie[];
    }

    return pollState.poll.movies
      .map((movie, index) => ({
        movie,
        index,
        voteCount: pollResults[movie.id] ?? countVotesForMovie(movie.id, pollState.poll?.votes ?? []),
      }))
      .sort((left, right) => right.voteCount - left.voteCount || left.index - right.index)
      .map((entry) => entry.movie);
  }, [pollResults, pollState.poll]);

  // Toasts stay visible long enough to copy details, then close themselves.
  useEffect(() => {
    if (!toast) {
      return;
    }

    const timeoutID = window.setTimeout(() => setToast(null), toastDurationMs);
    return () => window.clearTimeout(timeoutID);
  }, [toast]);

  // The saved display name gets a backend user ID before voting.
  useEffect(() => {
    async function ensureUserID() {
      if (currentUserId) {
        return;
      }

      const savedName = localStorage.getItem(userNameStorageKey)?.trim();
      if (!savedName) {
        return;
      }

      try {
        const user = await apiClient.createUser({ name: savedName });
        localStorage.setItem(userIDStorageKey, user.id);
        setCurrentUserId(user.id);
      } catch {
        setCurrentUserId('');
      }
    }

    ensureUserID();
  }, [currentUserId]);

  // loadPoll fetches the public poll by pollCode from the route.
  useEffect(() => {
    async function loadPoll() {
      setPollState({ poll: null, isLoading: true, errorMessage: '' });

      try {
        const poll = await apiClient.getPoll(pollCode);
        const results = await apiClient.getPollResults(pollCode);
        setPollResults(results);
        setPollState({ poll, isLoading: false, errorMessage: '' });
      } catch (error) {
        setPollState({
          poll: null,
          isLoading: false,
          errorMessage: error instanceof Error ? error.message : t('poll.notFound'),
        });
      }
    }

    if (pollCode) {
      loadPoll();
    }
  }, [pollCode, t]);

  // Search TMDB shortly after the user stops typing.
  useEffect(() => {
    const query = movieDraft.title.trim();

    if (query.length < 2 || movieSearch.selectedMovie) {
      setMovieSearch((currentSearch) => ({
        ...currentSearch,
        suggestions: [],
        isSearching: false,
        searchError: '',
        hasSearched: false,
      }));
      return;
    }

    const timeoutID = window.setTimeout(async () => {
      setMovieSearch((currentSearch) => ({
        ...currentSearch,
        isSearching: true,
        searchError: '',
        hasSearched: true,
      }));

      try {
        const suggestions = await apiClient.searchMovies(query);
        setMovieSearch((currentSearch) => ({
          ...currentSearch,
          suggestions,
          isSearching: false,
          searchError: '',
          hasSearched: true,
        }));
      } catch {
        setMovieSearch((currentSearch) => ({
          ...currentSearch,
          suggestions: [],
          isSearching: false,
          searchError: t('poll.searchError'),
          hasSearched: true,
        }));
      }
    }, 350);

    return () => window.clearTimeout(timeoutID);
  }, [movieDraft.title, movieSearch.selectedMovie, t]);

  async function refreshPoll() {
    const poll = await apiClient.getPoll(pollCode);
    const results = await apiClient.getPollResults(pollCode);
    setPollResults(results);
    setPollState({ poll, isLoading: false, errorMessage: '' });
  }

  async function getOrCreateCurrentUserID() {
    if (currentUserId) {
      return currentUserId;
    }

    const savedName = localStorage.getItem(userNameStorageKey)?.trim();
    if (!savedName) {
      throw new Error(t('poll.missingVoterName'));
    }

    const user = await apiClient.createUser({ name: savedName });
    localStorage.setItem(userIDStorageKey, user.id);
    setCurrentUserId(user.id);
    return user.id;
  }

  function showToast(nextToast: Omit<ToastState, 'id'>) {
    setToast({ ...nextToast, id: Date.now() });
  }

  // handleMovieDraftChange keeps the add-movie starter form connected to state.
  function handleMovieDraftChange(event: ChangeEvent<HTMLInputElement>) {
    const { value } = event.target;

    setMovieDraft({ title: value });
    setMovieSearch((currentSearch) => ({
      ...currentSearch,
      selectedMovie: null,
    }));
  }

  function handleSelectMovie(movie: ExternalMovie) {
    setMovieDraft({ title: movie.title });
    setMovieSearch({
      suggestions: [],
      selectedMovie: movie,
      isSearching: false,
      searchError: '',
      hasSearched: true,
    });
  }

  function handleToggleMovie(movieID: string) {
    if (!isVotingActive || votingEnded || hasAlreadyVoted) {
      return;
    }

    setVoteLimitMessage('');
    setSelectedMovieIds((currentMovieIds) => {
      if (currentMovieIds.includes(movieID)) {
        return currentMovieIds.filter((selectedMovieID) => selectedMovieID !== movieID);
      }

      const maxVotes = pollState.poll?.maxVotesPerPerson ?? 0;
      if (currentMovieIds.length >= maxVotes) {
        setVoteLimitMessage(formatVoteCount(t('poll.maxVotesMessage'), maxVotes));
        return currentMovieIds;
      }

      return [...currentMovieIds, movieID];
    });
  }

  function handleOpenVoteConfirm() {
    if (selectedMovieIds.length === 0 || !isVotingActive || votingEnded || hasAlreadyVoted) {
      return;
    }

    setIsVoteConfirmOpen(true);
  }

  async function handleConfirmVote() {
    if (!pollState.poll) {
      return;
    }

    setIsSubmittingVote(true);

    try {
      const userId = await getOrCreateCurrentUserID();
      await apiClient.submitVote({
        pollCode: pollState.poll.pollCode,
        userId,
        movieIds: selectedMovieIds,
      });
      setIsVoteConfirmOpen(false);
      setSelectedMovieIds([]);
      await refreshPoll();
      showToast({ type: 'success', message: t('poll.voteSubmitSuccess') });
    } catch (error) {
      showToast({
        type: 'error',
        message: error instanceof Error ? error.message : t('poll.voteSubmitError'),
      });
    } finally {
      setIsSubmittingVote(false);
    }
  }

  function handleOpenStartVotingConfirm() {
    if (!pollState.poll || pollState.poll.isVotingActive || votingEnded) {
      return;
    }

    setIsStartVotingConfirmOpen(true);
  }

  async function handleConfirmStartVoting() {
    if (!pollState.poll) {
      return;
    }

    setIsActivatingVoting(true);

    try {
      await apiClient.activateVoting(pollState.poll.pollCode);
      setIsStartVotingConfirmOpen(false);
      setSelectedMovieIds([]);
      await refreshPoll();
      showToast({ type: 'success', message: t('poll.votingStarted') });
    } catch (error) {
      showToast({
        type: 'error',
        message: error instanceof Error ? error.message : t('poll.startVotingError'),
      });
    } finally {
      setIsActivatingVoting(false);
    }
  }

  // handleAddMovie sends the selected TMDB movie to the backend create movie endpoint.
  async function handleAddMovie(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!pollState.poll) {
      return;
    }

    if (isVotingActive) {
      showToast({ type: 'error', message: t('poll.addMovieAfterVotingStarted') });
      return;
    }

    if (!movieSearch.selectedMovie) {
      showToast({ type: 'error', message: t('poll.chooseMovieFirst') });
      return;
    }

    setIsAddingMovie(true);

    try {
      await apiClient.createMovie({
        title: movieSearch.selectedMovie.title,
        pollId: pollState.poll.id,
        releaseYear: getReleaseYear(movieSearch.selectedMovie),
        description: movieSearch.selectedMovie.overview,
        posterUrl: getExternalPosterURL(movieSearch.selectedMovie),
      });

      setMovieDraft(initialMovieDraft);
      setMovieSearch(initialMovieSearch);
      await refreshPoll();
      showToast({ type: 'success', message: t('poll.addMovieSuccess') });
    } catch (error) {
      showToast({
        type: 'error',
        message: error instanceof Error ? error.message : t('poll.addMovieError'),
      });
    } finally {
      setIsAddingMovie(false);
    }
  }

  if (pollState.isLoading) {
    return <section className="page poll-page"><LoadingIndicator label={t('poll.loading')} /></section>;
  }

  if (!pollState.poll) {
    return <section className="page poll-page"><p>{pollState.errorMessage || t('poll.notFound')}</p></section>;
  }

  const selectedMovie = movieSearch.selectedMovie;
  const shouldShowNoMoviesFound =
    movieSearch.hasSearched &&
    !movieSearch.isSearching &&
    !movieSearch.searchError &&
    movieDraft.title.trim().length >= 2 &&
    movieSearch.suggestions.length === 0 &&
    !selectedMovie;

  return (
    <section className="page poll-page">
      {toast ? (
        <div className={'toast toast--' + toast.type} role={toast.type === 'error' ? 'alert' : 'status'}>
          <button className="toast-close" type="button" onClick={() => setToast(null)} aria-label={t('toast.close')}>
            x
          </button>
          <strong>{toast.message}</strong>
          {toast.detail ? <p>{t('poll.code')}: <code>{toast.detail}</code></p> : null}
        </div>
      ) : null}

      <div className="poll-heading">
        <div className="poll-title-block">
          <h1>{pollState.poll.name}</h1>
        </div>
        <div className="poll-side-panel">
          <div className="poll-meta-card">
            <div className="poll-meta-item">
              <span>{t('poll.code')}</span>
              <strong><code>{pollState.poll.pollCode}</code></strong>
            </div>
            <div className="poll-meta-divider" />
            <div className="poll-meta-item">
              <span>{t('poll.endVotingOn')}</span>
              <strong>{formatDate(pollState.poll.deadline)}</strong>
            </div>
          </div>
          {isVotingActive ? (
            <div className="vote-submit-panel">
              <span>{formatSelectedCount(t('poll.selectedCount'), displayedSelectedMovieIds.length, pollState.poll.maxVotesPerPerson)}</span>
              <button
                type="button"
                disabled={selectedMovieIds.length === 0 || !isVotingActive || votingEnded || hasAlreadyVoted || isSubmittingVote}
                onClick={handleOpenVoteConfirm}
              >
                {t('poll.submitVotes')}
              </button>
            </div>
          ) : (
            <div className="start-voting-panel">
              <p>{t('poll.setupPhase')}</p>
              <button type="button" disabled={votingEnded || isActivatingVoting} onClick={handleOpenStartVotingConfirm}>
                {isActivatingVoting ? t('poll.startingVoting') : t('poll.startVoting')}
              </button>
            </div>
          )}
        </div>
      </div>

      {votingEnded ? (
        <div className="feedback expired-message" role="status">
          {t('poll.votingEnded')}
        </div>
      ) : null}

      {isVotingActive && !votingEnded ? (
        <div className="feedback voting-active-message" role="status">
          {t('poll.votingActive')}
        </div>
      ) : null}

      {hasAlreadyVoted ? (
        <div className="feedback already-voted-message" role="status">
          {t('poll.alreadyVoted')}
        </div>
      ) : null}

      {voteLimitMessage ? (
        <div className="feedback vote-limit-message" role="alert">
          {voteLimitMessage}
        </div>
      ) : null}

      <section className="poll-workspace">
        {!isVotingActive ? (
        <form className="form movie-search-form" onSubmit={handleAddMovie}>
          <h2>{t('poll.addMovies')}</h2>
          <label>
            {t('poll.movieTitle')}
            <input
              name="title"
              type="text"
              placeholder={t('poll.movieTitlePlaceholder')}
              value={movieDraft.title}
              onChange={handleMovieDraftChange}
              disabled={votingEnded || isVotingActive}
              autoComplete="off"
            />
          </label>

          <div className="movie-search-status" aria-live="polite">
            {movieDraft.title.trim().length < 2 && !selectedMovie ? t('poll.searchHint') : null}
            {movieSearch.isSearching ? <LoadingIndicator compact label={t('poll.searchLoading')} /> : null}
            {movieSearch.searchError ? movieSearch.searchError : null}
            {shouldShowNoMoviesFound ? t('poll.noMoviesFound') : null}
          </div>

          {movieSearch.suggestions.length > 0 ? (
            <div className="suggestions-list">
              {movieSearch.suggestions.map((movie) => (
                <button key={movie.id} type="button" onClick={() => handleSelectMovie(movie)}>
                  <span>{movie.title}</span>
                  <span>{getReleaseYear(movie) || ''}</span>
                </button>
              ))}
            </div>
          ) : null}

          {selectedMovie ? (
            <div className="selected-movie-preview">
              {getExternalPosterURL(selectedMovie) ? (
                <img src={getExternalPosterURL(selectedMovie)} alt={t('poll.posterAlt')} />
              ) : null}
              <div>
                <strong>{t('poll.selectedMovie')}</strong>
                <h3>{selectedMovie.title}</h3>
                <p>{getReleaseYear(selectedMovie) || ''}</p>
                {selectedMovie.overview ? <p>{selectedMovie.overview}</p> : null}
              </div>
            </div>
          ) : null}

          <button type="submit" disabled={votingEnded || isVotingActive || isAddingMovie}>
            {isAddingMovie ? t('poll.addingMovie') : t('poll.addMovieButton')}
          </button>
        </form>
        ) : null}

        <section className="movie-grid-section">
          <h2>{t('poll.moviesInPoll')}</h2>
          <div className="movie-grid">
            {sortedMovies.length > 0 ? (
              sortedMovies.map((movie) => {
                const poster = movie.posterUrl;
                const releaseYear = movie.releaseYear;
                const description = movie.description;
                const voteCount = pollResults[movie.id] ?? countVotesForMovie(movie.id, pollState.poll?.votes ?? []);
                const isSelected = displayedSelectedMovieIds.includes(movie.id);
                const isSelectionDisabled = !isVotingActive || votingEnded || hasAlreadyVoted;

                return (
                  <article className={isVotingActive && isSelected ? 'movie-card movie-card--selected' : 'movie-card'} key={movie.id}>
                    {isVotingActive ? (
                    <label className="movie-select-control">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        disabled={isSelectionDisabled}
                        onChange={() => handleToggleMovie(movie.id)}
                      />
                      <span aria-hidden="true">✓</span>
                    </label>
                    ) : null}
                    <div className="movie-card-poster">
                      {poster ? <img src={poster} alt={t('poll.posterAlt')} /> : <span>{t('poll.noPoster')}</span>}
                    </div>
                    <div className="movie-card-body">
                      <div className="movie-card-heading">
                        <h3>{movie.title}</h3>
                        {releaseYear ? <span>{releaseYear}</span> : null}
                      </div>
                      <span className="vote-count-badge">{formatVoteCount(t('poll.votesLabel'), voteCount)}</span>
                      {description ? <p>{description}</p> : null}
                    </div>
                  </article>
                );
              })
            ) : (
              <p>{t('poll.noMovies')}</p>
            )}
          </div>
        </section>
      </section>

      {isStartVotingConfirmOpen ? (
        <div className="vote-modal-backdrop" role="presentation">
          <div className="vote-modal" role="dialog" aria-modal="true" aria-labelledby="start-voting-confirm-title">
            <h2 id="start-voting-confirm-title">{t('poll.confirmStartVotingTitle')}</h2>
            <p>{t('poll.confirmStartVotingMessage')}</p>
            <div className="vote-modal-actions">
              <button type="button" className="secondary-button" onClick={() => setIsStartVotingConfirmOpen(false)}>
                {t('poll.cancel')}
              </button>
              <button type="button" disabled={isActivatingVoting} onClick={handleConfirmStartVoting}>
                {t('poll.confirmStartVotingButton')}
              </button>
            </div>
          </div>
        </div>
      ) : null}

      {isVoteConfirmOpen ? (
        <div className="vote-modal-backdrop" role="presentation">
          <div className="vote-modal" role="dialog" aria-modal="true" aria-labelledby="vote-confirm-title">
            <h2 id="vote-confirm-title">{t('poll.confirmVoteTitle')}</h2>
            <p>{t('poll.confirmVoteMessage')}</p>
            <div className="vote-modal-actions">
              <button type="button" className="secondary-button" onClick={() => setIsVoteConfirmOpen(false)}>
                {t('poll.cancel')}
              </button>
              <button type="button" disabled={isSubmittingVote} onClick={handleConfirmVote}>
                {t('poll.submitVotes')}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  );
}
