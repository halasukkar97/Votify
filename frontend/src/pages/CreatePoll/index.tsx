import { useState } from 'react';
import type { ChangeEvent, FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiClient } from '../../api/client';
import { LoadingIndicator } from '../../components/LoadingIndicator';
import { usePageTitle } from '../../hooks/usePageTitle';
import type { TranslationKey } from '../../i18n/useTranslations';
import type { CreatePollFormValues } from './interfaces';
import './CreatePoll.scss';

const initialFormValues: CreatePollFormValues = {
  name: '',
  maxVotesPerPerson: 3,
  endVotingOn: '',
};

function createDeadlineFromDate(date: string) {
  return date + 'T23:59:59Z';
}

interface CreatePollPageProps {
  t: (key: TranslationKey) => string;
}

// CreatePollPage owns the form and feedback for creating a new movie poll.
export function CreatePollPage({ t }: CreatePollPageProps) {
  usePageTitle('Create Poll');
  const navigate = useNavigate();
  // formValues stores the input values before they are sent to the backend.
  const [formValues, setFormValues] = useState<CreatePollFormValues>(initialFormValues);
  const [isCreating, setIsCreating] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  // handleChange keeps local state in sync with the form inputs.
  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const { name, value } = event.target;

    setFormValues((currentValues) => ({
      ...currentValues,
      [name]: name === 'maxVotesPerPerson' ? Number(value) : value,
    }));
  }

  // handleSubmit converts the date to a datetime and calls the create-poll API.
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsCreating(true);
    setErrorMessage('');

    try {
      const poll = await apiClient.createPoll({
        name: formValues.name,
        maxVotesPerPerson: formValues.maxVotesPerPerson,
        deadline: createDeadlineFromDate(formValues.endVotingOn),
      });

      if (!poll.pollCode) {
        throw new Error(t('create.missingPollCode'));
      }

      navigate('/polls/' + poll.pollCode, {
        state: { createdPollCode: poll.pollCode },
      });
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : t('create.errorFallback'));
    } finally {
      setIsCreating(false);
    }
  }

  return (
    <section className="page create-poll-page">
      <h1>{t('create.title')}</h1>
      <form className="form" onSubmit={handleSubmit}>
        <label>
          {t('create.pollName')}
          <input
            name="name"
            type="text"
            placeholder={t('create.pollNamePlaceholder')}
            value={formValues.name}
            onChange={handleChange}
            required
          />
        </label>

        <label>
          {t('create.maxVotes')}
          <input
            name="maxVotesPerPerson"
            type="number"
            min="1"
            value={formValues.maxVotesPerPerson}
            onChange={handleChange}
            required
          />
        </label>

        <label>
          {t('create.endVotingOn')}
          <input
            name="endVotingOn"
            type="date"
            value={formValues.endVotingOn}
            onChange={handleChange}
            required
          />
        </label>

        <button type="submit" disabled={isCreating}>
          {isCreating ? <LoadingIndicator compact label={t('create.loading')} /> : t('create.button')}
        </button>
      </form>

      {errorMessage ? (
        <div className="feedback error-message" role="alert">
          {errorMessage}
        </div>
      ) : null}
    </section>
  );
}
