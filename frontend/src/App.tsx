import { useState } from 'react';
import { NavLink, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import { useTranslations } from './i18n/useTranslations';
import type { Language } from './i18n/useTranslations';
import { Breadcrumbs } from './components/Breadcrumbs';
import { CreatePollPage } from './pages/CreatePoll';
import { HistoryPage } from './pages/History';
import { HomePage } from './pages/Home';
import { JoinPollPage } from './pages/JoinPoll';
import { PollPage } from './pages/Poll';
import { PollResultsPage } from './pages/PollResults';

const USER_NAME_STORAGE_KEY = 'votify:userName';

export default function App() {
  const navigate = useNavigate();
  const location = useLocation();
  const { language, languages, setLanguage, t } = useTranslations();
  const [savedName, setSavedName] = useState(() =>
    localStorage.getItem(USER_NAME_STORAGE_KEY) ?? '',
  );
  const [draftName, setDraftName] = useState(savedName);
  const [isEditingName, setIsEditingName] = useState(savedName.length === 0);
  const [isLanguageMenuOpen, setIsLanguageMenuOpen] = useState(false);
  const isInitialNameEntry = savedName.length === 0;
  const isNameFormVisible = isInitialNameEntry || isEditingName;

  const navItems = [
    { to: '/', label: t('nav.home') },
    { to: '/polls/new', label: t('nav.createPoll') },
    { to: '/join', label: t('nav.joinPoll') },
    { to: '/history', label: t('nav.history') },
  ];

  // saveName writes the current name to localStorage for future visits.
  function saveName(nextName: string) {
    localStorage.setItem(USER_NAME_STORAGE_KEY, nextName);
    setSavedName(nextName);
    setDraftName(nextName);
    setIsEditingName(false);
  }

  // startEditingName sends the user back home so the focused name form is visible.
  function startEditingName() {
    setDraftName(savedName);
    setIsEditingName(true);
    navigate('/');
  }

  // cancelNameEdit restores the saved value without replacing the current user name.
  function cancelNameEdit() {
    setDraftName(savedName);
    setIsEditingName(false);
  }

  // chooseLanguage updates the app text and closes the small language menu.
  function chooseLanguage(nextLanguage: Language) {
    setLanguage(nextLanguage);
    setIsLanguageMenuOpen(false);
  }

  // isNavItemActive keeps header highlighting aligned with nested app routes.
  function isNavItemActive(path: string, isActive: boolean) {
    if (path === '/history') {
      return location.pathname === '/history' ||
        (location.pathname.startsWith('/polls/') && location.pathname !== '/polls/new') ||
        location.pathname === '/results';
    }

    return isActive;
  }

  return (
    <div className={isInitialNameEntry ? 'app-shell app-shell--name-entry' : 'app-shell'}>
      <header className="site-header">
        <a className="brand" href="/">
          {t('app.brand')}
        </a>

        <div className="header-menu">
          {!isInitialNameEntry ? (
            <nav aria-label="Main navigation">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) => (isNavItemActive(item.to, isActive) ? 'active' : undefined)}
                >
                  {item.label}
                </NavLink>
              ))}
              <button className="header-name-button" type="button" onClick={startEditingName}>
                {savedName}
              </button>
            </nav>
          ) : null}

          <div className="language-switcher">
            <button
              type="button"
              className="language-button"
              aria-expanded={isLanguageMenuOpen}
              aria-haspopup="menu"
              aria-label={t('language.label')}
              onClick={() => setIsLanguageMenuOpen((isOpen) => !isOpen)}
            >
              <span>{language.toUpperCase()}</span>
              <span className="language-chevron" aria-hidden="true">▾</span>
            </button>

            {isLanguageMenuOpen ? (
              <div className="language-menu" role="menu">
                {languages.map((availableLanguage) => (
                  <button
                    key={availableLanguage}
                    type="button"
                    role="menuitem"
                    onClick={() => chooseLanguage(availableLanguage)}
                  >
                    {availableLanguage.toUpperCase()}
                  </button>
                ))}
              </div>
            ) : null}
          </div>
        </div>
      </header>

      <main>
        {!isInitialNameEntry ? <Breadcrumbs t={t} /> : null}
        <Routes>
          <Route
            path="/"
            element={
              <HomePage
                draftName={draftName}
                isEditingName={isNameFormVisible}
                isInitialNameEntry={isInitialNameEntry}
                onCancelNameEdit={cancelNameEdit}
                onDraftNameChange={setDraftName}
                onSaveName={saveName}
                t={t}
              />
            }
          />
          <Route path="/polls/new" element={<CreatePollPage t={t} />} />
          <Route path="/join" element={<JoinPollPage savedName={savedName} t={t} />} />
          <Route path="/history" element={<HistoryPage t={t} />} />
          <Route path="/polls/:pollCode" element={<PollPage t={t} />} />
          <Route path="/results" element={<PollResultsPage t={t} />} />
        </Routes>
      </main>
    </div>
  );
}
