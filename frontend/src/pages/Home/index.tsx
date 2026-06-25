import { Link } from 'react-router-dom';
import { usePageTitle } from '../../hooks/usePageTitle';
import type { HomePageProps, NameFormSubmitEvent } from './interfaces';
import './Home.scss';

// HomePage is the starting point for Votify and stores the user's display name.
export function HomePage({
  draftName,
  isEditingName,
  isInitialNameEntry,
  onCancelNameEdit,
  onDraftNameChange,
  onSaveName,
  t,
}: HomePageProps) {
  usePageTitle('Home');
  // saveName sends the trimmed name up to App so it can be shared in the header.
  function saveName(event: NameFormSubmitEvent) {
    event.preventDefault();

    const trimmedName = draftName.trim();
    if (!trimmedName) {
      return;
    }

    onSaveName(trimmedName);
  }

  if (isEditingName) {
    return (
      <section className={isInitialNameEntry ? 'name-entry-page name-entry-page--initial' : 'name-entry-page'}>
        <form className="form name-form" onSubmit={saveName}>
          <h1>{t(isInitialNameEntry ? 'name.enter' : 'name.update')}</h1>
          <label>
            {t(isInitialNameEntry ? 'name.enter' : 'name.current')}
            <input
              name="name"
              type="text"
              placeholder={t('name.placeholder')}
              value={draftName}
              onChange={(event) => onDraftNameChange(event.target.value)}
              autoFocus
            />
          </label>
          <div className="name-form-actions">
            {!isInitialNameEntry ? (
              <button type="button" className="secondary-button" onClick={onCancelNameEdit}>
                {t('name.cancel')}
              </button>
            ) : null}
            <button type="submit">{t(isInitialNameEntry ? 'name.button' : 'name.updateButton')}</button>
          </div>
        </form>
      </section>
    );
  }

  return (
    <section className="page home-page">
      <h1>{t('home.title')}</h1>
      <p>{t('home.description')}</p>

      <div className="actions">
        <Link to="/polls/new">{t('home.createPoll')}</Link>
        <Link to="/join">{t('home.joinPoll')}</Link>
      </div>
    </section>
  );
}
