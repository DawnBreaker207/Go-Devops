import { useSyncExternalStore } from 'react';

const subscribe = (onChange: () => void) => {
  window.addEventListener('scroll', onChange, { passive: true });
  return () => window.removeEventListener('scroll', onChange);
};

/** True once the page has scrolled past `threshold` pixels.
 *
 *  `useSyncExternalStore` rather than a `useState` + `useEffect` pair: the scroll position is external
 *  state, and this is the API React provides for reading it without a setState inside an effect
 *  (which the repo's eslint config rejects). The server snapshot is `false` because nothing has
 *  scrolled before hydration. */
export const useScrolled = (threshold = 24): boolean =>
  useSyncExternalStore(
    subscribe,
    () => window.scrollY > threshold,
    () => false
  );

export default useScrolled;
