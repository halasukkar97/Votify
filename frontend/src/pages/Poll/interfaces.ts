import type { ExternalMovie, Poll } from '../../api/client';
import type { TranslationKey } from '../../i18n/useTranslations';

export interface PollPageProps {
  t: (key: TranslationKey) => string;
}

export interface PollRouteState {
  createdPollCode?: string;
}

export interface MovieDraftValues {
  title: string;
}

export interface ManualOptionValues {
  title: string;
  releaseYear: string;
  description: string;
  imageUrl: string;
  coverPreviewUrl: string;
  uploadError: string;
  author: string;
  isbn: string;
}

export interface MovieSearchState {
  suggestions: ExternalMovie[];
  selectedMovie: ExternalMovie | null;
  isSearching: boolean;
  searchError: string;
  hasSearched: boolean;
}

export interface ToastState {
  id: number;
  type: 'success' | 'error';
  message: string;
  detail?: string;
}

export interface LoadedPollState {
  poll: Poll | null;
  isLoading: boolean;
  errorMessage: string;
}
