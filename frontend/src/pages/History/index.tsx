import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../../api/client';
import { LoadingIndicator } from '../../components/LoadingIndicator';
import { usePageTitle } from '../../hooks/usePageTitle';
import type { HistoryPageProps, HistoryState } from './interfaces';
import './History.scss';

function formatDate(deadline: string) {
  if (!deadline) {
    return '';
  }

  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(deadline));
}

function getPollStatus(isClosed: boolean, deadline: string, t: HistoryPageProps['t']) {
  if (isClosed || (deadline && new Date(deadline).getTime() < Date.now())) {
    return t('history.closed');
  }

  return t('history.active');
}

// HistoryPage will show the user's previous polls once the backend supports it.
export function HistoryPage({ t }: HistoryPageProps) {
  usePageTitle('History');
  const [historyState, setHistoryState] = useState<HistoryState>({
    polls: [],
    isLoading: true,
    errorMessage: '',
  });

  // loadPolls reads every created poll from the backend history endpoint.
  useEffect(() => {
    async function loadPolls() {
      setHistoryState({ polls: [], isLoading: true, errorMessage: '' });

      try {
        const polls = await apiClient.listPolls();
        setHistoryState({ polls, isLoading: false, errorMessage: '' });
      } catch {
        setHistoryState({ polls: [], isLoading: false, errorMessage: t('history.error') });
      }
    }

    loadPolls();
  }, [t]);

  return (
    <section className="page history-page">
      <h1>{t('history.title')}</h1>

      {historyState.isLoading ? <LoadingIndicator label={t('history.loading')} /> : null}
      {historyState.errorMessage ? <p className="history-error">{historyState.errorMessage}</p> : null}

      {!historyState.isLoading && !historyState.errorMessage && historyState.polls.length === 0 ? (
        <p>{t('history.empty')}</p>
      ) : null}

      {historyState.polls.length > 0 ? (
        <div className="history-table-card">
          <table className="history-table">
            <thead>
              <tr>
                <th>{t('history.pollName')}</th>
                <th>{t('history.pollCode')}</th>
                <th>{t('history.status')}</th>
                <th>{t('history.endVotingOn')}</th>
                <th>{t('history.optionsCount')}</th>
                <th>{t('history.action')}</th>
              </tr>
            </thead>
            <tbody>
              {historyState.polls.map((poll) => {
                const status = getPollStatus(poll.isClosed, poll.deadline, t);

                return (
                  <tr key={poll.id}>
                    <td data-label={t('history.pollName')}>{poll.name}</td>
                    <td data-label={t('history.pollCode')}>
                      {poll.pollCode ? <code>{poll.pollCode}</code> : t('history.missingPollCode')}
                    </td>
                    <td data-label={t('history.status')}>
                      <span className={status === t('history.active') ? 'status-badge active' : 'status-badge closed'}>
                        {status}
                      </span>
                    </td>
                    <td data-label={t('history.endVotingOn')}>{formatDate(poll.deadline)}</td>
                    <td data-label={t('history.optionsCount')}>{(poll.options ?? poll.movies ?? []).length}</td>
                    <td data-label={t('history.action')}>
                      {poll.pollCode ? (
                        <Link className="history-action" to={'/polls/' + poll.pollCode}>
                          {t('history.openPoll')}
                        </Link>
                      ) : (
                        <span>{t('history.missingPollCode')}</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}
