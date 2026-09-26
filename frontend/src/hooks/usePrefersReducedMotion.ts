import { useSyncExternalStore } from 'react';

const QUERY = '(prefers-reduced-motion: reduce)';

const subscribe = (onChange: () => void) => {
  const mq = window.matchMedia(QUERY);
  mq.addEventListener('change', onChange);
  return () => mq.removeEventListener('change', onChange);
};

/** True when the viewer has asked their system for reduced motion.
 *
 *  `.claude/rules/motion.md` says autoplay must not run under reduced motion, and that anything the
 *  CSS tokens cannot express should branch on this. Its suggested `useReducedMotion()` comes from
 *  `motion/react`, which is NOT installed here (the owner has not approved that dependency), so this
 *  reads the same media query directly. */
export const usePrefersReducedMotion = (): boolean =>
  useSyncExternalStore(
    subscribe,
    () => window.matchMedia(QUERY).matches,
    () => false
  );

export default usePrefersReducedMotion;
