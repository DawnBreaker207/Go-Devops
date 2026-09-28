import { useSyncExternalStore } from 'react';

const subscribe = (onChange: () => void) => {
  window.addEventListener('scroll', onChange, { passive: true });
  return () => window.removeEventListener('scroll', onChange);
};

export const useScrolled = (threshold = 24): boolean =>
  useSyncExternalStore(
    subscribe,
    () => window.scrollY > threshold,
    () => false
  );

export default useScrolled;
