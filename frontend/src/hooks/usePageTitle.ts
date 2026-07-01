import { useEffect } from 'react';

const appTitle = 'Voting App';

// usePageTitle keeps the browser tab title in sync with the current screen.
export function usePageTitle(pageTitle: string) {
  useEffect(() => {
    document.title = pageTitle ? appTitle + ' | ' + pageTitle : appTitle;
  }, [pageTitle]);
}
